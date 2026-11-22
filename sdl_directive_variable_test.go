package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestVariableNotAllowedInSchemaDirectiveArgument(t *testing.T) {
	sdl := `directive @d(x: String) on OBJECT
type Query @d(x: $v) { f: String }`
	_, err := graphql.ParseSchema(sdl, nil)
	if err == nil {
		t.Fatal("variable in SDL directive argument was accepted, want an error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
