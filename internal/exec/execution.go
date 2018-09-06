package exec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/log"
	"github.com/graph-gophers/graphql-go/resolvers"
	"github.com/graph-gophers/graphql-go/trace"
	"reflect"
	"sync"
)

type Execution struct {
	Schema          *schema.Schema
	Vars            map[string]interface{}
	Doc             *query.Document
	Operation       *query.Operation
	Limiter         chan byte
	Tracer          trace.Tracer
	Logger          log.Logger
	Root            interface{}
	VarTypes        map[string]*introspection.Type
	Context         context.Context
	ResolverFactory resolvers.ResolverFactory
	Mu              sync.Mutex
	Errs            []*errors.QueryError
	Out             *bufio.Writer
}

func (this *Execution) GetSchema() *schema.Schema {
	return this.Schema
}

func (this *Execution) GetContext() context.Context {
	return this.Context
}
func (this *Execution) GetLimiter() *chan byte {
	return &this.Limiter
}
func (this *Execution) HandlePanic(path []string) error {
	if value := recover(); value != nil {
		this.Logger.LogPanic(this.Context, value)
		err := makePanicError(value)
		err.Path = stringToInterfaceArray(path)
		return err
	}
	return nil
}

func makePanicError(value interface{}) *errors.QueryError {
	return errors.Errorf("graphql: panic occurred: %v", value)
}

func (this *Execution) Execute() []*errors.QueryError {

	// This is the first execution goroutine.
	this.Limiter <- 1;
	defer func() { <-this.Limiter }()

	path := []string{}
	errors := this.recursiveExecute(path, this.Root, nil, this.Operation.Selections)
	this.Out.Flush()

	return errors
}

type selectionResolver struct {
	field      *query.Field
	resolver   resolvers.Resolver
	selections []query.Selection
}

type selectionResolvers struct {
	keys        []string
	valuesByKey map[string]*selectionResolver
}

func (this *selectionResolvers) set(key string, value *selectionResolver) *selectionResolver {
	if previousValue, found := this.valuesByKey[key]; found {
		this.valuesByKey[key] = value
		return previousValue;
	}
	this.valuesByKey[key] = value
	this.keys = append(this.keys, key)
	return nil;
}

func (this *Execution) CreateSelectionResolvers(path []string, selectionResolvers *selectionResolvers, parent interface{}, parentType common.Type, selections []query.Selection) {
	for _, selection := range selections {
		switch field := selection.(type) {
		case *query.Field:
			if this.skipByDirective(field.Directives) {
				continue
			}

			sr := selectionResolvers.valuesByKey[field.Alias.Name]
			if sr == nil {
				sr = &selectionResolver{}
			}
			sr.field = field
			sr.selections = append(sr.selections, field.Selections...)

			if sr.resolver == nil {

				typeName := parentType
				evaluatedArguments := make(map[string]interface{}, len(field.Arguments))
				for _, arg := range field.Arguments {
					evaluatedArguments[arg.Name.Name] = arg.Value.Value(this.Vars)
				}

				resolver := this.ResolverFactory.CreateResolver(&resolvers.ResolveRequest{
					Context:    this,
					ParentType: typeName,
					Parent:     parent,
					Field:      field.Schema.Field,
					Args:       evaluatedArguments,
					Selection:  field,
				})

				if resolver == nil {
					this.AddError(&errors.QueryError{
						Message: "No resolver found",
						Path:    stringToInterfaceArray(append(path, field.Alias.Name)),
					})
				} else {
					sr.resolver = resolver
					selectionResolvers.set(field.Alias.Name, sr)
				}
			}

		case *query.InlineFragment:
			if this.skipByDirective(field.Directives) {
				continue;
			}

			fragment := &field.Fragment
			this.CreateSelectionResolversForFragment(path, fragment, parentType, parent, selectionResolvers)

		case *query.FragmentSpread:
			if this.skipByDirective(field.Directives) {
				continue
			}
			fragment := &this.Doc.Fragments.Get(field.Name.Name).Fragment
			this.CreateSelectionResolversForFragment(path, fragment, parentType, parent, selectionResolvers)
		}
	}
}

func (this *Execution) CreateSelectionResolversForFragment(path []string, fragment *query.Fragment, parentType common.Type, parent interface{}, selectionResolvers *selectionResolvers) {
	if fragment.On.Name != "" && fragment.On.Name != parentType.String() {
		castType := this.Schema.Types[fragment.On.Name]
		if casted, ok := resolvers.TryCastFunction(parent, fragment.On.Name); ok {
			this.CreateSelectionResolvers(path, selectionResolvers, casted, castType, fragment.Selections)
		}
	} else {
		this.CreateSelectionResolvers(path, selectionResolvers, parent, parentType, fragment.Selections)
	}
}

