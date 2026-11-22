package graphql_test

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestDuplicateRootOperationTypesRejected(t *testing.T) {
	cases := []struct{ name, sdl string }{
		{"duplicate query", `schema { query: Query query: Query } type Query { f: String }`},
		{"extension redefines", `schema { query: Query } extend schema { query: Query } type Query { f: String }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if err == nil {
				t.Fatal("duplicate root operation type was accepted, want an error")
			}
			if !strings.Contains(err.Error(), "query") {
				t.Fatalf("error %q does not name the operation", err)
			}
		})
	}
}
