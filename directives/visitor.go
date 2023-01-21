package directives

import (
	"context"
)

// Visitor defines the interface that clients should use to implement a Directive
// see the graphql.DirectiveVisitors() Schema Option.
type Visitor interface {
	ResolverVisitor
}

type Resolver interface {
	Resolve(ctx context.Context, args interface{}) (output interface{}, err error)
}

type ResolverFunc func(ctx context.Context, args interface{}) (output interface{}, err error)

func (f ResolverFunc) Resolve(ctx context.Context, args interface{}) (output interface{}, err error) {
	return f(ctx, args)
}

type ResolverVisitor interface {
	Resolve(ctx context.Context, args interface{}, next Resolver) (output interface{}, err error)
}

type ResolverVisitorFunc func(ctx context.Context, args interface{}, next Resolver) (output interface{}, err error)

func (f ResolverVisitorFunc) Resolve(ctx context.Context, args interface{}, next Resolver) (output interface{}, err error) {
	return f(ctx, args, next)
}
