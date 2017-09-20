package main

import (
	"log"
	"net/http"

	"github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/example/starwars"
	"github.com/neelance/graphql-go/relay"
)

var schema *graphql.Schema

func init() {
	schema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})
}

func main() {
	h := relay.New(&relay.Config{
		Schema:   schema,
		Pretty:   true,
		GraphiQL: true,
	})
	// Endpoint can be set to anything, using `/graphiql` just for convention.
	http.Handle("/graphiql", h)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
