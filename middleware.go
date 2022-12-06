package graphql

import (
	"context"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
)

// Exec executes the given query with the schema's resolver.
type Exec func(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, res *resolvable.Schema) *Response

// Middleware can wrap Exec to add additional behaviour
type Middleware func(next Exec) Exec

func ParseErrorsMiddleware(parseErrors func([]*errors.QueryError) []*errors.QueryError) Middleware {
	return func(next Exec) Exec {
		return func(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, res *resolvable.Schema) *Response {
			// perform the original query
			response := next(ctx, queryString, operationName, variables, res)
			// mutate the errors
			response.Errors = parseErrors(response.Errors)
			// return the response
			return response
		}
	}
}
