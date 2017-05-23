package selected

import (
	"reflect"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/packer"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
)

type Request struct {
	Schema *schema.Schema
	Doc    *query.Document
	Vars   map[string]interface{}
	Mu     sync.Mutex
	Errs   []*errors.QueryError
}

func (r *Request) AddError(err *errors.QueryError) {
	r.Mu.Lock()
	r.Errs = append(r.Errs, err)
	r.Mu.Unlock()
}

func ApplyOperation(r *Request, s *resolvable.Schema, op *query.Operation) []*Field {
	var obj *resolvable.Object
	switch op.Type {
	case query.Query:
		obj = s.Query.(*resolvable.Object)
	case query.Mutation:
		obj = s.Mutation.(*resolvable.Object)
	}
	return applySelectionSet(r, obj, obj.Type, op.Selections)
}

type Field struct {
	resolvable.Field
	Alias       string
	Args        map[string]interface{}
	Resolver    resolvable.SelectedResolver
	SubSel      SubSelection
	Async       bool
	FixedResult reflect.Value
}

type SubSelection interface {
	Sels(t reflect.Type) []*Field
}

type concreteSubSelection []*Field

func (s concreteSubSelection) Sels(t reflect.Type) []*Field {
	return s
}

type interfaceSubSelection []interfaceSubSelectionOption

type interfaceSubSelectionOption struct {
	common.TypePair
	SubSel SubSelection
}

func (s interfaceSubSelection) Sels(t reflect.Type) []*Field {
	for _, opt := range s {
		if opt.GoType == t {
			return opt.SubSel.Sels(t)
		}
	}
	panic("invalid type") // TODO
}

func applySelectionSet(r *Request, e *resolvable.Object, t schema.NamedType, sels []query.Selection) (flattenedSels concreteSubSelection) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *query.Field:
			field := sel
			if skipByDirective(r, field.Directives) {
				continue
			}

			switch field.Name.Name {
			case "__typename":
				flattenedSels = append(flattenedSels, &Field{
					Field:       resolvable.MetaFieldTypename,
					Alias:       field.Alias.Name,
					FixedResult: reflect.ValueOf(t.TypeName()),
				})

			case "__schema":
				flattenedSels = append(flattenedSels, &Field{
					Field:       resolvable.MetaFieldSchema,
					Alias:       field.Alias.Name,
					SubSel:      applySelectionSet(r, resolvable.MetaSchema, resolvable.MetaSchema.Type, field.Selections),
					Async:       true,
					FixedResult: reflect.ValueOf(introspection.WrapSchema(r.Schema)),
				})

			case "__type":
				p := packer.ValuePacker{ValueType: reflect.TypeOf("")}
				v, err := p.Pack(field.Arguments.MustGet("name").Value(r.Vars))
				if err != nil {
					r.AddError(errors.Errorf("%s", err))
					return nil
				}

				t, ok := r.Schema.Types[v.String()]
				if !ok {
					return nil
				}

				flattenedSels = append(flattenedSels, &Field{
					Field:       resolvable.MetaFieldType,
					Alias:       field.Alias.Name,
					SubSel:      applySelectionSet(r, resolvable.MetaType, resolvable.MetaType.Type, field.Selections),
					Async:       true,
					FixedResult: reflect.ValueOf(introspection.WrapType(t)),
				})

			default:
				fe := e.Fields[field.Name.Name]

				var args map[string]interface{}
				if len(field.Arguments) > 0 {
					args = make(map[string]interface{})
					for _, arg := range field.Arguments {
						args[arg.Name.Name] = arg.Value.Value(r.Vars)
					}
				}

				res, async, err := fe.Resolver.Select(args)
				if err != nil {
					r.AddError(errors.Errorf("%s", err))
					return
				}

				fieldSels := applyField(r, fe.ValueExec, field.Selections)
				flattenedSels = append(flattenedSels, &Field{
					Field:    *fe,
					Alias:    field.Alias.Name,
					Args:     args,
					Resolver: res,
					SubSel:   fieldSels,
					Async:    async || hasAsyncSel(fieldSels),
				})
			}

		case *query.InlineFragment:
			frag := sel
			if skipByDirective(r, frag.Directives) {
				continue
			}
			flattenedSels = append(flattenedSels, applyFragment(r, e, t, &frag.Fragment)...)

		case *query.FragmentSpread:
			spread := sel
			if skipByDirective(r, spread.Directives) {
				continue
			}
			flattenedSels = append(flattenedSels, applyFragment(r, e, t, &r.Doc.Fragments.Get(spread.Name.Name).Fragment)...)

		default:
			panic("invalid type")
		}
	}
	return
}

func applyFragment(r *Request, e *resolvable.Object, t schema.NamedType, frag *query.Fragment) []*Field {
	if frag.On.Name != "" {
		for _, on := range possibleTypes(r.Schema.Types[frag.On.Name]) {
			if on == t {
				return applySelectionSet(r, e, t, frag.Selections)
			}
		}
		return nil
	}
	return applySelectionSet(r, e, t, frag.Selections)
}

func possibleTypes(t common.Type) []*schema.Object {
	switch t := t.(type) {
	case *schema.Object:
		return []*schema.Object{t}
	case *schema.Interface:
		return t.PossibleTypes
	case *schema.Union:
		return t.PossibleTypes
	default:
		panic("unreachable")
	}
}

func applyField(r *Request, e resolvable.Resolvable, sels []query.Selection) SubSelection {
	switch e := e.(type) {
	case *resolvable.Object:
		return applySelectionSet(r, e, e.Type, sels)
	case *resolvable.Interface:
		var s interfaceSubSelection
		for _, opt := range e.Options {
			s = append(s, interfaceSubSelectionOption{
				TypePair: opt.TypePair,
				SubSel:   applySelectionSet(r, opt.Exec.(*resolvable.Object), opt.GraphQLType.(schema.NamedType), sels),
			})
		}
		return s
	case *resolvable.List:
		return applyField(r, e.Elem, sels)
	case *resolvable.Scalar:
		return nil
	default:
		panic("unreachable")
	}
}

func skipByDirective(r *Request, directives common.DirectiveList) bool {
	if d := directives.Get("skip"); d != nil {
		p := packer.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(d.Args.MustGet("if").Value(r.Vars))
		if err != nil {
			r.AddError(errors.Errorf("%s", err))
		}
		if err == nil && v.Bool() {
			return true
		}
	}

	if d := directives.Get("include"); d != nil {
		p := packer.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(d.Args.MustGet("if").Value(r.Vars))
		if err != nil {
			r.AddError(errors.Errorf("%s", err))
		}
		if err == nil && !v.Bool() {
			return true
		}
	}

	return false
}

func HasAsyncSel(subSels []SubSelection) bool {
	for _, s := range subSels {
		if hasAsyncSel(s) {
			return true
		}
	}
	return false
}

func HasAsyncSel1(sels []*Field) bool {
	return hasAsyncSel(concreteSubSelection(sels))
}

func hasAsyncSel(subSel SubSelection) bool {
	switch subSel := subSel.(type) {
	case concreteSubSelection:
		for _, f := range subSel {
			if f.Async {
				return true
			}
		}
		return false
	case interfaceSubSelection:
		for _, opt := range subSel {
			if hasAsyncSel(opt.SubSel) {
				return true
			}
		}
		return false
	case nil:
		return false
	default:
		panic("unreachable")
	}
}
