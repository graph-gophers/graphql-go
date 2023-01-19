package directives

import (
	"context"

	"github.com/graph-gophers/graphql-go/types"
)

// Visitor defines the interface that clients should use to implement a Directive
// see the graphql.DirectiveVisitors() Schema Option.
type Visitor interface {
	// Before() is always called when the operation includes a directive matching this implementation's name.
	// When the first return value is true, the field resolver will not be called.
	// Errors in Before() will prevent field resolution.
	Before(ctx context.Context, directive *types.Directive, input interface{}) (skipResolver bool, err error)
	// After is called if Before() *and* the field resolver do not error.
	After(ctx context.Context, directive *types.Directive, output interface{}) (modified interface{}, err error)
}
