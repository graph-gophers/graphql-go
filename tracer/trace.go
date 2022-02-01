package tracer

import (
	"context"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
)

type QueryFinishFunc func([]*errors.QueryError)
type FieldFinishFunc func(*errors.QueryError)

type Tracer interface {
	TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, QueryFinishFunc)
	TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, FieldFinishFunc)
}

type Noop struct{}

func (Noop) TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, QueryFinishFunc) {
	return ctx, func(errs []*errors.QueryError) {}
}

func (Noop) TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, FieldFinishFunc) {
	return ctx, func(err *errors.QueryError) {}
}

func (Noop) TraceValidation(context.Context) ValidationFinishFunc {
	return func(errs []*errors.QueryError) {}
}

type ValidationFinishFunc = QueryFinishFunc

// Deprecated: use ValidationTracer.
type LegacyValidationTracer interface {
	TraceValidation() ValidationFinishFunc
}

type ValidationTracer interface {
	TraceValidation(ctx context.Context) ValidationFinishFunc
}

// Deprecated: <reason> ?
type NoopValidationTracer struct{}

// Deprecated: use a Tracer which implements ValidationTracer.
func (NoopValidationTracer) TraceValidation() ValidationFinishFunc {
	return func(errs []*errors.QueryError) {}
}

