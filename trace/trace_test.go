package trace_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/trace"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ trace.ValidationTracerContext = &trace.OpenTelemetryTracer{}
	var _ trace.Tracer = &trace.OpenTelemetryTracer{}

	var _ trace.ValidationTracerContext = &trace.OpenTracingTracer{}
	var _ trace.Tracer = &trace.OpenTracingTracer{}
}
