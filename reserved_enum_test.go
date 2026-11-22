package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestReservedEnumValueNamesRejected(t *testing.T) {
	for _, name := range []string{"true", "false", "null"} {
		t.Run(name, func(t *testing.T) {
			sdl := `enum Color { RED ` + name + ` } type Query { color: Color }`
			_, err := graphql.ParseSchema(sdl, nil)
			if err == nil {
				t.Fatalf("enum value %q was accepted, want a parse error", name)
			}
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("error %q does not mention enum value %q", err, name)
			}
		})
	}
}
