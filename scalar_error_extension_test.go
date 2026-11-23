package graphql_test

import (
	"context"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

type faultyScalar string

func (faultyScalar) ImplementsGraphQLType(name string) bool {
	return name == "FaultyScalar"
}

func (*faultyScalar) UnmarshalGraphQL(input interface{}) error {
	return &faultyScalarError{}
}

type faultyScalarError struct{}

func (*faultyScalarError) Error() string {
	return "invalid scalar"
}

func (*faultyScalarError) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": "INVALID_SCALAR"}
}

type scalarErrorResolver struct{}

func (*scalarErrorResolver) Field(args struct{ Value faultyScalar }) string {
	return "unreachable"
}

func TestScalarErrorExtensionsPreserved(t *testing.T) {
	schema := graphql.MustParseSchema(`
		schema { query: Query }
		scalar FaultyScalar
		type Query { field(value: FaultyScalar!): String! }
	`, &scalarErrorResolver{})

	resp := schema.Exec(context.Background(), `query { field(value: "bad") }`, "", nil)
	if len(resp.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(resp.Errors))
	}
	if resp.Errors[0].Extensions["code"] != "INVALID_SCALAR" {
		t.Fatalf("expected scalar error extensions to be preserved, got %v", resp.Errors[0].Extensions)
	}
}
