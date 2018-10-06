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
		err.Path = path
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

	rootValue := reflect.ValueOf(this.Root)
	errors := this.recursiveExecute(nil, rootValue, nil, this.Operation.Selections)
	this.Out.Flush()

	return errors
}

type selectionResolver struct {
	parent     *selectionResolver
	field      *query.Field
	resolver   resolvers.Resolver
	selections []query.Selection
}

func (this *selectionResolver) Path() []string {
	if (this == nil) {
		return []string{}
	}
	if this.parent == nil {
		return []string{this.field.Alias.Name}
	}
	return append(this.parent.Path(), this.field.Alias.Name)
}

type linkedMapEntry struct {
	value interface{}
	next  *linkedMapEntry
}
type linkedMap struct {
	valuesByKey map[interface{}]*linkedMapEntry
	first       *linkedMapEntry
	last        *linkedMapEntry
}

func CreateLinkedMap(size int) *linkedMap {
	return &linkedMap{
		valuesByKey: make(map[interface{}]*linkedMapEntry, size),
	}
}

func (this *linkedMap) Get(key interface{}) interface{} {
	entry := this.valuesByKey[key]
	if entry == nil {
		return nil
	}
	return entry.value
}

func (this *linkedMap) Set(key interface{}, value interface{}) interface{} {
	if previousEntry, found := this.valuesByKey[key]; found {
		prevValue := previousEntry.value
		entry := this.valuesByKey[key]
		entry.value = value
		return prevValue;
	}
	entry := &linkedMapEntry{
		value: value,
	}
	if this.first == nil {
		this.first = entry
		this.last = entry
	} else {
		this.last.next = entry
		this.last = entry
	}
	this.valuesByKey[key] = entry
	return nil;
}

func (this *Execution) CreateSelectionResolvers(parentSelectionResolver *selectionResolver, selectionResolvers *linkedMap, parentValue reflect.Value, parentType common.Type, selections []query.Selection) {
	for _, selection := range selections {
		switch field := selection.(type) {
		case *query.Field:
			if this.skipByDirective(field.Directives) {
				continue
			}

			var sr *selectionResolver = nil
			x := selectionResolvers.Get(field.Alias.Name)
			if x != nil {
				sr = x.(*selectionResolver)
			} else {
				sr = &selectionResolver{}
				sr.field = field
				sr.parent = parentSelectionResolver
			}

			if sr.resolver == nil {

				// This field has not been resolved yet..
				sr.selections = field.Selections

				typeName := parentType
				evaluatedArguments := make(map[string]interface{}, len(field.Arguments))
				for _, arg := range field.Arguments {
					evaluatedArguments[arg.Name.Name] = arg.Value.Value(this.Vars)
				}

				resolver := this.ResolverFactory.CreateResolver(&resolvers.ResolveRequest{
					Context:       this,
					ParentType:    typeName,
					Parent:        parentValue,
					Field:         field.Schema.Field,
					Args:          evaluatedArguments,
					Selection:     field,
					SelectionPath: sr.Path,
				})

				if resolver == nil {
					this.AddError(&errors.QueryError{
						Message: "No resolver found",
						Path:    append(parentSelectionResolver.Path(), field.Alias.Name),
					})
				} else {
					sr.resolver = resolver
					selectionResolvers.Set(field.Alias.Name, sr)
				}
			} else {
				// field previously resolved, but fragment is adding more child field selections.
				sr.selections = append(sr.selections, field.Selections...)
			}

		case *query.InlineFragment:
			if this.skipByDirective(field.Directives) {
				continue;
			}

			fragment := &field.Fragment
			this.CreateSelectionResolversForFragment(parentSelectionResolver, fragment, parentType, parentValue, selectionResolvers)

		case *query.FragmentSpread:
			if this.skipByDirective(field.Directives) {
				continue
			}
			fragment := &this.Doc.Fragments.Get(field.Name.Name).Fragment
			this.CreateSelectionResolversForFragment(parentSelectionResolver, fragment, parentType, parentValue, selectionResolvers)
		}
	}
}

func (this *Execution) CreateSelectionResolversForFragment(parentSelectionResolver *selectionResolver, fragment *query.Fragment, parentType common.Type, parentValue reflect.Value, selectionResolvers *linkedMap) {
	if fragment.On.Name != "" && fragment.On.Name != parentType.String() {
		castType := this.Schema.Types[fragment.On.Name]
		if casted, ok := resolvers.TryCastFunction(parentValue, fragment.On.Name); ok {
			this.CreateSelectionResolvers(parentSelectionResolver, selectionResolvers, casted, castType, fragment.Selections)
		}
	} else {
		this.CreateSelectionResolvers(parentSelectionResolver, selectionResolvers, parentValue, parentType, fragment.Selections)
	}
}

