package directives

import (
	"context"
)

// Directive defines the interface that clients should use to implement a custom Directive.
// Implementations then choose to implement other *optional* interfaces based on the needs of the directive, but each
// must implement *at least 1* of the optional functions
//
// See the graphql.Directives() Schema Option.
type Directive interface {
	ImplementsDirective() string
}

// Resolver for a field definition during execution of a request.
type Resolver interface {
	Resolve(ctx context.Context, args interface{}) (output interface{}, err error)
}

// ResolverInterceptor for a field resolver function, applying the directive logic.
// This is an *optional* directive function (at least 1 optional function must be declared for each directive).
type ResolverInterceptor interface {
	Resolve(ctx context.Context, args interface{}, next Resolver) (output interface{}, err error)
}
