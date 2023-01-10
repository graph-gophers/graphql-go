package trace_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/trace"
	"github.com/graph-gophers/graphql-go/trace/tracer"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ tracer.ValidationTracer = &trace.OpenTracingTracer{}
	var _ tracer.Tracer = &trace.OpenTracingTracer{}

	var _ tracer.ValidationTracer = &trace.NoopTracer{}
	var _ tracer.Tracer = &trace.NoopTracer{}
}

func TestTracerOption(t *testing.T) {
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(trace.OpenTracingTracer{}))
	if err != nil {
		t.Fatal(err)
	}
}

// MockVlidationTracer is a struct that implements the tracer.LegacyValidationTracer inteface.
type MockValidationTracer struct{}

func (MockValidationTracer) TraceValidation() func([]*errors.QueryError) {
	return func([]*errors.QueryError) {}
}

func TestValidationTracer(t *testing.T) {
	// test the legacy validation tracer interface (validating without using context) to ensure backwards compatibility
	vt := MockValidationTracer{}
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.ValidationTracer(vt))
	if err != nil {
		t.Fatal(err)
	}
}
