package opentracing_test

import (
	"testing"

	"github.com/tokopedia/graphql-go"
	"github.com/tokopedia/graphql-go/example/starwars"
	"github.com/tokopedia/graphql-go/trace/opentracing"
	"github.com/tokopedia/graphql-go/trace/tracer"
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
