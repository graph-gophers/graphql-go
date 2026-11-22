package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestExtensionNonRepeatableDirectiveRejected(t *testing.T) {
	sdl := `directive @tag(name: String!) on OBJECT
type Query @tag(name: "original") { f: String }
extend type Query @tag(name: "duplicate") { g: String }`
	_, err := graphql.ParseSchema(sdl, nil)
	if err == nil {
		t.Fatal("repeating a non-repeatable directive on an extension should fail")
	}
	if !strings.Contains(err.Error(), "tag") {
		t.Fatalf("unexpected error: %v", err)
	}
}