func (this *Execution) recursiveExecute(path []string, parent interface{}, parentType common.Type, selections []query.Selection) []*errors.QueryError {
	func() {
		defer func() {
			if value := recover(); value != nil {
				this.Logger.LogPanic(this.Context, value)
				err := makePanicError(value)
				err.Path = stringToInterfaceArray(path)
				this.AddError(err)
			}
		}()

		// Create resolvers for the the selections.  Creating resolvers can trigger async fetching of
		// the field data.
		selectedFields := &selectionResolvers{}
		selectedFields.valuesByKey = make(map[string]*selectionResolver)
		this.CreateSelectionResolvers(path, selectedFields, parent, parentType, selections)

		// Write the
		this.Out.WriteByte('{')

		writeComma := false;
		for _, selectedFieldName := range selectedFields.keys {
			if writeComma {
				this.Out.WriteByte(',')
			}
			writeComma = true;
			selected := selectedFields.valuesByKey[selectedFieldName]
			field := selected.field
			childPath := append(path, selectedFieldName)

			this.Out.WriteByte('"')
			this.Out.WriteString(selectedFieldName)
			this.Out.WriteByte('"')
			this.Out.WriteByte(':')

			resolver := selected.resolver

			child, err := resolver()
			if err != nil {
				this.AddError(&errors.QueryError{
					Message:       err.Error(),
					Path:          stringToInterfaceArray(childPath),
					ResolverError: err,
				})
				continue
			}

			childType, nonNullType := unwrapNonNull(field.Schema.Field.Type)
			if (child == nil) {
				if (nonNullType) {
					this.AddError(&errors.QueryError{
						Message: "ResolverFactory produced a nil value for a Non Null type",
						Path:    stringToInterfaceArray(childPath),
					})
				} else {
					this.Out.WriteString("null")
				}
				continue
			}

			// Are we a leaf node?
			if selected.selections == nil {
				this.writeLeaf(child, childPath, childType)
			} else {
				switch childType := childType.(type) {
				case *common.List:
					this.writeList(*childType, child, childPath, func(elementType common.Type, element interface{}) {
						this.recursiveExecute(childPath, element, elementType, selected.selections)
					})
				case *schema.Object, *schema.Interface, *schema.Union:
					this.recursiveExecute(childPath, child, childType, selected.selections)
				}
			}

		}
		this.Out.WriteByte('}')

	}()
	if err := this.Context.Err(); err != nil {
		return []*errors.QueryError{errors.Errorf("%s", err)}
	}
	return this.Errs
}

func (this *Execution) skipByDirective(directives common.DirectiveList) bool {
	if d := directives.Get("skip"); d != nil {
		p := packer.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(d.Args.MustGet("if").Value(this.Vars))
		if err != nil {
			this.AddError(errors.Errorf("%s", err))
		}
		if err == nil && v.Bool() {
			return true
		}
	}

	if d := directives.Get("include"); d != nil {
		p := packer.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(d.Args.MustGet("if").Value(this.Vars))
		if err != nil {
			this.AddError(errors.Errorf("%s", err))
		}
		if err == nil && !v.Bool() {
			return true
		}
	}
	return false
}

func stringToInterfaceArray(v []string) []interface{} {
	rc := make([]interface{}, len(v))
	for key, value := range v {
		rc[key] = value
	}
	return rc
}

func (this *Execution) writeList(listType common.List, list interface{}, listPath []string, writeElement func(elementType common.Type, element interface{})) {
	childValue := reflect.ValueOf(list);

	// Dereference pointers..
	for ; childValue.Kind() == reflect.Ptr; {
		childValue = childValue.Elem()
	}

	switch childValue.Kind() {
	case reflect.Slice, reflect.Array:
		l := childValue.Len()
		this.Out.WriteByte('[')
		for i := 0; i < l; i++ {
			if i > 0 {
				this.Out.WriteByte(',')
			}
			element := childValue.Index(i).Interface()
			switch elementType := listType.OfType.(type) {
			case *common.List:
				this.writeList(*elementType, element, listPath, writeElement)
			default:
				writeElement(elementType, element)
			}
		}
		this.Out.WriteByte(']')
	default:
		this.AddError(&errors.QueryError{
			Message: fmt.Sprintf("Resolved object was not an array, it was a: %v", childValue.Kind()),
			Path:    stringToInterfaceArray(listPath),
		});
	}
}

func (this *Execution) writeLeaf(child interface{}, childPath []string, childType common.Type) {
	switch childType := childType.(type) {
	case *common.NonNull:
		childValue := reflect.ValueOf(child)
		if (childValue.Kind() == reflect.Ptr && childValue.Elem().IsNil()) {
			this.AddError(&errors.QueryError{
				Message: "Resolved to nil value for a Non Null type.",
				Path:    stringToInterfaceArray(childPath),
			})
		} else {
			this.writeLeaf(child, childPath, childType.OfType)
		}

	case *schema.Scalar:
		data, err := json.Marshal(child)
		if err != nil {
			this.AddError(&errors.QueryError{
				Message:       fmt.Sprintf("could not json.Marshal(%v)", child),
				Path:          stringToInterfaceArray(childPath),
				ResolverError: err,
			})
			return
		}
		this.Out.Write(data)

	case *schema.Enum:

		// Deref the pointer.
		childValue := reflect.ValueOf(child)
		for ; childValue.Kind() == reflect.Ptr; {
			childValue = childValue.Elem()
		}

		this.Out.WriteByte('"')
		this.Out.WriteString(childValue.String())
		this.Out.WriteByte('"')

	case *common.List:
		this.writeList(*childType, child, childPath, func(elementType common.Type, element interface{}) {
			this.writeLeaf(element, childPath, childType.OfType)
		})

	default:
		panic(fmt.Sprintf("Unknown type: %s", childType))
	}
}

func (r *Execution) AddError(err *errors.QueryError) {
	if err != nil {
		r.Mu.Lock()
		r.Errs = append(r.Errs, err)
		r.Mu.Unlock()
	}
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
