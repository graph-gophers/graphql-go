package otel_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/trace"
	"github.com/graph-gophers/graphql-go/trace/otel"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ trace.ValidationTracerContext = &otel.Tracer{}
	var _ trace.Tracer = &otel.Tracer{}
}
