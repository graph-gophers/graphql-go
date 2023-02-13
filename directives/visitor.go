package directives

import (
	"context"
)

// Resolver for a field definition during execution of a request.
type Resolver interface {
	Resolve(ctx context.Context, args interface{}) (output interface{}, err error)
}

// ResolverInterceptor wraps a field resolver function, applying the directive logic.
type ResolverInterceptor interface {
	Resolve(ctx context.Context, args interface{}, next Resolver) (output interface{}, err error)
}
