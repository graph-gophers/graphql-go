package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestDuplicateInterfaceImplementationRejected(t *testing.T) {
	cases := []struct{ name, sdl string }{
		{"object", `interface Node { id: ID! } type Foo implements Node & Node { id: ID! } type Query { f: Foo }`},
		{"interface", `interface Named { name: String } interface Node implements Named & Named { id: ID! name: String } type Query { n: Node }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if err == nil {
				t.Fatal("duplicate interface implementation was accepted, want an error")
			}
			if !strings.Contains(err.Error(), "more than once") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
