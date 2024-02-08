package main

import (
	"log"
	"net/http"

	"github.com/tribunadigital/graphql-go"
	"github.com/tribunadigital/graphql-go/relay"
)

var schema = `
	schema {
		query: Query
	}

	type Query {
		hi: String!
	}
`

type resolver struct{}

func (r *resolver) Hi() string {
	return "Hi from subgraph two!"
}

func main() {
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.MaxParallelism(20)}
	schema := graphql.MustParseSchema(schema, &resolver{}, opts...)

	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":4002", nil))
}
