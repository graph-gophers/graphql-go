package validation_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	v "github.com/graph-gophers/graphql-go/internal/validation"
)

// FuzzValidateOverlapMixed exercises the overlap validation logic with randomly generated queries
// containing many sibling fields and fragment spreads to ensure it does not panic or explode in memory.
// It uses a modest overlap pair cap to keep each iteration bounded.
func FuzzValidateOverlapMixed(f *testing.F) {
	baseQueries := []string{
		"query{root{id}}",
		"query Q{root{id name}}",
	}
	for _, q := range baseQueries {
		f.Add(q)
	}

	s := schema.New()
	_ = schema.Parse(s, `schema{query:Query} type Query{root: Thing} type Thing { id: ID name: String value: String }`, false)

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	f.Fuzz(func(t *testing.T, seed string) {
		// Use hash of seed to deterministically generate but bound complexity.
		r := rand.New(rand.NewSource(int64(len(seed)) + randSource.Int63()))
		fieldCount := 50 + r.Intn(150) // 50-199
		fragCount := 1 + r.Intn(5)

		// Build fragments.
		fragBodies := make([]string, fragCount)
		for i := 0; i < fragCount; i++ {
			// each fragment gets subset of fields
			var body string
			innerFields := 5 + r.Intn(20)
			for j := 0; j < innerFields; j++ {
				body += " f" + nameIdx(r.Intn(500)) + ":id"
			}
			fragBodies[i] = "fragment F" + nameIdx(i) + " on Thing{" + body + " }"
		}

		// Root selection
		sel := "query{root{"
		for i := 0; i < fieldCount; i++ {
			sel += " a" + nameIdx(r.Intn(1000)) + ":id"
		}
		// Sprinkle fragment spreads
		for i := 0; i < fragCount; i++ {
			sel += " ...F" + nameIdx(i)
		}
		sel += "}}"
		queryText := sel
		for _, fb := range fragBodies {
			queryText += fb
		}

		doc, err := query.Parse(queryText)
		if err != nil {
			return
		} // parser fuzzing not our goal
		if len(doc.Operations) == 0 {
			return
		}
		// Use overlap limit to bound cost.
		errs := v.Validate(s, doc, nil, 0, 10_000)
		// Ensure no panic (implicit). Optionally sanity check: errors slice must not be ridiculously huge.
		if len(errs) > 1000 {
			t.Fatalf("too many errors: %d", len(errs))
		}
	})
}

func nameIdx(i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	if i < len(letters) {
		return string(letters[i])
	}
	return string(letters[i%len(letters)]) + nameIdx(i/len(letters))
}
