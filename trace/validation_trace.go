package trace

import (
	"github.com/graph-gophers/graphql-go/tracer"
)

// Deprecated: <reason> ?
type TraceValidationFinishFunc = tracer.ValidationFinishFunc

// Deprecated: use ValidationTracerContext.
type ValidationTracer = tracer.ValidationTracer

// Deprecated: <reason> ?
type ValidationTracerContext = tracer.ValidationTracerContext

// Deprecated: <reason> ?
type NoopValidationTracer = tracer.NoopValidationTracer
