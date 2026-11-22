package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestInputObjectCircularReferences(t *testing.T) {
	cases := []struct{ name, sdl string }{
		{"self", `input Loop { next: Loop! value: String } type Query { f(x: Loop): String }`},
		{"spread", `input A { b: B! } input B { a: A! } type Query { f(x: A): String }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if err == nil {
				t.Fatal("unbreakable input object cycle was accepted, want an error")
			}
			if !strings.Contains(err.Error(), "non-null fields") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
