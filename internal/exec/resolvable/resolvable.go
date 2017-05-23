package resolvable

import (
	"context"
	"fmt"
	"reflect"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/packer"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	schema.Schema
	Query    Resolvable
	Mutation Resolvable
	Resolver reflect.Value
}

type Resolvable interface { // TODO rename? remove?
	isResolvable()
}

type Object struct {
	Type   schema.NamedType
	Fields map[string]*Field
}

type Field struct {
	schema.Field
	TypeName   string
	Resolver   *Resolver
	ValueExec  Resolvable
	TraceLabel string
}

type Interface struct {
	Options []*InterfaceOption
}

type InterfaceOption struct {
	common.TypePair
	Exec Resolvable
}

type List struct {
	Elem Resolvable
}

type Scalar struct{}

func (*Object) isResolvable()    {}
func (*Interface) isResolvable() {}
func (*List) isResolvable()      {}
func (*Scalar) isResolvable()    {}

type Resolver struct {
	Select     func(args map[string]interface{}) (SelectedResolver, bool, error)
	ResultType reflect.Type
}

type TypeToResolversMap map[common.TypePair]FieldToResolverMap
type FieldToResolverMap map[string]*Resolver

type SelectedResolver func(ctx context.Context, parent reflect.Value) (reflect.Value, error)

func ApplyResolvers(s *schema.Schema, resolvers TypeToResolversMap, root interface{}) (*Schema, error) {
	b := newBuilder(s, resolvers)

	var query, mutation Resolvable

	if t, ok := s.EntryPoints["query"]; ok {
		if err := b.assignExec(&query, t, reflect.TypeOf(root)); err != nil {
			return nil, err
		}
	}

	if t, ok := s.EntryPoints["mutation"]; ok {
		if err := b.assignExec(&mutation, t, reflect.TypeOf(root)); err != nil {
			return nil, err
		}
	}

	if err := b.finish(); err != nil {
		return nil, err
	}

	return &Schema{
		Schema:   *s,
		Resolver: reflect.ValueOf(root),
		Query:    query,
		Mutation: mutation,
	}, nil
}

type execBuilder struct {
	schema    *schema.Schema
	resolvers TypeToResolversMap
	resMap    map[common.TypePair]*resMapEntry
}

type resMapEntry struct {
	exec    Resolvable
	targets []*Resolvable
}

func newBuilder(s *schema.Schema, resolvers TypeToResolversMap) *execBuilder {
	return &execBuilder{
		schema:    s,
		resolvers: resolvers,
		resMap:    make(map[common.TypePair]*resMapEntry),
	}
}

func (b *execBuilder) finish() error {
	for _, entry := range b.resMap {
		for _, target := range entry.targets {
			*target = entry.exec
		}
	}

	return nil
}

func (b *execBuilder) assignExec(target *Resolvable, t common.Type, valueType reflect.Type) error {
	k := common.TypePair{GraphQLType: t, GoType: valueType}
	ref, ok := b.resMap[k]
	if !ok {
		ref = &resMapEntry{}
		b.resMap[k] = ref
		var err error
		ref.exec, err = b.makeExec(t, valueType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makeExec(t common.Type, valueType reflect.Type) (Resolvable, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	switch t := t.(type) {
	case *schema.Object:
		return b.makeObjectExec(t, valueType, nonNull)

	case *schema.Interface:
		return b.makeInterfaceExec(t.PossibleTypes)

	case *schema.Union:
		return b.makeInterfaceExec(t.PossibleTypes)
	}

	if !nonNull {
		if valueType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", valueType)
		}
		valueType = valueType.Elem()
	}

	switch t := t.(type) {
	case *schema.Scalar:
		return makeScalarExec(t, valueType)

	case *schema.Enum:
		return &Scalar{}, nil

	case *common.List:
		if valueType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", valueType)
		}
		e := &List{}
		if err := b.assignExec(&e.Elem, t.OfType, valueType.Elem()); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type")
	}
}

func makeScalarExec(t *schema.Scalar, valueType reflect.Type) (Resolvable, error) {
	implementsType := false
	switch r := reflect.New(valueType).Interface().(type) {
	case *int32:
		implementsType = (t.Name == "Int")
	case *float64:
		implementsType = (t.Name == "Float")
	case *string:
		implementsType = (t.Name == "String")
	case *bool:
		implementsType = (t.Name == "Boolean")
	case packer.Unmarshaler:
		implementsType = r.ImplementsGraphQLType(t.Name)
	}
	if !implementsType {
		return nil, fmt.Errorf("can not use %s as %s", valueType, t.Name)
	}
	return &Scalar{}, nil
}

func (b *execBuilder) makeObjectExec(t *schema.Object, valueType reflect.Type, nonNull bool) (*Object, error) {
	if !nonNull {
		if valueType.Kind() != reflect.Ptr && valueType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", valueType)
		}
	}

	fieldResolvers, ok := b.resolvers[common.TypePair{GraphQLType: t, GoType: valueType}]
	if !ok {
		return nil, fmt.Errorf("no resolvers declared for %s used as %q", valueType, t)
	}

	resFields := make(map[string]*Field)
	for _, f := range t.Fields {
		r, ok := fieldResolvers[f.Name]
		if !ok {
			return nil, fmt.Errorf("missing resolver %q for %s used as %q", f.Name, valueType, t)
		}
		fe := &Field{
			Field:      *f,
			TypeName:   t.String(),
			Resolver:   r,
			TraceLabel: fmt.Sprintf("GraphQL field: %s.%s", t, f.Name),
		}
		if err := b.assignExec(&fe.ValueExec, f.Type, r.ResultType); err != nil {
			return nil, err
		}
		resFields[f.Name] = fe
	}

	return &Object{
		Type:   t,
		Fields: resFields,
	}, nil
}

func (b *execBuilder) makeInterfaceExec(possibleTypes []*schema.Object) (*Interface, error) {
	iface := &Interface{}
	for _, pt := range possibleTypes {
		for tp := range b.resolvers {
			if tp.GraphQLType == pt {
				opt := &InterfaceOption{TypePair: tp}
				if err := b.assignExec(&opt.Exec, tp.GraphQLType, tp.GoType); err != nil {
					return nil, err
				}
				iface.Options = append(iface.Options, opt)
			}
		}
	}
	return iface, nil
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
