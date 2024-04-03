# graphql-go [![Sourcegraph](https://sourcegraph.com/github.com/graph-gophers/graphql-go/-/badge.svg)](https://sourcegraph.com/github.com/graph-gophers/graphql-go?badge) [![Build Status](https://graph-gophers.semaphoreci.com/badges/graphql-go/branches/master.svg?style=shields)](https://graph-gophers.semaphoreci.com/projects/graphql-go) [![Go Report](https://goreportcard.com/badge/github.com/graph-gophers/graphql-go)](https://goreportcard.com/report/github.com/graph-gophers/graphql-go) [![GoDoc](https://godoc.org/github.com/graph-gophers/graphql-go?status.svg)](https://godoc.org/github.com/graph-gophers/graphql-go)

<p align="center"><img src="docs/img/logo.png" width="300"></p>

The goal of this project is to provide full support of the [October 2021 GraphQL specification](https://spec.graphql.org/October2021/) with a set of idiomatic, easy to use Go packages.

While still under development (`internal` APIs are almost certainly subject to change), this library is safe for production use.

## Features

- minimal API
- support for `context.Context`
- support for the `OpenTelemetry` and `OpenTracing` standards
- schema type-checking against resolvers
- resolvers are matched to the schema based on method sets (can resolve a GraphQL schema with a Go interface or Go struct).
- handles panics in resolvers
- parallel execution of resolvers
- subscriptions
  - [sample WS transport](https://github.com/graph-gophers/graphql-transport-ws)
- directive visitors on fields (the API is subject to change in future versions)

## (Some) Documentation [![GoDoc](https://godoc.org/github.com/graph-gophers/graphql-go?status.svg)](https://godoc.org/github.com/graph-gophers/graphql-go)

### Getting started

In order to run a simple GraphQL server locally create a `main.go` file with the following content:
```go
package main

import (
	"log"
	"net/http"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type query struct{}

func (query) Hello() string { return "Hello, world!" }

func main() {
	s := `
        type Query {
                hello: String!
        }
    `
	schema := graphql.MustParseSchema(s, &query{})
	http.Handle("/query", &relay.Handler{Schema: schema})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```
Then run the file with `go run main.go`. To test:
	    
```sh
curl -XPOST -d '{"query": "{ hello }"}' localhost:8080/query
```
For more realistic usecases check our [examples section](https://github.com/graph-gophers/graphql-go/wiki/Examples).

### Resolvers

A resolver must have one method or field for each field of the GraphQL type it resolves. The method or field name has to be [exported](https://golang.org/ref/spec#Exported_identifiers) and match the schema's field's name in a non-case-sensitive way.
You can use struct fields as resolvers by using `SchemaOpt: UseFieldResolvers()`. For example,
```
opts := []graphql.SchemaOpt{graphql.UseFieldResolvers()}
schema := graphql.MustParseSchema(s, &query{}, opts...)
```   

When using `UseFieldResolvers` schema option, a struct field will be used *only* when:
- there is no method for a struct field
- a struct field does not implement an interface method
- a struct field does not have arguments

The method has up to two arguments:

- Optional `context.Context` argument.
- Mandatory `*struct { ... }` argument if the corresponding GraphQL field has arguments. The names of the struct fields have to be [exported](https://golang.org/ref/spec#Exported_identifiers) and have to match the names of the GraphQL arguments in a non-case-sensitive way.

The method has up to two results:

- The GraphQL field's value as determined by the resolver.
- Optional `error` result.

Example for a simple resolver method:

```go
func (r *helloWorldResolver) Hello() string {
	return "Hello world!"
}
```

The following signature is also allowed:

```go
func (r *helloWorldResolver) Hello(ctx context.Context) (string, error) {
	return "Hello world!", nil
}
```

### Separate resolvers for different operations
> **NOTE**: This feature is not in the stable release yet. In order to use it you need to run `go get github.com/graph-gophers/graphql-go@master` and in your `go.mod` file you will have something like:
>  ```
>  v1.5.1-0.20230216224648-5aa631d05992
>  ```
> It is expected to be released in `v1.6.0` soon.

The GraphQL specification allows for fields with the same name defined in different query types. For example, the schema below is a valid schema definition:
```graphql
schema {
  query: Query
  mutation: Mutation
}

type Query {
  hello: String!
}

type Mutation {
  hello: String!
}
```
The above schema would result in name collision if we use a single resolver struct because fields from both operations correspond to methods in the root resolver (the same Go struct). In order to resolve this issue, the library allows resolvers for query, mutation and subscription operations to be separated using the `Query`, `Mutation` and `Subscription` methods of the root resolver. These special methods are optional and if defined return the resolver for each opeartion. For example, the following is a resolver corresponding to the schema definition above. Note that there is a field named `hello` in both the query and the mutation definitions:

```go
type RootResolver struct{}
type QueryResolver struct{}
type MutationResolver struct{}

func(r *RootResolver) Query() *QueryResolver {
  return &QueryResolver{}
}

func(r *RootResolver) Mutation() *MutationResolver {
  return &MutationResolver{}
}

func (*QueryResolver) Hello() string {
	return "Hello query!"
}

func (*MutationResolver) Hello() string {
	return "Hello mutation!"
}

schema := graphql.MustParseSchema(sdl, &RootResolver{}, nil)
...
```

### Schema Options

- `UseStringDescriptions()` enables the usage of double quoted and triple quoted. When this is not enabled, comments are parsed as descriptions instead.
- `UseFieldResolvers()` specifies whether to use struct field resolvers.
- `MaxDepth(n int)` specifies the maximum field nesting depth in a query. The default is 0 which disables max depth checking.
- `MaxParallelism(n int)` specifies the maximum number of resolvers per request allowed to run in parallel. The default is 10.
- `Tracer(tracer trace.Tracer)` is used to trace queries and fields. It defaults to `noop.Tracer`.
- `Logger(logger log.Logger)` is used to log panics during query execution. It defaults to `exec.DefaultLogger`.
- `PanicHandler(panicHandler errors.PanicHandler)` is used to transform panics into errors during query execution. It defaults to `errors.DefaultPanicHandler`.
- `DisableIntrospection()` disables introspection queries.
- `DirectiveVisitors()` adds directive visitor implementations to the schema. See examples/directives/authorization for an example.

### Custom Errors

Errors returned by resolvers can include custom extensions by implementing the `ResolverError` interface:

```go
type ResolverError interface {
	error
	Extensions() map[string]interface{}
}
```

Example of a simple custom error:

```go
type droidNotFoundError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e droidNotFoundError) Error() string {
	return fmt.Sprintf("error [%s]: %s", e.Code, e.Message)
}

func (e droidNotFoundError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	}
}
```

Which could produce a GraphQL error such as:

```go
{
  "errors": [
    {
      "message": "error [NotFound]: This is not the droid you are looking for",
      "path": [
        "droid"
      ],
      "extensions": {
        "code": "NotFound",
        "message": "This is not the droid you are looking for"
      }
    }
  ],
  "data": null
}
```

### Tracing

By default the library uses `noop.Tracer`. If you want to change that you can use the OpenTelemetry or the OpenTracing implementations, respectively:

```go
// OpenTelemetry tracer
package main

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	otelgraphql "github.com/graph-gophers/graphql-go/trace/otel"
	"github.com/graph-gophers/graphql-go/trace/tracer"
)
// ...
_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(otelgraphql.DefaultTracer()))
// ...
```
Alternatively you can pass an existing trace.Tracer instance:
```go
tr := otel.Tracer("example")
_, err = graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(&otelgraphql.Tracer{Tracer: tr}))
```


```go
// OpenTracing tracer
package main

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/trace/opentracing"
	"github.com/graph-gophers/graphql-go/trace/tracer"
)
// ...
_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(opentracing.Tracer{}))

// ...
```

If you need to implement a custom tracer the library would accept any tracer which implements the interface below:
```go
type Tracer interface {
    TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, func([]*errors.QueryError))
    TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, func(*errors.QueryError))
    TraceValidation(context.Context) func([]*errors.QueryError)
}
```


### [Examples](https://github.com/graph-gophers/graphql-go/wiki/Examples)

