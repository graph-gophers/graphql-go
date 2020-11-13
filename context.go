package graphql

import (
	"context"

	gcontext "github.com/graph-gophers/graphql-go/internal/context"
	"github.com/graph-gophers/graphql-go/selected"
)

type Context struct {
	Field selected.Field
}

// GraphQLContext is used to retrieved the graphql from the context. If no graphql
// is present in the context, the `fallbackGraphql` received in parameter
// is returned instead.
func GraphQLContext(ctx context.Context) *Context {
	field, found := gcontext.GraphQL(ctx)
	if !found {
		return nil
	}

	return &Context{
		Field: field.ToSelection().(selected.Field),
	}
}
