package graphql_test

import (
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

func TestExtensionWithoutFieldsKeepsNextDefinition(t *testing.T) {
	_, err := graphql.ParseSchema(`
		type Query { field: String }
		interface Greeting { field: String }
		interface Farewell { field: String extra: String }
		type Hello implements Greeting { field: String extra: String }

		extend type Hello implements Farewell
		extend type Hello { newField: String }
	`, nil)
	if err != nil {
		t.Fatalf("expected adjacent interface-only extension to parse, got %v", err)
	}
}
