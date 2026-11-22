package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestSchemaTypePositions(t *testing.T) {
	cases := []struct{ name, sdl string }{
		{"object field with input type", `input Filter { x: String } type Query { f: Filter }`},
		{"argument with output type", `type Foo { x: String } type Query { f(a: Foo): String }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if err == nil {
				t.Fatal("invalid type position was accepted, want an error")
			}
			if !strings.Contains(err.Error(), "must be an input type") && !strings.Contains(err.Error(), "must be an output type") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
