package trace

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// DefaultOpenTelemetryTracer creates a tracer using a default name
func DefaultOpenTelemetryTracer() Tracer {
	return &OpenTelemetryTracer{
		Tracer: otel.Tracer("graphql-go"),
	}
}

// OpenTelemetryTracer is an OpenTelemetry implementation for graphql-go. Set the Tracer
// property to your tracer instance as required.
type OpenTelemetryTracer struct {
	Tracer oteltrace.Tracer
}

func (t *OpenTelemetryTracer) TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, TraceQueryFinishFunc) {
	spanCtx, span := t.Tracer.Start(ctx, "GraphQL Request")

	var attributes []attribute.KeyValue
	attributes = append(attributes, attribute.String("graphql.query", queryString))
	if operationName != "" {
		attributes = append(attributes, attribute.String("graphql.operationName", operationName))
	}
	if len(variables) != 0 {
		attributes = append(attributes, attribute.String("graphql.variables", fmt.Sprintf("%v", variables)))
	}
	span.SetAttributes(attributes...)

	return spanCtx, func(errs []*errors.QueryError) {
		if len(errs) > 0 {
			msg := errs[0].Error()
			if len(errs) > 1 {
				msg += fmt.Sprintf(" (and %d more errors)", len(errs)-1)
			}

			span.SetStatus(codes.Error, msg)
		}
		span.End()
	}
}

func (t *OpenTelemetryTracer) TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, TraceFieldFinishFunc) {
	if trivial {
		return ctx, func(*errors.QueryError) {}
	}

	var attributes []attribute.KeyValue

	spanCtx, span := t.Tracer.Start(ctx, fmt.Sprintf("Field: %v", label))
	attributes = append(attributes, attribute.String("graphql.type", typeName))
	attributes = append(attributes, attribute.String("graphql.field", fieldName))
	for name, value := range args {
		attributes = append(attributes, attribute.String("graphql.args."+name, fmt.Sprintf("%v", value)))
	}
	span.SetAttributes(attributes...)

	return spanCtx, func(err *errors.QueryError) {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

func (t *OpenTelemetryTracer) TraceValidation(ctx context.Context) TraceValidationFinishFunc {
	_, span := t.Tracer.Start(ctx, "GraphQL Validate")

	return func(errs []*errors.QueryError) {
		if len(errs) > 0 {
			msg := errs[0].Error()
			if len(errs) > 1 {
				msg += fmt.Sprintf(" (and %d more errors)", len(errs)-1)
			}
			span.SetStatus(codes.Error, msg)
		}
		span.End()
	}
}
