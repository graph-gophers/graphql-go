package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

var sdl = `
	type Query {
		hi: String!
		_service: Service!
	}

	type Service {
		sdl: String!
	}
`

type resolver struct {
	Service func() Service `graphql:"_service"`
}

func (r *resolver) Hi() string {
	return "Hi from subgraph two!"
}

func service(s string) func() Service {
	return func() Service {
		return Service{SDL: s}
	}
}

type Service struct {
	SDL string
}

func main() {
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.MaxParallelism(20)}
	schema := graphql.MustParseSchema(sdl, &resolver{Service: service(sdl)}, opts...)

	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":4002", nil))
}
