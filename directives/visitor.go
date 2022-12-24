package directives

import (
	"context"

	"github.com/graph-gophers/graphql-go/types"
)

// Visitor defines the interface that clients should use to implement a Directive
// see the graphql.DirectiveVisitors() Schema Option
type Visitor interface {
	Before(ctx context.Context, directive *types.Directive, input interface{}) error
	After(ctx context.Context, directive *types.Directive, output interface{}) (interface{}, error)
}
