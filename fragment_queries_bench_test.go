//go:build bench
// +build bench

package graphql_test

// Benchmark inspired by historical PR #102 ("Control memory explosion on large list of queries").
// The validation phase used to exhibit O(n^2) memory/time growth for large
// numbers of root selections due to pairwise overlap checks. This benchmark
// recreates a simplified version focusing on many aliased root fields to
// exercise overlap validation. Two scenarios are measured:
//   * overlapping aliases   (all fields share the same alias name)
//   * non-overlapping aliases (each field has a unique alias)
// The overlapping case forces the validator to consider merging, potentially
// increasing work. Non-overlapping aliases avoid that specific merge path.
//
// Larger counts (5000, 10000) are included only when BIG_FRAGMENT_BENCH=1 is
// set in the environment to keep default benchmarks fast.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

const fragmentBenchSchema = `
    type Query { a: Int! }
`

type fragmentBenchResolver struct{}

func (fragmentBenchResolver) A() int32 { return 1 }

func buildAliasQuery(count int, nonOverlap bool) string {
	// Build a single operation with many aliased uses of the same field.
	// Example (nonOverlap, count=3):
	// query Q { f0: a f1: a f2: a }
	// Example (overlap, count=3):
	// query Q { x: a x: a x: a }
	q := "query Q {"
	if nonOverlap {
		for i := 0; i < count; i++ {
			q += fmt.Sprintf(" f%d: a", i)
		}
	} else {
		for i := 0; i < count; i++ {
			q += " x: a" // same alias each time
		}
	}
	q += " }"
	return q
}

// BenchmarkSimpleRootAlias measures validation/exec cost for large flat selection sets
// with and without overlapping aliases.
func BenchmarkSimpleRootAlias(b *testing.B) {
	schema := graphql.MustParseSchema(fragmentBenchSchema, &fragmentBenchResolver{})

	counts := []int{1, 10, 100, 500, 1000}
	if os.Getenv("BIG_FRAGMENT_BENCH") == "1" {
		counts = append(counts, 5000, 10000)
	}

	ctx := context.Background()

	for _, c := range counts {
		for _, nonOverlap := range []bool{false, true} {
			aliasMode := "overlapping"
			if nonOverlap {
				aliasMode = "non-overlapping"
			}
			queryStr := buildAliasQuery(c, nonOverlap)
			// Warm-up single execution (outside timing) to catch schema issues early.
			if resp := schema.Exec(ctx, queryStr, "", nil); len(resp.Errors) > 0 {
				b.Fatalf("unexpected exec errors preparing benchmark: %v", resp.Errors)
			}
			b.Run(fmt.Sprintf("%d_queries_%s_aliases", c, aliasMode), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if resp := schema.Exec(ctx, queryStr, "", nil); len(resp.Errors) > 0 {
						b.Fatalf("exec errors: %v", resp.Errors)
					}
				}
			})
		}
	}
}

func TestOverlappingAlias(t *testing.T) {
	query := `
		{
			hero(episode: EMPIRE) {
				a: name
				a: id
			}
		}
	`
	result := starwarsSchema.Exec(context.Background(), query, "", nil)
	if len(result.Errors) == 0 {
		t.Fatal("Expected error from overlapping alias")
	}
}

// go test -bench=FragmentQueries -benchmem
// BenchmarkStarWarsFragmentAlias expands the scenario with fragment spreads on the
// canonical StarWars schema exercising deeper selection hashing paths.
func BenchmarkStarWarsFragmentAlias(b *testing.B) {
	singleQuery := `
		composed_%d: hero(episode: EMPIRE) {
			name
			...friendsNames
			...friendsIds
		}
	`

	queryTemplate := `
		{
			%s
		}
		fragment friendsNames on Character {
			friends {
				name
			}
		}
		fragment friendsIds on Character {
			friends {
				id
			}
		}
	`

	testCases := []int{
		1,
		10,
		100,
		1000,
		10000,
	}

	for _, c := range testCases {
		// for each count, add a case for overlapping aliases vs non-overlapping aliases
		for _, o := range []bool{true} {
			var buffer bytes.Buffer
			for i := 0; i < c; i++ {
				idx := 0
				if o {
					idx = i
				}
				buffer.WriteString(fmt.Sprintf(singleQuery, idx))
			}

			query := fmt.Sprintf(queryTemplate, buffer.String())
			a := "overlapping"
			if o {
				a = "non-overlapping"
			}
			b.Run(fmt.Sprintf("%d queries %s aliases", c, a), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					result := starwarsSchema.Exec(context.Background(), query, "", nil)
					if len(result.Errors) != 0 {
						b.Fatal(result.Errors[0])
					}
				}
			})
		}
	}
}

// Performance mitigation roadmap (see discussion):
// 1. Implement "early exit after first conflict" in field overlap validation.
//    - Spec allows returning after first conflict; minimizes quadratic blow-up.
//    - Lowest risk change; preserves correctness, may slightly reduce number of
//      reported sibling conflicts (acceptable trade-off for widely used lib).
// 2. (Optional, behind internal constant) Threshold-based fallback: if conflicts
//    in a single comparison group exceed a limit, short‑circuit with one
//    aggregate error to cap worst-case cost.
// 3. Defer heavier changes (fragment-aware hashing, canonical structural
//    deduplication) until profiling shows residual hot spots; these add
//    complexity and require careful determinism/error messaging review.
