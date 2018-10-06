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
	SelectionPath func()[]string
	ParentType    common.Type
	Parent        reflect.Value
	Field         *schema.Field
	Args          map[string]interface{}
	Selection     *query.Field
}

type resolution struct {
	result reflect.Value
	err    error
}

func (this *ResolveRequest) RunAsync(resolver Resolver) Resolver {
	channel := make(chan *resolution, 1)
	r := resolution{
		result: reflect.Value{},
		err:    errors.New("Unknown"),
	}

	// Limit the number of concurrent go routines that we startup.
	*this.Context.GetLimiter() <- 1
	go func() {

		// Setup some post processing
		defer func() {
			err := this.Context.HandlePanic(this.SelectionPath())
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
	return func() (reflect.Value, error) {
		r := <-channel
		return r.result, r.err
	}
}

type ResolverFactoryFunc func(request *ResolveRequest) Resolver

type ResolverFactory interface {
	CreateResolver(request *ResolveRequest) Resolver
}
type Resolver func() (reflect.Value, error)

type dynamicResolverFactory struct {
}

func (this *dynamicResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	resolver := (&MetadataResolverFactory{}).CreateResolver(request)
	if resolver != nil {
		return resolver
	}
	resolver = (&MethodResolverFactory{}).CreateResolver(request)
	if resolver != nil {
		return resolver
	}
	resolver = (&FieldResolverFactory{}).CreateResolver(request)
	if resolver != nil {
		return resolver
	}
	resolver = (&MapResolverFactory{}).CreateResolver(request)
	if resolver != nil {
		return resolver
	}
	return nil
}

func DynamicResolverFactory() ResolverFactory {
	return &dynamicResolverFactory{}
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
	parentValue := dereference(request.Parent)
	if (parentValue.Kind() != reflect.Struct) {
		return nil
	}
	childValue, found := getChildField(&parentValue, request.Field.Name)
	if !found {
		return nil
	}
	return func() (reflect.Value, error) {
		return *childValue, nil
	}
}

func dereference(value reflect.Value) reflect.Value {
	for ; value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface ; {
		value = value.Elem()
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
	childMethod := getChildMethod(&request.Parent, request.Field.Name)
	if childMethod == nil {
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

	return func() (reflect.Value, error) {
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
				return reflect.Value{}, err
			}
			in = append(in, argValue)

		}
		result := request.Parent.Method(childMethod.Index).Call(in)
		if childMethod.hasError && !result[1].IsNil() {
			return reflect.Value{}, result[1].Interface().(error)
		}
		return result[0], nil

	}
}

///////////////////////////////////////////////////////////////////////
//
// MapResolverFactory resolves fields using entries in a map
//
///////////////////////////////////////////////////////////////////////
type MapResolverFactory struct{}

func (this *MapResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	parentValue := dereference(request.Parent)
	if (parentValue.Kind() != reflect.Map || parentValue.Type().Key().Kind() != reflect.String) {
		return nil
	}

	value := parentValue.MapIndex(reflect.ValueOf(request.Field.Name))
	if (!value.IsValid()) {
		return nil
	}

	return func() (reflect.Value, error) {
		return value, nil
	}
}

type MetadataResolverFactory struct{}

func (this *MetadataResolverFactory) CreateResolver(request *ResolveRequest) Resolver {
	s := request.Context.GetSchema()
	switch (request.Field.Name) {
	case "__typename":
		return func() (reflect.Value, error) {

			switch schemaType := request.ParentType.(type) {
			case *schema.Union:
				for _, pt := range schemaType.PossibleTypes {
					if _, ok := TryCastFunction(request.Parent, pt.Name); ok {
						return reflect.ValueOf(pt.Name), nil
					}
				}
			case *schema.Interface:
				for _, pt := range schemaType.PossibleTypes {
					if _, ok := TryCastFunction(request.Parent, pt.Name); ok {
						return reflect.ValueOf(pt.Name), nil
					}
				}
			default:
				return reflect.ValueOf(schemaType.String()), nil
			}
			return reflect.ValueOf(""), nil
		}

	case "__schema":
		return func() (reflect.Value, error) {
			return reflect.ValueOf(introspection.WrapSchema(s)), nil
		}

	case "__type":
		return func() (reflect.Value, error) {
			t, ok := s.Types[request.Args["name"].(string)]
			if !ok {
				return reflect.Value{}, fmt.Errorf("Could not find the type")
			}
			return reflect.ValueOf(introspection.WrapType(t)), nil
		}
	}
	return nil
}

func normalizeMethodName(method string) string {
	method = strings.Replace(method, "_", "", -1)
	method = strings.ToLower(method)
	return method;
}

var castMethodCache common.Cache

func TryCastFunction(parentValue reflect.Value, toType string) (reflect.Value, bool) {
	var key struct {
		fromType reflect.Type
		toType   string
	}
	key.fromType = parentValue.Type()
	key.toType = toType

	methodIndex := childMethodTypeCache.GetOrElseUpdate(key, func() interface{} {
		needle := normalizeMethodName("To" + toType)
		for methodIndex := 0; methodIndex < key.fromType.NumMethod(); methodIndex++ {
			method := normalizeMethodName(key.fromType.Method(methodIndex).Name);
			if needle == method {
				if key.fromType.Method(methodIndex).Type.NumIn() != 1 {
					continue;
				}
				if key.fromType.Method(methodIndex).Type.NumOut() != 2 {
					continue
				}
				if key.fromType.Method(methodIndex).Type.Out(1) != reflect.TypeOf(true) {
					continue;
				}
				return methodIndex
			}
		}
		return -1
	}).(int)
	if methodIndex == -1 {
		return reflect.Value{}, false
	}
	out := parentValue.Method(methodIndex).Call(nil)
	return out[0], out[1].Bool()
}
