package trace_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/trace"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ trace.ValidationTracerContext = &trace.OpenTracingTracer{}
	var _ trace.Tracer = &trace.OpenTracingTracer{}
}

func TestTracerOption(t *testing.T) {
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(trace.OpenTracingTracer{}))
	if err != nil {
		t.Fatal(err)
	}
}
