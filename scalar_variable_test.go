package graphql_test

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

type varCoercionResolver struct{}

func (r *varCoercionResolver) Hello(args struct{ Name int32 }) string { return "ok" }

func TestVariableCoercionWrongScalarTypeFailsBeforeExecution(t *testing.T) {
	schema := graphql.MustParseSchema(`
		schema { query: Query }
		type Query { hello(name: Int!): String! }
	`, &varCoercionResolver{})

	resp := schema.Exec(context.Background(),
		`query($name: Int!) { hello(name: $name) }`,
		"", map[string]interface{}{"name": "not_an_int"})

	if len(resp.Errors) == 0 {
		t.Fatal("expected an error for the scalar type mismatch")
	}
	if resp.Data != nil {
		t.Fatalf("operation with invalid variable must not execute, got data: %s", resp.Data)
	}
}
