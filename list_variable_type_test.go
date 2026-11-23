package graphql_test

import (
	"context"
	"strings"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

type listVariableResolver struct{}

func (*listVariableResolver) Values(args struct{ Values []string }) []string {
	return args.Values
}

func TestConcreteSliceVariableReportsClearError(t *testing.T) {
	schema := graphql.MustParseSchema(`
		type Query { values(values: [String!]!): [String!]! }
	`, &listVariableResolver{})

	resp := schema.Exec(context.Background(),
		`query($values: [String!]!) { values(values: $values) }`, "",
		map[string]interface{}{"values": []map[string]interface{}{{"key": "value"}}})
	if len(resp.Errors) == 0 {
		t.Fatal("expected an error for a concrete slice variable")
	}
	if !strings.Contains(resp.Errors[0].Message, "[]interface{}") {
		t.Fatalf("expected a clear type error, got %q", resp.Errors[0].Message)
	}
}
