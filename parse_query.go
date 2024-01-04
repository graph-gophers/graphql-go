package graphql

import (
	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/query"
)

// ParseQuery parses a GraphQL query string and returns the AST root node and
// any errors. It only serves to expose the internal query.Parse function.
func ParseQuery(queryString string) (*ast.ExecutableDefinition, *errors.QueryError) {
	return query.Parse(queryString)
}
