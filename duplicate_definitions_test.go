package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestDuplicateTypeDefinitionsRejected(t *testing.T) {
	cases := []struct{ name, sdl string }{
		{"object", `type Foo { a: String } type Foo { b: String } type Query { f: Foo }`},
		{"mixed kinds", `type Foo { a: String } scalar Foo type Query { f: Foo }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if err == nil {
				t.Fatal("duplicate type definition was accepted, want an error")
			}
			if !strings.Contains(err.Error(), "Foo") {
				t.Fatalf("error %q does not name the duplicate type", err)
			}
		})
	}
}

func TestDuplicateDirectiveDefinitionsRejected(t *testing.T) {
	sdl := `directive @foo on FIELD | QUERY
directive @foo on FIELD | QUERY
type Query { f: String }`
	_, err := graphql.ParseSchema(sdl, nil)
	if err == nil {
		t.Fatal("duplicate directive definition was accepted, want an error")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Fatalf("error %q does not name the duplicate directive", err)
	}
}

func TestTypeAndDirectiveMayShareName(t *testing.T) {
	sdl := `directive @foo on FIELD | QUERY
scalar foo
type Query { f: foo }`
	if _, err := graphql.ParseSchema(sdl, nil); err != nil {
		t.Fatalf("type and directive sharing a name should be valid: %v", err)
	}
}
