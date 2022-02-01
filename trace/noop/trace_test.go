package noop_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/trace/noop"
	"github.com/graph-gophers/graphql-go/trace/tracer"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ tracer.ValidationTracer = &noop.Tracer{}
	var _ tracer.Tracer = &noop.Tracer{}
}

func TestTracerOption(t *testing.T) {
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(noop.Tracer{}))
	if err != nil {
		t.Fatal(err)
	}
}
