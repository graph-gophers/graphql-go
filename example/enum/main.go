// Package main demonstrates a simple web app that uses type-safe enums in a GraphQL resolver.
package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

//go:embed index.html
var page []byte

//go:embed schema.graphql
var sdl string

type resolver struct {
	state State
}

func (r *resolver) Query() *queryResolver {
	return &queryResolver{state: &r.state}
}

func (r *resolver) Mutation() *mutationResolver {
	return &mutationResolver{state: &r.state}
}

type queryResolver struct {
	state *State
}

func (q *queryResolver) State(ctx context.Context) State {
	return *q.state
}

type mutationResolver struct {
	state *State
}

func (m *mutationResolver) State(ctx context.Context, args *struct{ State State }) State {
	*m.state = args.State
	return *m.state
}

func main() {
	opts := []graphql.SchemaOpt{graphql.UseStringDescriptions()}
	schema := graphql.MustParseSchema(sdl, &resolver{}, opts...)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write(page) })
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
