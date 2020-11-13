package selected

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/selected"
)

type Request struct {
	Schema               *schema.Schema
	Doc                  *query.Document
	Vars                 map[string]interface{}
	Mu                   sync.Mutex
	Errs                 []*errors.QueryError
	DisableIntrospection bool
}

func (r *Request) AddError(err *errors.QueryError) {
	r.Mu.Lock()
	r.Errs = append(r.Errs, err)
	r.Mu.Unlock()
}

func ApplyOperation(r *Request, s *resolvable.Schema, op *query.Operation) []Selection {
	var obj *resolvable.Object
	switch op.Type {
	case query.Query:
		obj = s.Query.(*resolvable.Object)
	case query.Mutation:
		obj = s.Mutation.(*resolvable.Object)
	case query.Subscription:
		obj = s.Subscription.(*resolvable.Object)
	}
	return applySelectionSet(r, s, obj, op.Selections)
}

type Selection interface {
	isSelection()
	ToSelection() selected.Selection
}

func toSelections(sels []Selection) (out []selected.Selection) {
	if len(sels) == 0 {
		return
	}

	out = make([]selected.Selection, len(sels))
	for i, sel := range sels {
		out[i] = sel.ToSelection()
	}
	return
}

type SchemaField struct {
	resolvable.Field
	Alias       string
	Args        map[string]interface{}
	PackedArgs  reflect.Value
	Sels        []Selection
	Async       bool
	FixedResult reflect.Value
}

func (f *SchemaField) Kind() selected.Kind {
	return selected.FieldKind
}

func (f *SchemaField) Identifier() string {
	return f.Name
}

func (f *SchemaField) Aliased() string {
	return f.Alias
}

func (f *SchemaField) Children() (out []selected.Selection) {
	return toSelections(f.Sels)
}

func (f *SchemaField) ToSelection() selected.Selection {
	return selected.Selection(f)
}

type TypeAssertion struct {
	resolvable.TypeAssertion
	Sels []Selection
}

func (f *TypeAssertion) Kind() selected.Kind {
	return selected.TypeAssertionKind
}

func (f *TypeAssertion) Type() string {
	var toType func(resolvable.Resolvable) string
	toType = func(r resolvable.Resolvable) string {
		if f.TypeExec == nil {
			return ""
		}

		switch v := f.TypeExec.(type) {
		case *resolvable.Scalar:
			return "scalar"
		case *resolvable.List:
			return toType(v.Elem)
		case *resolvable.Object:
			return v.Name
		default:
			return "<unknown>"
		}
	}

	return toType(f.TypeExec)
}

func (f *TypeAssertion) Children() (out []selected.Selection) {
	return toSelections(f.Sels)
}

func (f *TypeAssertion) ToSelection() selected.Selection {
	return selected.Selection(f)
}

type TypenameField struct {
	resolvable.Object
	Alias string
}

func (f *TypenameField) Kind() selected.Kind {
	return selected.TypenameFieldKind
}

func (f *TypenameField) Aliased() string {
	return f.Alias
}

func (f *TypenameField) Type() string {
	return f.Name
}

func (f *TypenameField) ToSelection() selected.Selection {
	return selected.Selection(f)
}

func (*SchemaField) isSelection()   {}
func (*TypeAssertion) isSelection() {}
func (*TypenameField) isSelection() {}

func applySelectionSet(r *Request, s *resolvable.Schema, e *resolvable.Object, sels []query.Selection) (flattenedSels []Selection) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *query.Field:
			field := sel
			if skipByDirective(r, field.Directives) {
				continue
			}

			switch field.Name.Name {
			case "__typename":
				if !r.DisableIntrospection {
					flattenedSels = append(flattenedSels, &TypenameField{
						Object: *e,
						Alias:  field.Alias.Name,
					})
				}

			case "__schema":
				if !r.DisableIntrospection {
					flattenedSels = append(flattenedSels, &SchemaField{
						Field:       s.Meta.FieldSchema,
						Alias:       field.Alias.Name,
						Sels:        applySelectionSet(r, s, s.Meta.Schema, field.Selections),
						Async:       true,
						FixedResult: reflect.ValueOf(introspection.WrapSchema(r.Schema)),
					})
				}

			case "__type":
				if !r.DisableIntrospection {
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

					flattenedSels = append(flattenedSels, &SchemaField{
						Field:       s.Meta.FieldType,
						Alias:       field.Alias.Name,
						Sels:        applySelectionSet(r, s, s.Meta.Type, field.Selections),
						Async:       true,
						FixedResult: reflect.ValueOf(introspection.WrapType(t)),
					})
				}

			default:
				fe := e.Fields[field.Name.Name]

				var args map[string]interface{}
				var packedArgs reflect.Value
				if fe.ArgsPacker != nil {
					args = make(map[string]interface{})
					for _, arg := range field.Arguments {
						args[arg.Name.Name] = arg.Value.Value(r.Vars)
					}
					var err error
					packedArgs, err = fe.ArgsPacker.Pack(args)
					if err != nil {
						r.AddError(errors.Errorf("%s", err))
						return
					}
				}

				fieldSels := applyField(r, s, fe.ValueExec, field.Selections)
				flattenedSels = append(flattenedSels, &SchemaField{
					Field:      *fe,
					Alias:      field.Alias.Name,
					Args:       args,
					PackedArgs: packedArgs,
					Sels:       fieldSels,
					Async:      fe.HasContext || fe.ArgsPacker != nil || fe.HasError || HasAsyncSel(fieldSels),
				})
			}

		case *query.InlineFragment:
			frag := sel
			if skipByDirective(r, frag.Directives) {
				continue
			}
			flattenedSels = append(flattenedSels, applyFragment(r, s, e, &frag.Fragment)...)

		case *query.FragmentSpread:
			spread := sel
			if skipByDirective(r, spread.Directives) {
				continue
			}
			flattenedSels = append(flattenedSels, applyFragment(r, s, e, &r.Doc.Fragments.Get(spread.Name.Name).Fragment)...)

		default:
			panic("invalid type")
		}
	}
	return
}

func applyFragment(r *Request, s *resolvable.Schema, e *resolvable.Object, frag *query.Fragment) []Selection {
	if frag.On.Name != "" && frag.On.Name != e.Name {
		a, ok := e.TypeAssertions[frag.On.Name]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.Name)) // TODO proper error handling
		}

		return []Selection{&TypeAssertion{
			TypeAssertion: *a,
			Sels:          applySelectionSet(r, s, a.TypeExec.(*resolvable.Object), frag.Selections),
		}}
	}
	return applySelectionSet(r, s, e, frag.Selections)
}

func applyField(r *Request, s *resolvable.Schema, e resolvable.Resolvable, sels []query.Selection) []Selection {
	switch e := e.(type) {
	case *resolvable.Object:
		return applySelectionSet(r, s, e, sels)
	case *resolvable.List:
		return applyField(r, s, e.Elem, sels)
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

func HasAsyncSel(sels []Selection) bool {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *SchemaField:
			if sel.Async {
				return true
			}
		case *TypeAssertion:
			if HasAsyncSel(sel.Sels) {
				return true
			}
		case *TypenameField:
			// sync
		default:
			panic("unreachable")
		}
	}
	return false
}
