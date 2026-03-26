package validation_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/internal/validation"
)

const benchmarkSchemaSDL = `
schema {
  query: Query
}

type Query {
  root: Thing
}

type Thing {
  id: ID!
  name: String
  value: String
}
`

var benchErrs any

func BenchmarkValidate(b *testing.B) {
	s := schema.New()
	if err := schema.Parse(s, benchmarkSchemaSDL, false); err != nil {
		b.Fatal(err)
	}

	cases := []struct {
		name  string
		query string
	}{
		{
			name:  "baseline",
			query: `query { root { id name value } }`,
		},
		{
			name:  "overlap-heavy-10",
			query: overlapHeavyQuery(10),
		},
		{
			name:  "overlap-heavy-50",
			query: overlapHeavyQuery(50),
		},
		{
			name:  "overlap-heavy-100",
			query: overlapHeavyQuery(100),
		},
	}

	for _, tc := range cases {
		doc, err := query.Parse(tc.query)
		if err != nil {
			b.Fatalf("parse %q: %v", tc.name, err)
		}

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchErrs = validation.Validate(s, doc, nil, 0, 0)
			}
		})
	}
}

// BenchmarkValidateWorstCaseAliasCollision captures the unchanged O(n^2)
// overlap-validation ceiling by forcing all field selections into the same
// response-name bucket (identical alias), yielding ~n*(n-1)/2 comparisons.
func BenchmarkValidateWorstCaseAliasCollision(b *testing.B) {
	s := schema.New()
	if err := schema.Parse(s, benchmarkSchemaSDL, false); err != nil {
		b.Fatal(err)
	}

	sizes := []int{50, 100, 200, 400}
	for _, n := range sizes {
		doc, err := query.Parse(aliasCollisionQuery(n))
		if err != nil {
			b.Fatalf("parse alias-collision-%d: %v", n, err)
		}

		expectedPairs := n * (n - 1) / 2
		b.Run("alias-collision-"+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchErrs = validation.Validate(s, doc, nil, 0, 0)
			}

			if b.N > 0 && expectedPairs > 0 {
				nsPerPair := float64(b.Elapsed().Nanoseconds()) / float64(b.N*expectedPairs)
				b.ReportMetric(nsPerPair, "ns/pair")
				b.ReportMetric(float64(expectedPairs), "pairs")
			}
		})
	}
}

func overlapHeavyQuery(n int) string {
	var builder strings.Builder
	builder.Grow(64 + n*24)
	builder.WriteString("query { root {")
	for i := range n {
		builder.WriteString(" f")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString(": id")
		builder.WriteString(" f")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString(": name")
	}
	builder.WriteString(" } }")
	return builder.String()
}

func aliasCollisionQuery(n int) string {
	var builder strings.Builder
	builder.Grow(32 + n*6)
	builder.WriteString("query { root {")
	for range n {
		builder.WriteString(" c:id")
	}
	builder.WriteString(" } }")
	return builder.String()
}
