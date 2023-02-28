package resolvable

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/decode"
	"github.com/graph-gophers/graphql-go/directives"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
)

const (
	Query        = "Query"
	Mutation     = "Mutation"
	Subscription = "Subscription"
)

type Schema struct {
	*Meta
	ast.Schema
	Query                Resolvable
	Mutation             Resolvable
	Subscription         Resolvable
	QueryResolver        reflect.Value
	MutationResolver     reflect.Value
	SubscriptionResolver reflect.Value
}

type Resolvable interface {
	isResolvable()
}

type Object struct {
	Name           string
	Fields         map[string]*Field
	TypeAssertions map[string]*TypeAssertion
	Interfaces     map[string]struct{}
}

type Field struct {
	ast.FieldDefinition
	TypeName    string
	MethodIndex int
	FieldIndex  []int
	HasContext  bool
	HasError    bool
	ArgsPacker  *packer.StructPacker
	Visitors    *FieldVisitors
	ValueExec   Resolvable
	TraceLabel  string
}

type FieldVisitors struct {
	Interceptors []directives.ResolverInterceptor
	Validators   []directives.Validator
}

func (f *Field) UseMethodResolver() bool {
	return len(f.FieldIndex) == 0
}

func (f *Field) Resolve(ctx context.Context, resolver reflect.Value, args interface{}) (output interface{}, err error) {
	// Short circuit case to avoid wrapping functions
	v := f.Visitors.Interceptors
	if len(v) == 0 {
		return f.resolve(ctx, resolver, args)
	}

	wrapResolver := func(ctx context.Context, args interface{}) (output interface{}, err error) {
		return f.resolve(ctx, resolver, args)
	}

	for _, d := range v {
		d := d // Needed to avoid passing only the last directive, since we're closing over this loop var pointer
		innerResolver := wrapResolver

		wrapResolver = func(ctx context.Context, args interface{}) (output interface{}, err error) {
			return d.Resolve(ctx, args, resolverFunc(innerResolver))
		}
	}

	return wrapResolver(ctx, args)
}

