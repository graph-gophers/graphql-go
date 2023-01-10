package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

var schema = `
	schema {
		query: Query
	}
	
	type Query {
		hello: String!
	}
`

type resolver struct{}

func (r *resolver) Hello() string {
	return "Hello from subgraph one!"
}

func main() {
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.MaxParallelism(20)}
	schema := graphql.MustParseSchema(schema, &resolver{}, opts...)

	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":4001", nil))
}
