package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/internal/graphiql"
	"github.com/graph-gophers/graphql-go/example/social"
	"github.com/graph-gophers/graphql-go/relay"
)

func main() {
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.MaxParallelism(20)}
	schema := graphql.MustParseSchema(social.Schema, &social.Resolver{}, opts...)

	http.Handle("GET /", graphiql.Handler())
	http.Handle("POST /query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