func (f *Field) Validate(ctx context.Context, args interface{}) []error {
	d := f.Visitors
	if d == nil {
		// Meta schema fields don't include directives on those fields
		return nil
	}

	var errs []error

	for _, v := range d.Validators {
		if err := v.Validate(ctx, args); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (f *Field) resolve(ctx context.Context, resolver reflect.Value, args interface{}) (output interface{}, err error) {
	if !f.UseMethodResolver() {
		res := resolver

		// TODO extract out unwrapping ptr logic to a common place
		if res.Kind() == reflect.Ptr {
			res = res.Elem()
		}

		return res.FieldByIndex(f.FieldIndex).Interface(), nil
	}

	var in []reflect.Value
	var callOut []reflect.Value

	if f.HasContext {
		in = append(in, reflect.ValueOf(ctx))
	}

	if f.ArgsPacker != nil {
		in = append(in, reflect.ValueOf(args))
	}

	callOut = resolver.Method(f.MethodIndex).Call(in)
	result := callOut[0]

	if f.HasError && !callOut[1].IsNil() {
		resolverErr := callOut[1].Interface().(error)
		return result.Interface(), resolverErr
	}

	return result.Interface(), nil
}

type resolverFunc func(ctx context.Context, args interface{}) (output interface{}, err error)

func (f resolverFunc) Resolve(ctx context.Context, args interface{}) (output interface{}, err error) {
	return f(ctx, args)
}

type TypeAssertion struct {
	MethodIndex int
	TypeExec    Resolvable
}

type List struct {
	Elem Resolvable
}

type Scalar struct{}

func (*Object) isResolvable() {}
func (*List) isResolvable()   {}
func (*Scalar) isResolvable() {}

func ApplyResolver(s *ast.Schema, resolver interface{}, dirs []directives.Directive, useFieldResolvers bool) (*Schema, error) {
	if resolver == nil {
		return &Schema{Meta: newMeta(s), Schema: *s}, nil
	}

	ds, err := applyDirectives(s, dirs)
	if err != nil {
		return nil, err
	}

	directivePackers, err := buildDirectivePackers(s, ds)
	if err != nil {
		return nil, err
	}

	b := newBuilder(s, directivePackers, useFieldResolvers)

	var query, mutation, subscription Resolvable

	resolvers := map[string]interface{}{}

	rv := reflect.ValueOf(resolver)
	// use separate resolvers in case Query, Mutation and/or Subscription methods are defined
	for _, op := range [...]string{Query, Mutation, Subscription} {
		m := rv.MethodByName(op)
		if m.IsValid() { // if the root resolver has a method for the current operation
			mt := m.Type()
			if mt.NumIn() != 0 {
				return nil, fmt.Errorf("method %q of %v must not accept any arguments, got %d", op, rv.Type(), mt.NumIn())
			}
			if mt.NumOut() != 1 {
				return nil, fmt.Errorf("method %q of %v must have 1 return value, got %d", op, rv.Type(), mt.NumOut())
			}
			ot := mt.Out(0)
			if ot.Kind() != reflect.Pointer && ot.Kind() != reflect.Interface {
				return nil, fmt.Errorf("method %q of %v must return an interface or a pointer, got %+v", op, rv.Type(), ot)
			}
			out := m.Call(nil)
			res := out[0]
			if res.IsNil() {
				return nil, fmt.Errorf("method %q of %v must return a non-nil result, got %v", op, rv.Type(), res)
			}
			switch res.Kind() {
			case reflect.Pointer:
				resolvers[op] = res.Elem().Addr().Interface()
			case reflect.Interface:
				resolvers[op] = res.Elem().Interface()
			default:
				panic("ureachable")
			}
		}
		// If a method for the current operation is not defined in the root resolver,
		// then use the root resolver for the operation.
		if resolvers[op] == nil {
			resolvers[op] = resolver
		}
	}

	if t, ok := s.RootOperationTypes["query"]; ok {
		if err := b.assignExec(&query, t, reflect.TypeOf(resolvers[Query])); err != nil {
			return nil, err
		}
	}

	if t, ok := s.RootOperationTypes["mutation"]; ok {
		if err := b.assignExec(&mutation, t, reflect.TypeOf(resolvers[Mutation])); err != nil {
			return nil, err
		}
	}

	if t, ok := s.RootOperationTypes["subscription"]; ok {
		if err := b.assignExec(&subscription, t, reflect.TypeOf(resolvers[Subscription])); err != nil {
			return nil, err
		}
	}

	if err := b.finish(); err != nil {
		return nil, err
	}

	return &Schema{
		Meta:                 newMeta(s),
		Schema:               *s,
		QueryResolver:        reflect.ValueOf(resolvers[Query]),
		MutationResolver:     reflect.ValueOf(resolvers[Mutation]),
		SubscriptionResolver: reflect.ValueOf(resolvers[Subscription]),
		Query:                query,
		Mutation:             mutation,
		Subscription:         subscription,
	}, nil
}

func buildDirectivePackers(s *ast.Schema, visitors map[string]directives.Directive) (map[string]*packer.StructPacker, error) {
	// Directive packers need to use a dedicated builder which is ready ('finish()' called) while
	// schema fields (and their argument packers) are still being built
	builder := packer.NewBuilder()

	packers := map[string]*packer.StructPacker{}
	for _, d := range s.Directives {
		n := d.Name

		v, ok := visitors[n]
		if !ok {
			// Directives which need visitors have already been checked
			// Anything without a visitor now is a built-in directive without a packer.
			continue
		}

		switch v.(type) {
		case directives.ResolverInterceptor, directives.Validator:
			// Accepted directive type
		default:
			// Directive doesn't apply at field resolution time, skip it
			continue
		}

		r := reflect.TypeOf(v)

		p, err := builder.MakeStructPacker(d.Arguments, r)
		if err != nil {
			return nil, err
		}

		packers[n] = p
	}

	if err := builder.Finish(); err != nil {
		return nil, err
	}

	return packers, nil
}

func applyDirectives(s *ast.Schema, visitors []directives.Directive) (map[string]directives.Directive, error) {
	byName := make(map[string]directives.Directive, len(s.Directives))

	for _, v := range visitors {
		name := v.ImplementsDirective()

		if existing, ok := byName[name]; ok {
			return nil, fmt.Errorf("multiple implementations registered for directive %q. Implementation types %T and %T", name, existing, v)
		}

		// At least 1 of the optional directive functions must be defined for each directive.
		switch v.(type) {
		case directives.ResolverInterceptor, directives.Validator:
			byName[name] = v
		default:
			return nil, fmt.Errorf("directive %q (implemented by %T) does not implement a valid directive visitor function", name, v)
		}
	}

	return byName, nil
}

type execBuilder struct {
	schema            *ast.Schema
	resMap            map[typePair]*resMapEntry
	directivePackers  map[string]*packer.StructPacker
	packerBuilder     *packer.Builder
	useFieldResolvers bool
}

type typePair struct {
	graphQLType  ast.Type
	resolverType reflect.Type
}

type resMapEntry struct {
	exec    Resolvable
	targets []*Resolvable
}

func newBuilder(s *ast.Schema, directives map[string]*packer.StructPacker, useFieldResolvers bool) *execBuilder {
	return &execBuilder{
		schema:            s,
		resMap:            make(map[typePair]*resMapEntry),
		directivePackers:  directives,
		packerBuilder:     packer.NewBuilder(),
		useFieldResolvers: useFieldResolvers,
	}
}

func (b *execBuilder) finish() error {
	for _, entry := range b.resMap {
		for _, target := range entry.targets {
			*target = entry.exec
		}
	}

	return b.packerBuilder.Finish()
}

func (b *execBuilder) assignExec(target *Resolvable, t ast.Type, resolverType reflect.Type) error {
	k := typePair{t, resolverType}
	ref, ok := b.resMap[k]
	if !ok {
		ref = &resMapEntry{}
		b.resMap[k] = ref
		var err error
		ref.exec, err = b.makeExec(t, resolverType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makeExec(t ast.Type, resolverType reflect.Type) (Resolvable, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	switch t := t.(type) {
	case *ast.ObjectTypeDefinition:
		return b.makeObjectExec(t.Name, t.Fields, nil, t.Interfaces, nonNull, resolverType)

	case *ast.InterfaceTypeDefinition:
		return b.makeObjectExec(t.Name, t.Fields, t.PossibleTypes, nil, nonNull, resolverType)

	case *ast.Union:
		return b.makeObjectExec(t.Name, nil, t.UnionMemberTypes, nil, nonNull, resolverType)
	}

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", resolverType)
		}
		resolverType = resolverType.Elem()
	}

	switch t := t.(type) {
	case *ast.ScalarTypeDefinition:
		return makeScalarExec(t, resolverType)

	case *ast.EnumTypeDefinition:
		return &Scalar{}, nil

	case *ast.List:
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e := &List{}
		if err := b.assignExec(&e.Elem, t.OfType, resolverType.Elem()); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type: " + t.String())
	}
}

func makeScalarExec(t *ast.ScalarTypeDefinition, resolverType reflect.Type) (Resolvable, error) {
	implementsType := false
	switch r := reflect.New(resolverType).Interface().(type) {
	case *int32:
		implementsType = t.Name == "Int"
	case *float64:
		implementsType = t.Name == "Float"
	case *string:
		implementsType = t.Name == "String"
	case *bool:
		implementsType = t.Name == "Boolean"
	case decode.Unmarshaler:
		implementsType = r.ImplementsGraphQLType(t.Name)
	}

	if !implementsType {
		return nil, fmt.Errorf("can not use %s as %s", resolverType, t.Name)
	}
	return &Scalar{}, nil
}

func (b *execBuilder) makeObjectExec(typeName string, fields ast.FieldsDefinition, possibleTypes []*ast.ObjectTypeDefinition,
	interfaces []*ast.InterfaceTypeDefinition, nonNull bool, resolverType reflect.Type) (*Object, error) {
	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	methodHasReceiver := resolverType.Kind() != reflect.Interface

	Fields := make(map[string]*Field)
	rt := unwrapPtr(resolverType)
	fieldsCount, fieldTagsCount := fieldCount(rt, map[string]int{}, map[string]int{})
	for _, f := range fields {
		var fieldIndex []int
		methodIndex := findMethod(resolverType, f.Name)
		if b.useFieldResolvers && methodIndex == -1 {
			// If a resolver field is ambiguous thrown an error unless there is exactly one field with the given graphql
			// reflect tag. In that case use the field with the reflect tag.
			if fieldTagsCount[f.Name] > 1 {
				return nil, fmt.Errorf("%s does not resolve %q: multiple fields have a graphql reflect tag %q", resolverType, typeName, f.Name)
			} else if fieldsCount[strings.ToLower(stripUnderscore(f.Name))] > 1 && fieldTagsCount[f.Name] != 1 {
				return nil, fmt.Errorf("%s does not resolve %q: ambiguous field %q", resolverType, typeName, f.Name)
			}
			fieldIndex = findField(rt, f.Name, []int{}, fieldTagsCount)
		}
		if methodIndex == -1 && len(fieldIndex) == 0 {
			var hint string
			if findMethod(reflect.PtrTo(resolverType), f.Name) != -1 {
				hint = " (hint: the method exists on the pointer type)"
			}
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q%s", resolverType, typeName, f.Name, hint)
		}

		var m reflect.Method
		var sf reflect.StructField
		if methodIndex != -1 {
			m = resolverType.Method(methodIndex)
		} else {
			sf = rt.FieldByIndex(fieldIndex)
		}
		fe, err := b.makeFieldExec(typeName, f, m, sf, methodIndex, fieldIndex, methodHasReceiver)
		if err != nil {
			var resolverName string
			if methodIndex != -1 {
				resolverName = m.Name
			} else {
				resolverName = sf.Name
			}
			return nil, fmt.Errorf("%s\n\tused by (%s).%s", err, resolverType, resolverName)
		}
		Fields[f.Name] = fe
	}

	// Check type assertions when
	//	1) using method resolvers
	//	2) Or resolver is not an interface type
	typeAssertions := make(map[string]*TypeAssertion)
	if !b.useFieldResolvers || resolverType.Kind() != reflect.Interface {
		for _, impl := range possibleTypes {
			methodIndex := findMethod(resolverType, "To"+impl.Name)
			if methodIndex == -1 {
				return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "To"+impl.Name, impl.Name)
			}
			m := resolverType.Method(methodIndex)
			expectedIn := 0
			if methodHasReceiver {
				expectedIn = 1
			}
			if m.Type.NumIn() != expectedIn {
				return nil, fmt.Errorf("%s does not resolve %q: method %q should't have any arguments", resolverType, typeName, "To"+impl.Name)
			}
			if m.Type.NumOut() != 2 {
				return nil, fmt.Errorf("%s does not resolve %q: method %q should return a value and a bool indicating success", resolverType, typeName, "To"+impl.Name)
			}
			a := &TypeAssertion{
				MethodIndex: methodIndex,
			}
			if err := b.assignExec(&a.TypeExec, impl, resolverType.Method(methodIndex).Type.Out(0)); err != nil {
				return nil, err
			}
			typeAssertions[impl.Name] = a
		}
	}

	ifaces := make(map[string]struct{})
	for _, iface := range interfaces {
		ifaces[iface.Name] = struct{}{}
	}

	return &Object{
		Name:           typeName,
		Fields:         Fields,
		TypeAssertions: typeAssertions,
		Interfaces:     ifaces,
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *execBuilder) makeFieldExec(typeName string, f *ast.FieldDefinition, m reflect.Method, sf reflect.StructField, methodIndex int, fieldIndex []int, methodHasReceiver bool) (*Field, error) {
	var argsPacker *packer.StructPacker
	var hasError bool
	var hasContext bool

	// Validate resolver method only when there is one
	if methodIndex != -1 {
		in := make([]reflect.Type, m.Type.NumIn())
		for i := range in {
			in[i] = m.Type.In(i)
		}
		if methodHasReceiver {
			in = in[1:] // first parameter is receiver
		}

		hasContext = len(in) > 0 && in[0] == contextType
		if hasContext {
			in = in[1:]
		}

		if len(f.Arguments) > 0 {
			if len(in) == 0 {
				return nil, fmt.Errorf("must have `args struct { ... }` argument for field arguments")
			}
			var err error
			argsPacker, err = b.packerBuilder.MakeStructPacker(f.Arguments, in[0])
			if err != nil {
				return nil, err
			}
			in = in[1:]
		}

		if len(in) > 0 {
			return nil, fmt.Errorf("too many arguments")
		}

		maxNumOfReturns := 2
		if m.Type.NumOut() < maxNumOfReturns-1 {
			return nil, fmt.Errorf("too few return values")
		}

		if m.Type.NumOut() > maxNumOfReturns {
			return nil, fmt.Errorf("too many return values")
		}

		hasError = m.Type.NumOut() == maxNumOfReturns
		if hasError {
			if m.Type.Out(maxNumOfReturns-1) != errorType {
				return nil, fmt.Errorf(`must have "error" as its last return value`)
			}
		}
	}

	visitors, err := packDirectives(f.Directives, b.directivePackers)
	if err != nil {
		return nil, err
	}

	fe := &Field{
		FieldDefinition: *f,
		TypeName:        typeName,
		MethodIndex:     methodIndex,
		FieldIndex:      fieldIndex,
		HasContext:      hasContext,
		ArgsPacker:      argsPacker,
		Visitors:        visitors,
		HasError:        hasError,
		TraceLabel:      fmt.Sprintf("GraphQL field: %s.%s", typeName, f.Name),
	}

	var out reflect.Type
	if methodIndex != -1 {
		out = m.Type.Out(0)
		sub, ok := b.schema.RootOperationTypes["subscription"]
		if ok && typeName == sub.TypeName() && out.Kind() == reflect.Chan {
			out = m.Type.Out(0).Elem()
		}
	} else {
		out = sf.Type
	}
	if err := b.assignExec(&fe.ValueExec, f.Type, out); err != nil {
		return nil, err
	}

	return fe, nil
}

func packDirectives(ds ast.DirectiveList, packers map[string]*packer.StructPacker) (*FieldVisitors, error) {
	var resolvers []directives.ResolverInterceptor
	var validators []directives.Validator

	for _, d := range ds {
		dp, ok := packers[d.Name.Name]
		if !ok {
			continue // skip directives without packers
		}

		args := make(map[string]interface{})
		for _, arg := range d.Arguments {
			args[arg.Name.Name] = arg.Value.Deserialize(nil)
		}

		p, err := dp.Pack(args)
		if err != nil {
			return nil, err
		}

		v := p.Interface()

		// Visitors can implement any of these types optionally, and may implement multiple
		if v, ok := v.(directives.ResolverInterceptor); ok {
			resolvers = append(resolvers, v)
		}

		if v, ok := v.(directives.Validator); ok {
			validators = append(validators, v)
		}
	}

	return &FieldVisitors{Interceptors: resolvers, Validators: validators}, nil
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(stripUnderscore(name), stripUnderscore(t.Method(i).Name)) {
			return i
		}
	}
	return -1
}

func findField(t reflect.Type, name string, index []int, matchingTagsCount map[string]int) []int {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			newIndex := findField(field.Type, name, []int{i}, matchingTagsCount)
			if len(newIndex) > 1 {
				return append(index, newIndex...)
			}
		}

		if gt, ok := field.Tag.Lookup("graphql"); ok {
			if name == gt {
				return append(index, i)
			}
		}

		// The current field's tag didn't match, however, if the tag of another field matches,
		// then skip the name matching until we find the desired field with the correct tag.
		if matchingTagsCount[name] > 0 {
			continue
		}

		if strings.EqualFold(stripUnderscore(name), stripUnderscore(field.Name)) {
			return append(index, i)
		}
	}

	return index
}

// fieldCount helps resolve ambiguity when more than one embedded struct contains fields with the same name.
// or when a field has a `graphql` reflect tag with the same name as some other field causing name collision.
func fieldCount(t reflect.Type, count, tagsCount map[string]int) (map[string]int, map[string]int) {
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		var fieldName, gt string
		var hasTag bool
		if gt, hasTag = field.Tag.Lookup("graphql"); hasTag && gt != "" {
			fieldName = gt
		} else {
			fieldName = strings.ToLower(stripUnderscore(field.Name))
		}

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			count, tagsCount = fieldCount(field.Type, count, tagsCount)
		} else {
			if _, ok := count[fieldName]; !ok {
				count[fieldName] = 0
			}
			count[fieldName]++
			if !hasTag {
				continue
			}
			if _, ok := count[gt]; !ok {
				tagsCount[gt] = 0
			}
			tagsCount[gt]++
		}
	}

	return count, tagsCount
}

func unwrapNonNull(t ast.Type) (ast.Type, bool) {
	if nn, ok := t.(*ast.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}

func stripUnderscore(s string) string {
	return strings.Replace(s, "_", "", -1)
}

func unwrapPtr(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}
