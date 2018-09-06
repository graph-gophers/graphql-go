package resolvers

import (
	"context"
	"errors"
	"fmt"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/introspection"
	"reflect"
	"strings"
)

type ExecutionContext interface {
	GetSchema() *schema.Schema
	GetContext() context.Context
	GetLimiter() *chan byte
	HandlePanic(selectionPath []string) error
}

type ResolveRequest struct {
	Context       ExecutionContext
	ParentResolve *ResolveRequest
	SelectionPath []string
	ParentType    common.Type
	Parent        interface{}
	Field         *schema.Field
	Args          map[string]interface{}
	Selection     *query.Field
}

type resolution struct {
	result interface{}
	err    error
}

func (this *ResolveRequest) RunAsync(resolver Resolver) Resolver {
	channel := make(chan *resolution, 1)
	r := resolution{
		result: nil,
		err:    errors.New("Unknown"),
	}

	// Limit the number of concurrent go routines that we startup.
	*this.Context.GetLimiter() <- 1
	go func() {

		// Setup some post processing
		defer func() {
			err := this.Context.HandlePanic(this.SelectionPath)
			if err != nil {
				r.err = err
			}
			<-*this.Context.GetLimiter()
			channel <- &r // we do this in defer since the resolver() could panic.
		}()

		//  Stash the results, then get sent back in the defer.
		r.result, r.err = resolver()
	}()

	// Return a resolver that waits for the async results
	return func() (interface{}, error) {
		r := <-channel
		return r.result, r.err
	}
}

type ResolverFactoryFunc func(request *ResolveRequest) Resolver

type ResolverFactory interface {
	CreateResolver(request *ResolveRequest) Resolver
}
type Resolver func() (interface{}, error)

func DynamicResolverFactory() ResolverFactory {
	return &ResolverFactoryList{
		&MetadataResolverFactory{},
		&MethodResolverFactory{},
		&FieldResolverFactory{},
		&MapResolverFactory{},
	}
}

type FuncResolverFactory struct {
	ResolverFactory ResolverFactoryFunc
}

func (this *FuncResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	return this.ResolverFactory(request)
}

type TypeResolverFactory map[string]ResolverFactoryFunc

func (this TypeResolverFactory) Set(typeName string, factory ResolverFactoryFunc) {
	this[typeName] = factory
}

func (this TypeResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	if request.ParentType == nil {
		return nil
	}
	resolverFunc := this[request.ParentType.String()]
	if resolverFunc == nil {
		return nil
	}
	return resolverFunc(request)
}

///////////////////////////////////////////////////////////////////////
//
// ResolverFactoryList uses a list of other resolvers to resolve
// requests.  First resolver that matches wins.
//
///////////////////////////////////////////////////////////////////////

type ResolverFactoryList []ResolverFactory

func (this *ResolverFactoryList) Add(factory ResolverFactory) {
	*this = append(*this, factory)
}

func (this *ResolverFactoryList) CreateResolver(request *ResolveRequest) Resolver {
	for _, f := range *this {
		resolver := f.CreateResolver(request)
		if (resolver != nil) {
			return resolver
		}
	}
	return nil
}

///////////////////////////////////////////////////////////////////////
//
// FieldResolverFactory resolves fields using struct fields on the parent
// value.
//
///////////////////////////////////////////////////////////////////////
type FieldResolverFactory struct{}

func (this *FieldResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	parentValue := derefIfPointer(reflect.ValueOf(request.Parent))
	if (parentValue.Kind() != reflect.Struct) {
		return nil
	}
	childValue, found := getChildField(&parentValue, request.Field.Name)
	if !found {
		return nil
	}
	return func() (interface{}, error) {
		return childValue.Interface(), nil
	}
}
func derefIfPointer(value reflect.Value) reflect.Value {
	if (value.Kind() == reflect.Ptr) {
		return value.Elem()
	}
	return value
}

///////////////////////////////////////////////////////////////////////
//
// MethodResolverFactory resolves fields using the method
// implemented by a receiver type.
//
///////////////////////////////////////////////////////////////////////
type MethodResolverFactory struct{}

