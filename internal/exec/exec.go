package exec

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

// keep in sync with main package
const OpenTracingTagType = "graphql.type"
const OpenTracingTagField = "graphql.field"
const OpenTracingTagTrivial = "graphql.trivial"
const OpenTracingTagArgsPrefix = "graphql.args."
const OpenTracingTagError = "graphql.error"

type contextKey string
type FieldList map[string]FieldList

const ResolverFields contextKey = "Resolver_Fields"

type Exec struct {
	queryExec    iExec
	mutationExec iExec
	schema       *schema.Schema
	resolver     reflect.Value
}

func Make(s *schema.Schema, resolver interface{}) (*Exec, error) {
	b := newExecBuilder(s)

	var queryExec, mutationExec iExec

	if t, ok := s.EntryPoints["query"]; ok {
		if err := b.assignExec(&queryExec, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if t, ok := s.EntryPoints["mutation"]; ok {
		if err := b.assignExec(&mutationExec, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if err := b.finish(); err != nil {
		return nil, err
	}

	return &Exec{
		schema:       s,
		resolver:     reflect.ValueOf(resolver),
		queryExec:    queryExec,
		mutationExec: mutationExec,
	}, nil
}

type execBuilder struct {
	schema        *schema.Schema
	execMap       map[typePair]*execMapEntry
	packerMap     map[typePair]*packerMapEntry
	structPackers []*structPacker
}

type typePair struct {
	graphQLType  common.Type
	resolverType reflect.Type
}

type execMapEntry struct {
	exec    iExec
	targets []*iExec
}

type packerMapEntry struct {
	packer  packer
	targets []*packer
}

func newExecBuilder(s *schema.Schema) *execBuilder {
	return &execBuilder{
		schema:    s,
		execMap:   make(map[typePair]*execMapEntry),
		packerMap: make(map[typePair]*packerMapEntry),
	}
}

func (b *execBuilder) finish() error {
	for _, entry := range b.execMap {
		for _, target := range entry.targets {
			*target = entry.exec
		}
	}

	for _, entry := range b.packerMap {
		for _, target := range entry.targets {
			*target = entry.packer
		}
	}

	for _, p := range b.structPackers {
		p.defaultStruct = reflect.New(p.structType).Elem()
		for _, f := range p.fields {
			if f.field.Default != nil {
				v, err := f.fieldPacker.pack(nil, f.field.Default)
				if err != nil {
					return err
				}
				p.defaultStruct.FieldByIndex(f.fieldIndex).Set(v)
			}
		}
	}

	return nil
}

func (b *execBuilder) assignExec(target *iExec, t common.Type, resolverType reflect.Type) error {
	k := typePair{t, resolverType}
	ref, ok := b.execMap[k]
	if !ok {
		ref = &execMapEntry{}
		b.execMap[k] = ref
		var err error
		ref.exec, err = b.makeExec(t, resolverType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makeExec(t common.Type, resolverType reflect.Type) (iExec, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	switch t := t.(type) {
	case *schema.Object:
		return b.makeObjectExec(t.Name, t.Fields, nil, nonNull, resolverType)

	case *schema.Interface:
		return b.makeObjectExec(t.Name, t.Fields, t.PossibleTypes, nonNull, resolverType)

	case *schema.Union:
		return b.makeObjectExec(t.Name, nil, t.PossibleTypes, nonNull, resolverType)
	}

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", resolverType)
		}
		resolverType = resolverType.Elem()
	}

	switch t := t.(type) {
	case *schema.Scalar:
		return makeScalarExec(t, resolverType)

	case *schema.Enum:
		return &scalarExec{}, nil

	case *common.List:
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e := &listExec{nonNull: nonNull}
		if err := b.assignExec(&e.elem, t.OfType, resolverType.Elem()); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type")
	}
}

func makeScalarExec(t *schema.Scalar, resolverType reflect.Type) (iExec, error) {
	implementsType := false
	switch r := reflect.New(resolverType).Interface().(type) {
	case *int32:
		implementsType = (t.Name == "Int")
	case *float64:
		implementsType = (t.Name == "Float")
	case *string:
		implementsType = (t.Name == "String")
	case *bool:
		implementsType = (t.Name == "Boolean")
	case Unmarshaler:
		implementsType = r.ImplementsGraphQLType(t.Name)
	}
	if !implementsType {
		return nil, fmt.Errorf("can not use %s as %s", resolverType, t.Name)
	}
	return &scalarExec{}, nil
}

func (b *execBuilder) makeObjectExec(typeName string, fields map[string]*schema.Field, possibleTypes []*schema.Object, nonNull bool, resolverType reflect.Type) (*objectExec, error) {
	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	methodHasReceiver := resolverType.Kind() != reflect.Interface
	fieldExecs := make(map[string]*fieldExec)
	for name, f := range fields {
		methodIndex := findMethod(resolverType, name)
		if methodIndex == -1 {
			hint := ""
			if findMethod(reflect.PtrTo(resolverType), name) != -1 {
				hint = " (hint: the method exists on the pointer type)"
			}
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q%s", resolverType, typeName, name, hint)
		}

		m := resolverType.Method(methodIndex)
		fe, err := b.makeFieldExec(typeName, f, m, methodIndex, methodHasReceiver)
		if err != nil {
			return nil, fmt.Errorf("%s\n\treturned by (%s).%s", err, resolverType, m.Name)
		}
		fieldExecs[name] = fe
	}

	typeAssertions := make(map[string]*typeAssertExec)
	for _, impl := range possibleTypes {
		methodIndex := findMethod(resolverType, "to"+impl.Name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "to"+impl.Name, impl.Name)
		}
		a := &typeAssertExec{
			methodIndex: methodIndex,
		}
		if err := b.assignExec(&a.typeExec, impl, resolverType.Method(methodIndex).Type.Out(0)); err != nil {
			return nil, err
		}
		typeAssertions[impl.Name] = a
	}

	return &objectExec{
		name:           typeName,
		fields:         fieldExecs,
		typeAssertions: typeAssertions,
		nonNull:        nonNull,
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *execBuilder) makeFieldExec(typeName string, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool) (*fieldExec, error) {
	in := make([]reflect.Type, m.Type.NumIn())
	for i := range in {
		in[i] = m.Type.In(i)
	}
	if methodHasReceiver {
		in = in[1:] // first parameter is receiver
	}

	hasContext := len(in) > 0 && in[0] == contextType
	if hasContext {
		in = in[1:]
	}

	var argsPacker *structPacker
	if len(f.Args.Fields) > 0 {
		if len(in) == 0 {
			return nil, fmt.Errorf("must have parameter for field arguments")
		}
		var err error
		argsPacker, err = b.makeStructPacker(&f.Args, in[0])
		if err != nil {
			return nil, err
		}
		in = in[1:]
	}

	if len(in) > 0 {
		return nil, fmt.Errorf("too many parameters")
	}

	if m.Type.NumOut() > 2 {
		return nil, fmt.Errorf("too many return values")
	}

	hasError := m.Type.NumOut() == 2
	if hasError {
		if m.Type.Out(1) != errorType {
			return nil, fmt.Errorf(`must have "error" as its second return value`)
		}
	}

	fe := &fieldExec{
		typeName:    typeName,
		field:       f,
		methodIndex: methodIndex,
		hasContext:  hasContext,
		argsPacker:  argsPacker,
		hasError:    hasError,
	}
	if err := b.assignExec(&fe.valueExec, f.Type, m.Type.Out(0)); err != nil {
		return nil, err
	}
	return fe, nil
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(name, t.Method(i).Name) {
			return i
		}
	}
	return -1
}

type semaphore chan struct{}

type request struct {
	doc     *query.Document
	vars    map[string]interface{}
	schema  *schema.Schema
	limiter semaphore
	mu      sync.Mutex
	errs    []*errors.QueryError
}

func (r *request) addError(err *errors.QueryError) {
	r.mu.Lock()
	r.errs = append(r.errs, err)
	r.mu.Unlock()
}

func (r *request) handlePanic() {
	if err := recover(); err != nil {
		execErr := errors.Errorf("graphql: panic occurred: %v", err)
		r.addError(execErr)

		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Printf("%s\n%s", execErr, buf)
	}
}

func (r *request) resolveVar(value interface{}) interface{} {
	if v, ok := value.(common.Variable); ok {
		value = r.vars[string(v)]
	}
	return value
}

func ExecuteRequest(ctx context.Context, e *Exec, document *query.Document, operationName string, variables map[string]interface{}, maxParallelism int) (interface{}, []*errors.QueryError) {
	op, err := getOperation(document, operationName)
	if err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	r := &request{
		doc:     document,
		vars:    variables,
		schema:  e.schema,
		limiter: make(semaphore, maxParallelism),
	}

	var opExec iExec
	var serially bool
	switch op.Type {
	case query.Query:
		opExec = e.queryExec
		serially = false
	case query.Mutation:
		opExec = e.mutationExec
		serially = true
	}

	data := func() interface{} {
		defer r.handlePanic()
		return opExec.exec(ctx, r, op.SelSet, e.resolver, serially)
	}()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return data, r.errs
}

func getOperation(document *query.Document, operationName string) (*query.Operation, error) {
	if len(document.Operations) == 0 {
		return nil, fmt.Errorf("no operations in query document")
	}

	if operationName == "" {
		if len(document.Operations) > 1 {
			return nil, fmt.Errorf("more than one operation in query document and no operation name given")
		}
		for _, op := range document.Operations {
			return op, nil // return the one and only operation
		}
	}

	op, ok := document.Operations[operationName]
	if !ok {
		return nil, fmt.Errorf("no operation with name %q", operationName)
	}
	return op, nil
}

type iExec interface {
	exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{}
}

type scalarExec struct{}

func (e *scalarExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	return resolver.Interface()
}

type listExec struct {
	elem    iExec
	nonNull bool
}

func (e *listExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	if !e.nonNull {
		if resolver.IsNil() {
			return nil
		}
		resolver = resolver.Elem()
	}
	l := make([]interface{}, resolver.Len())
	var wg sync.WaitGroup
	for i := range l {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer r.handlePanic()
			l[i] = e.elem.exec(ctx, r, selSet, resolver.Index(i), false)
		}(i)
	}
	wg.Wait()
	return l
}

type objectExec struct {
	name           string
	fields         map[string]*fieldExec
	typeAssertions map[string]*typeAssertExec
	nonNull        bool
}

type addResultFn func(key string, value interface{})

func (e *objectExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	if resolver.IsNil() {
		if e.nonNull {
			r.addError(errors.Errorf("got nil for non-null %q", e.name))
		}
		return nil
	}
	var mu sync.Mutex
	results := make(map[string]interface{})
	addResult := func(key string, value interface{}) {
		mu.Lock()
		results[key] = value
		mu.Unlock()
	}
	e.execSelectionSet(ctx, r, selSet, resolver, addResult, serially)
	return results
}

func (e *objectExec) execSelectionSet(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, addResult addResultFn, serially bool) {
	var wg sync.WaitGroup
	for _, sel := range selSet.Selections {
		execSel := func(f func()) {
			if serially {
				defer r.handlePanic()
				f()
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer r.handlePanic()
				f()
			}()
		}

		switch sel := sel.(type) {
		case *query.Field:
			if skipByDirective(r, sel.Directives) {
				continue
			}

			namedFields := resolveFieldNames(make(FieldList), sel.SelSet, r.doc)
			execSel(func() {
				e.execField(context.WithValue(ctx, ResolverFields, namedFields), r, sel, resolver, addResult)
			})

		case *query.FragmentSpread:
			if skipByDirective(r, sel.Directives) {
				continue
			}

			fs := sel
			execSel(func() {
				frag, ok := r.doc.Fragments[fs.Name]
				if !ok {
					panic(fmt.Errorf("fragment %q not found", fs.Name)) // TODO proper error handling
				}
				e.execFragment(ctx, r, &frag.Fragment, resolver, addResult)
			})

		case *query.InlineFragment:
			if skipByDirective(r, sel.Directives) {
				continue
			}

			frag := sel
			execSel(func() {
				e.execFragment(ctx, r, &frag.Fragment, resolver, addResult)
			})

		default:
			panic("invalid type")
		}
	}
	wg.Wait()
}

func resolveFieldNames(namedFields FieldList, selSet *query.SelectionSet, doc *query.Document) FieldList {
	if selSet != nil {
		for _, selectionField := range selSet.Selections {
			switch selectionField := selectionField.(type) {
			case *query.Field:
				namedFields[selectionField.Name] = make(FieldList)
				resolveFieldNames(namedFields[selectionField.Name], selectionField.SelSet, doc)
			case *query.FragmentSpread:
				frag, ok := doc.Fragments[selectionField.Name]
				if !ok {
					panic(fmt.Errorf("fragment %q not found", selectionField.Name)) // TODO proper error handling
				}
				resolveFieldNames(namedFields, frag.SelSet, doc)
			case *query.InlineFragment:
				resolveFieldNames(namedFields, selectionField.SelSet, doc)
			default:
				panic("invalid type")
			}
		}
	}
	return namedFields
}

func (e *objectExec) execField(ctx context.Context, r *request, f *query.Field, resolver reflect.Value, addResult addResultFn) {
	switch f.Name {
	case "__typename":
		if len(e.typeAssertions) == 0 {
			addResult(f.Alias, e.name)
			return
		}

		for name, a := range e.typeAssertions {
			out, err := callWithLimiter(ctx, resolver.Method(a.methodIndex), nil, r)
			if err != nil {
				return
			}
			if out[1].Bool() {
				addResult(f.Alias, name)
				return
			}
		}

	case "__schema":
		addResult(f.Alias, introspectSchema(ctx, r, f.SelSet))

	case "__type":
		p := valuePacker{valueType: stringType}
		v, err := p.pack(r, r.resolveVar(f.Arguments["name"]))
		if err != nil {
			r.addError(errors.Errorf("%s", err))
			addResult(f.Alias, nil)
			return
		}
		addResult(f.Alias, introspectType(ctx, r, v.String(), f.SelSet))

	default:
		fe, ok := e.fields[f.Name]
		if !ok {
			panic(fmt.Errorf("%q has no field %q", e.name, f.Name)) // TODO proper error handling
		}

		span, spanCtx := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("GraphQL field: %s.%s", fe.typeName, fe.field.Name))
		defer span.Finish()
		span.SetTag(OpenTracingTagType, fe.typeName)
		span.SetTag(OpenTracingTagField, fe.field.Name)
		if !fe.hasContext && fe.argsPacker == nil && !fe.hasError {
			span.SetTag(OpenTracingTagTrivial, true)
		}

		result, err := fe.exec(spanCtx, r, f, resolver, span)

		if err != nil {
			queryError := errors.Errorf("%s", err)
			queryError.ResolverError = err
			r.addError(queryError)
			addResult(f.Alias, nil) // TODO handle non-nil

			ext.Error.Set(span, true)
			span.SetTag(OpenTracingTagError, err)
			return
		}

		addResult(f.Alias, result)
	}
}

func (e *objectExec) execFragment(ctx context.Context, r *request, frag *query.Fragment, resolver reflect.Value, addResult addResultFn) {
	if frag.On != "" && frag.On != e.name {
		a, ok := e.typeAssertions[frag.On]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.name)) // TODO proper error handling
		}
		out, err := callWithLimiter(ctx, resolver.Method(a.methodIndex), nil, r)
		if err != nil {
			return
		}
		if !out[1].Bool() {
			return
		}
		a.typeExec.(*objectExec).execSelectionSet(ctx, r, frag.SelSet, out[0], addResult, false)
		return
	}
	e.execSelectionSet(ctx, r, frag.SelSet, resolver, addResult, false)
}

