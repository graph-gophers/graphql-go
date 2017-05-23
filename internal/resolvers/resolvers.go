package resolvers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/packer"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/schema"
)

type Builder struct {
	Schema        *schema.Schema
	ResolverMap   resolvable.TypeToResolversMap
	PackerBuilder *packer.Builder
}

func NewBuilder(s *schema.Schema) *Builder {
	return &Builder{
		Schema:        s,
		ResolverMap:   make(resolvable.TypeToResolversMap),
		PackerBuilder: packer.NewBuilder(),
	}
}

func (b *Builder) Resolvers(graphqlType string, goType interface{}, fieldMap map[string]interface{}) {
	st, ok := b.Schema.Types[graphqlType]
	if !ok {
		panic("type not found") // TODO
	}

	t := common.TypePair{GraphQLType: st, GoType: reflect.TypeOf(goType)}
	m := make(resolvable.FieldToResolverMap)
	for field, resolver := range fieldMap {
		f := fields(t.GraphQLType).Get(field)
		if f == nil {
			panic("field not found") // TODO
		}

		var utRes UntypedResolver
		switch r := resolver.(type) {
		case string:
			utRes = newResolverField(r)
		default:
			utRes = newResolverFunc(r, b.PackerBuilder)
		}

		res, err := utRes(t.GoType, f.Args)
		if err != nil {
			panic(err) // TODO
		}

		m[field] = res
	}
	b.ResolverMap[t] = m
}

type UntypedResolver func(valueType reflect.Type, args common.InputValueList) (*resolvable.Resolver, error)

func newResolverField(goField string) UntypedResolver {
	return func(valueType reflect.Type, args common.InputValueList) (*resolvable.Resolver, error) {
		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		sf, ok := valueType.FieldByName(goField)
		if !ok {
			return nil, fmt.Errorf("type %s has no field %q", valueType, goField)
		}
		if sf.PkgPath != "" {
			return nil, fmt.Errorf("field %q must be exported", sf.Name)
		}

		return &resolvable.Resolver{
			Select: func(args map[string]interface{}) (resolvable.SelectedResolver, bool, error) {
				return func(ctx context.Context, parent reflect.Value) (reflect.Value, error) {
					if parent.Kind() == reflect.Ptr {
						parent = parent.Elem()
					}
					return parent.FieldByIndex(sf.Index), nil
				}, false, nil
			},
			ResultType: sf.Type,
		}, nil
	}
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func newResolverFunc(fn interface{}, packerBuilder *packer.Builder) UntypedResolver {
	fnVal := reflect.ValueOf(fn)
	return func(valueType reflect.Type, args common.InputValueList) (*resolvable.Resolver, error) {
		in := make([]reflect.Type, fnVal.Type().NumIn())
		for i := range in {
			in[i] = fnVal.Type().In(i)
		}
		in = in[1:] // first parameter is value

		hasContext := len(in) > 0 && in[0] == contextType
		if hasContext {
			in = in[1:]
		}

		var argsPacker *packer.StructPacker
		if len(args) > 0 {
			if len(in) == 0 {
				return nil, fmt.Errorf("must have parameter for field arguments")
			}
			var err error
			argsPacker, err = packerBuilder.MakeStructPacker(args, in[0])
			if err != nil {
				return nil, err
			}
			in = in[1:]
		}

		if len(in) > 0 {
			return nil, fmt.Errorf("too many parameters")
		}

		if fnVal.Type().NumOut() > 2 {
			return nil, fmt.Errorf("too many return values")
		}

		hasError := fnVal.Type().NumOut() == 2
		if hasError {
			if fnVal.Type().Out(1) != errorType {
				return nil, fmt.Errorf(`must have "error" as its second return value`)
			}
		}

		return &resolvable.Resolver{
			Select: func(args map[string]interface{}) (resolvable.SelectedResolver, bool, error) {
				var packedArgs reflect.Value
				if argsPacker != nil {
					var err error
					packedArgs, err = argsPacker.Pack(args)
					if err != nil {
						return nil, false, err
					}
				}

				async := hasContext || argsPacker != nil || hasError
				return func(ctx context.Context, parent reflect.Value) (reflect.Value, error) {
					in := []reflect.Value{parent}
					if hasContext {
						in = append(in, reflect.ValueOf(ctx))
					}
					if argsPacker != nil {
						in = append(in, packedArgs)
					}
					callOut := fnVal.Call(in)
					if hasError && !callOut[1].IsNil() {
						return reflect.Value{}, callOut[1].Interface().(error)
					}
					return callOut[0], nil
				}, async, nil
			},
			ResultType: fnVal.Type().Out(0),
		}, nil
	}
}

func fields(t common.Type) schema.FieldList {
	switch t := t.(type) {
	case *schema.Object:
		return t.Fields
	case *schema.Interface:
		return t.Fields
	default:
		return nil
	}
}