func (this *MethodResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	parentValue := reflect.ValueOf(request.Parent)

	childMethod, found := getChildMethod(&parentValue, request.Field.Name)
	if !found {
		return nil
	}

	var structPacker *packer.StructPacker = nil
	if childMethod.argumentsType != nil {
		p := packer.NewBuilder()
		defer p.Finish()
		sp, err := p.MakeStructPacker(request.Field.Args, *childMethod.argumentsType)
		if err != nil {
			return nil
		}
		structPacker = sp;
	}

	return func() (interface{}, error) {

		var in []reflect.Value
		if childMethod.hasContext {
			in = append(in, reflect.ValueOf(request.Context.GetContext()))
		}
		if childMethod.hasExecutionContext {
			in = append(in, reflect.ValueOf(request.Context))
		}

		if childMethod.argumentsType != nil {

			argValue, err := structPacker.Pack(request.Args)
			if err != nil {
				return nil, err
			}
			in = append(in, argValue)

		}
		result := parentValue.Method(childMethod.Index).Call(in)
		if childMethod.hasError && !result[1].IsNil() {
			return nil, result[1].Interface().(error)
		}
		rc := result[0]
		switch rc.Kind() {
		case reflect.Ptr:
			if rc.IsNil() {
				return nil, nil
			}
		case reflect.String:
			return rc.String(), nil
		}
		return result[0].Interface(), nil

	}
}

///////////////////////////////////////////////////////////////////////
//
// MapResolverFactory resolves fields using entries in a map
//
///////////////////////////////////////////////////////////////////////
type MapResolverFactory struct{}

func (this *MapResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	parentValue := derefIfPointer(reflect.ValueOf(request.Parent))
	if (parentValue.Kind() != reflect.Map || parentValue.Type().Key().Kind() != reflect.String) {
		return nil
	}

	value := parentValue.MapIndex(reflect.ValueOf(request.Field.Name))
	if (!value.IsValid()) {
		return nil
	}

	return func() (interface{}, error) {
		return value.Interface(), nil
	}
}

type MetadataResolverFactory struct{}

func (this *MetadataResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	s := request.Context.GetSchema()
	switch (request.Field.Name) {
	case "__typename":
		return func() (interface{}, error) {

			switch schemaType := request.ParentType.(type) {
			case *schema.Union:
				for _, pt := range schemaType.PossibleTypes {
					if _, ok := TryCastFunction(request.Parent, pt.Name); ok {
						return pt.Name, nil
					}
				}
			case *schema.Interface:
				for _, pt := range schemaType.PossibleTypes {
					if _, ok := TryCastFunction(request.Parent, pt.Name); ok {
						return pt.Name, nil
					}
				}
			default:
				return schemaType.String(), nil
			}
			return "", nil
		}

	case "__schema":
		return func() (interface{}, error) {
			return introspection.WrapSchema(s), nil
		}

	case "__type":
		return func() (interface{}, error) {
			t, ok := s.Types[request.Args["name"].(string)]
			if !ok {
				return nil, fmt.Errorf("Could not find the type")
			}
			return introspection.WrapType(t), nil
		}
	}
	return nil
}

func normalizeMethodName(method string) string {
	method = strings.Replace(method, "_", "", -1)
	method = strings.ToLower(method)
	return method;
}

func TryCastFunction(parent interface{}, toType string) (interface{}, bool) {
	parentValue := reflect.ValueOf(parent);
	resolverType := parentValue.Type()
	needle := normalizeMethodName("To" + toType)
	for methodIndex := 0; methodIndex < resolverType.NumMethod(); methodIndex++ {
		method := normalizeMethodName(resolverType.Method(methodIndex).Name);
		if needle == method {

			if resolverType.Method(methodIndex).Type.NumIn() != 1 {
				continue;
			}
			if resolverType.Method(methodIndex).Type.NumOut() != 2 {
				continue
			}
			if resolverType.Method(methodIndex).Type.Out(1) != reflect.TypeOf(true) {
				continue;
			}
			out := parentValue.Method(methodIndex).Call(nil)
			return out[0].Interface(), out[1].Bool()
		}
	}
	return nil, false
}
