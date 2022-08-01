// The trace package provides tracing functionality.
// Deprecated: this package has been deprecated. Use package trace/tracer instead.
package trace

import (
	"github.com/tokopedia/graphql-go/errors"
	"github.com/tokopedia/graphql-go/trace/noop"
	"github.com/tokopedia/graphql-go/trace/opentracing"
	"github.com/tokopedia/graphql-go/trace/tracer"
)

// Deprecated: this type has been deprecated. Use tracer.QueryFinishFunc instead.
type TraceQueryFinishFunc = func([]*errors.QueryError)

// Deprecated: this type has been deprecated. Use tarcer.FieldFinishFunc instead.
type TraceFieldFinishFunc = func(*errors.QueryError)

// Deprecated: this interface has been deprecated. Use tracer.Tracer instead.
type Tracer = tracer.Tracer

// Deprecated: this type has been deprecated. Use opentracing.Tracer instead.
type OpenTracingTracer = opentracing.Tracer

// Deprecated: this type has been deprecated. Use noop.Tracer instead.
type NoopTracer = noop.Tracer
