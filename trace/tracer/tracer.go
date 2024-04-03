// The tracer package provides tracing functionality.
package tracer

import (
	"context"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
)

type QueryFinishFunc = func([]*errors.QueryError)
type FieldFinishFunc = func(*errors.QueryError)
type ValidationFinishFunc = func([]*errors.QueryError)

type Tracer interface {
	TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, QueryFinishFunc)
	TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, FieldFinishFunc)
}

type ValidationTracer interface {
	TraceValidation(ctx context.Context) ValidationFinishFunc
}

// Deprecated: use [ValidationTracer] instead.
type LegacyValidationTracer interface {
	TraceValidation() func([]*errors.QueryError)
}

// Deprecated: use a Tracer which implements [ValidationTracer].
type LegacyNoopValidationTracer struct{}

// Deprecated: use a Tracer which implements [ValidationTracer].
func (LegacyNoopValidationTracer) TraceValidation() func([]*errors.QueryError) {
	return func(errs []*errors.QueryError) {}
}
