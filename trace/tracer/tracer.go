package tracer

import (
	"context"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
)

type Tracer interface {
	TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, func([]*errors.QueryError))
	TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, func(*errors.QueryError))
}

type ValidationTracer interface {
	TraceValidation(ctx context.Context) func([]*errors.QueryError)
}

// Deprecated: use ValidationTracerContext instead.
type LegacyValidationTracer interface {
	TraceValidation() func([]*errors.QueryError)
}

// Deprecated: use a Tracer which implements ValidationTracerContext.
type LegacyNoopValidationTracer struct{}

// Deprecated: use a Tracer which implements ValidationTracerContext.
func (LegacyNoopValidationTracer) TraceValidation() func([]*errors.QueryError) {
	return func(errs []*errors.QueryError) {}
}
