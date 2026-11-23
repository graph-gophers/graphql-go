package graphql_test

import (
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

func TestSchemaRejectsReusedRootOperationTypes(t *testing.T) {
	_, err := graphql.ParseSchema(`
		type SomeObject { field: String }
		schema {
			query: SomeObject
			mutation: SomeObject
		}
	`, nil)
	if err == nil {
		t.Fatal("expected schema parse to reject a type used as both query and mutation root")
	}
}
