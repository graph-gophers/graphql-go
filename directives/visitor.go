package directives

import (
	"context"

	"github.com/graph-gophers/graphql-go/types"
)

// Visitor defines the interface that clients should use to implement a Directive
// see the graphql.DirectiveVisitors() Schema Option
type Visitor interface {
	// Before() is always called when the operation includes a directive matching this implementation's name
	// Errors in Before() will prevent field resolution
	Before(ctx context.Context, directive *types.Directive, input interface{}) error
	// After is called if Before() *and* the field resolver do not error
	After(ctx context.Context, directive *types.Directive, output interface{}) (interface{}, error)
}