type fieldExec struct {
	typeName    string
	field       *schema.Field
	methodIndex int
	hasContext  bool
	argsPacker  *structPacker
	hasError    bool
	valueExec   iExec
}

func (e *fieldExec) exec(ctx context.Context, r *request, f *query.Field, resolver reflect.Value, span opentracing.Span) (interface{}, error) {
	var in []reflect.Value

	if e.hasContext {
		in = append(in, reflect.ValueOf(ctx))
	}

	if e.argsPacker != nil {
		for name, arg := range f.Arguments {
			span.SetTag(OpenTracingTagArgsPrefix+name, arg)
		}
		packed, err := e.argsPacker.pack(r, f.Arguments)
		if err != nil {
			return nil, err
		}
		in = append(in, packed)
	}

	m := resolver.Method(e.methodIndex)
	out, err := callWithLimiter(ctx, m, in, r)
	if err != nil {
		return nil, err
	}
	if e.hasError && !out[1].IsNil() {
		return nil, out[1].Interface().(error)
	}

	return e.valueExec.exec(ctx, r, f.SelSet, out[0], false), nil
}

func callWithLimiter(ctx context.Context, fn reflect.Value, in []reflect.Value, r *request) ([]reflect.Value, error) {
	r.limiter <- struct{}{}
	defer func() { <-r.limiter }()

	if err := ctx.Err(); err != nil {
		return nil, err // don't execute any more resolvers if context got cancelled
	}

	return fn.Call(in), nil
}

type typeAssertExec struct {
	methodIndex int
	typeExec    iExec
}

func skipByDirective(r *request, d map[string]common.DirectiveArgs) bool {
	if args, ok := d["skip"]; ok {
		p := valuePacker{valueType: boolType}
		v, err := p.pack(r, r.resolveVar(args["if"]))
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && v.Bool() {
			return true
		}
	}

	if args, ok := d["include"]; ok {
		p := valuePacker{valueType: boolType}
		v, err := p.pack(r, r.resolveVar(args["if"]))
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && !v.Bool() {
			return true
		}
	}

	return false
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
