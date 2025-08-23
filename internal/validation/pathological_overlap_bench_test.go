//go:build bench
// +build bench

package validation_test

import (
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/internal/validation"
)

// Builds a query with n repetitions of the same response key whose underlying field differs
// every other occurrence, forcing overlap validation to perform many pairwise comparisons if
// optimization is ineffective. Used to evaluate the adaptive identical-field fast path.
func buildPathologicalQuery(n int) string {
	// Alternate between a: realField and a: otherField to trigger conflicts only in half pairs.
	body := "{\n"
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			body += "  x: fieldA { leaf }\n"
		} else {
			body += "  x: fieldB { leaf }\n"
		}
	}
	body += "}\n"
	return body
}

var pathologicalSchemaSDL = `
  type Query { fieldA: Obj fieldB: Obj }
  type Obj { leaf: Int }
`

func preparePathologicalSchema(b *testing.B) *ast.Schema {
	s := schema.New()
	if err := schema.Parse(s, pathologicalSchemaSDL, false); err != nil {
		b.Fatalf("schema parse: %v", err)
	}
	return s
}

// BenchmarkPathologicalOverlap stresses the worst-case alias group: repeated identical response key
// with alternating underlying field definitions. Without the identical-fields fast path, this trends
// toward O(k^2) pairwise comparisons; with optimization it aims for near O(k) (shallow pass + hashes).
func BenchmarkPathologicalOverlap(b *testing.B) {
	s := preparePathologicalSchema(b)
	cases := []int{10, 50, 100, 250, 500}
	for _, n := range cases {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			queryStr := buildPathologicalQuery(n)
			for b.Loop() {
				doc, err := query.Parse(queryStr)
				if err != nil {
					b.Fatalf("parse: %v", err)
				}
				_ = validation.Validate(s, doc, nil, 0)
			}
		})
	}
}
