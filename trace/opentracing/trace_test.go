package opentracing_test

import (
	"testing"

	"github.com/tribunadigital/graphql-go"
	"github.com/tribunadigital/graphql-go/example/starwars"
	"github.com/tribunadigital/graphql-go/trace/opentracing"
	"github.com/tribunadigital/graphql-go/trace/tracer"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ tracer.ValidationTracer = &opentracing.Tracer{}
	var _ tracer.Tracer = &opentracing.Tracer{}
}

func TestTracerOption(t *testing.T) {
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(opentracing.Tracer{}))
	if err != nil {
		t.Fatal(err)
	}
}
