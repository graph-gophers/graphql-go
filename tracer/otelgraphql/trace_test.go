package otelgraphql_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/tracer"
	"github.com/graph-gophers/graphql-go/tracer/otelgraphql"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ tracer.ValidationTracerContext = &otelgraphql.OpenTelemetryTracer{}
	var _ tracer.Tracer = &otelgraphql.OpenTelemetryTracer{}
}