func (this *Execution) recursiveExecute(parentSelection *selectionResolver, parentValue reflect.Value, parentType common.Type, selections []query.Selection) []*errors.QueryError {
	{
		defer func() {
			if value := recover(); value != nil {
				this.Logger.LogPanic(this.Context, value)
				err := makePanicError(value)
				err.Path = parentSelection.Path()
				this.AddError(err)
			}
		}()

		// Create resolvers for the the selections.  Creating resolvers can trigger async fetching of
		// the field data.
		selectedFields := CreateLinkedMap(len(selections))
		this.CreateSelectionResolvers(parentSelection, selectedFields, parentValue, parentType, selections)

		// Write the
		this.Out.WriteByte('{')

		writeComma := false;
		for entry := selectedFields.first; entry != nil; entry = entry.next {
			if writeComma {
				this.Out.WriteByte(',')
			}
			writeComma = true;
			selected := entry.value.(*selectionResolver)
			field := selected.field

			this.Out.WriteByte('"')
			this.Out.WriteString(selected.field.Alias.Name)
			this.Out.WriteByte('"')
			this.Out.WriteByte(':')

			resolver := selected.resolver

			childValue, err := resolver()
			if err != nil {
				this.AddError(&errors.QueryError{
					Message:       err.Error(),
					Path:          selected.Path(),
					ResolverError: err,
				})
				continue
			}

			childType, nonNullType := unwrapNonNull(field.Schema.Field.Type)
			if ((childValue.Kind() == reflect.Ptr || childValue.Kind() == reflect.Interface) &&
				childValue.IsNil()) {
				if (nonNullType) {
					this.AddError(&errors.QueryError{
						Message: "ResolverFactory produced a nil value for a Non Null type",
						Path:    selected.Path(),
					})
				} else {
					this.Out.WriteString("null")
				}
				continue
			}

			// Are we a leaf node?
			if selected.selections == nil {
				this.writeLeaf(childValue, selected, childType)
			} else {
				switch childType := childType.(type) {
				case *common.List:
					this.writeList(*childType, childValue, selected, func(elementType common.Type, element reflect.Value) {
						this.recursiveExecute(selected, element, elementType, selected.selections)
					})
				case *schema.Object, *schema.Interface, *schema.Union:
					this.recursiveExecute(selected, childValue, childType, selected.selections)
				}
			}

		}
		this.Out.WriteByte('}')

	}
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

func (this *Execution) writeList(listType common.List, childValue reflect.Value, selectionResolver *selectionResolver, writeElement func(elementType common.Type, element reflect.Value)) {

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
			element := childValue.Index(i)
			switch elementType := listType.OfType.(type) {
			case *common.List:
				this.writeList(*elementType, element, selectionResolver, writeElement)
			default:
				writeElement(elementType, element)
			}
		}
		this.Out.WriteByte(']')
	default:
		this.AddError(&errors.QueryError{
			Message: fmt.Sprintf("Resolved object was not an array, it was a: %v", childValue.Kind()),
			Path:    selectionResolver.Path(),
		});
	}
}

func (this *Execution) writeLeaf(childValue reflect.Value, selectionResolver *selectionResolver, childType common.Type) {
	switch childType := childType.(type) {
	case *common.NonNull:
		if (childValue.Kind() == reflect.Ptr && childValue.Elem().IsNil()) {
			panic(errors.Errorf("got nil for non-null %q", childType))
		} else {
			this.writeLeaf(childValue, selectionResolver, childType.OfType)
		}

	case *schema.Scalar:
		data, err := json.Marshal(childValue.Interface())
		if err != nil {
			panic(errors.Errorf("could not marshal %v: %s", childValue, err))
		}
		this.Out.Write(data)

	case *schema.Enum:

		// Deref the pointer.
		for ; childValue.Kind() == reflect.Ptr; {
			childValue = childValue.Elem()
		}

		this.Out.WriteByte('"')
		this.Out.WriteString(childValue.String())
		this.Out.WriteByte('"')

	case *common.List:
		this.writeList(*childType, childValue, selectionResolver, func(elementType common.Type, element reflect.Value) {
			this.writeLeaf(element, selectionResolver, childType.OfType)
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
