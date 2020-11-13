package context

import (
	"context"

	"github.com/graph-gophers/graphql-go/internal/exec/selected"
)

type graphqlKeyType int

const graphqlFieldKey graphqlKeyType = iota

// WithGraphQLContext is used to create a new context with a graphql added to it
// so it can be later retrieved using `Graphql`.
func WithGraphQLContext(ctx context.Context, field *selected.SchemaField) context.Context {
	return context.WithValue(ctx, graphqlFieldKey, field)
}

// GraphQL is used to retrieved the graphql from the context. If no graphql
// is present in the context, the `fallbackGraphql` received in parameter
// is returned instead.
func GraphQL(ctx context.Context) (field *selected.SchemaField, found bool) {
	if ctx == nil {
		return
	}

	if v, ok := ctx.Value(graphqlFieldKey).(*selected.SchemaField); ok {
		return v, true
	}

	return
}
