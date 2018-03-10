package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/playground"
	"github.com/graph-gophers/graphql-go/relay"
)

var schema *graphql.Schema

func init() {
	schema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})
}

func main() {
	const endpoint = "/query"
	http.Handle("/", playground.Handler(endpoint, playground.WithTitle("Starwars example")))
	http.Handle(endpoint, &relay.Handler{Schema: schema})
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
