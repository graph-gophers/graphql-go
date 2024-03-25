package validation_test

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"encoding/json"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/internal/validation"
)

type Test struct {
	Name   string
	Rule   string
	Schema int
	Query  string
	Vars   map[string]interface{}
	Errors []*errors.QueryError
}

func TestValidate(t *testing.T) {
	skip := map[string]struct{}{
		// The meta schema always includes the standard types, so this isn't applicable
		"Validate: Known type names/references to standard scalars that are missing in schema": {},
		// Ignore tests using experimental @stream
		"Validate: Overlapping fields can be merged/different stream directive label":               {},
		"Validate: Overlapping fields can be merged/different stream directive initialCount":        {},
		"Validate: Overlapping fields can be merged/different stream directive first missing args":  {},
		"Validate: Overlapping fields can be merged/different stream directive second missing args": {},
		"Validate: Overlapping fields can be merged/different stream directive extra argument":      {},
		"Validate: Overlapping fields can be merged/mix of stream and no stream":                    {},
	}

	f, err := os.Open("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData struct {
		Schemas []string
		Tests   []*Test
	}
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		t.Fatal(err)
	}

	schemas := make([]*ast.Schema, len(testData.Schemas))
	for i, schemaStr := range testData.Schemas {
		s := schema.New()

		s.Directives["oneOf"] = &ast.DirectiveDefinition{
			// graphql-js includes support for @oneOf, currently in RFC
			// This is not available in graphql-go, nor is it expected to be unless the RFC is accepted
			// See https://github.com/graphql/graphql-js/pull/3513 & https://github.com/graphql/graphql-spec/pull/825/
			Name:      "oneOf",
			Desc:      "Indicates exactly one field must be supplied and this field must not be `null`.",
			Locations: []string{"INPUT_OBJECT"},
		}

		err := schema.Parse(s, schemaStr, false)
		if err != nil {
			t.Fatal(err)
		}
		schemas[i] = s
	}

	for _, test := range testData.Tests {
		t.Run(test.Name, func(t *testing.T) {
			if _, ok := skip[test.Name]; ok {
				t.Skip("Test ignored")
			}

			d, err := query.Parse(test.Query)
			if err != nil {
				t.Fatalf("failed to parse query: %s", err)
			}
			errs := validation.Validate(schemas[test.Schema], d, test.Vars, 0)
			got := []*errors.QueryError{}
			for _, err := range errs {
				// graphql-js adjusted its rule naming. Preserve the existing naming to minimise changes for now
				if rule := strings.TrimSuffix(test.Rule, "Rule"); err.Rule == rule {
					err.Rule = ""
					got = append(got, err)
				}
			}
			sortLocations(test.Errors)
			sortLocations(got)
			if !reflect.DeepEqual(test.Errors, got) {
				t.Errorf("wrong errors for rule %q\nexpected: %v\ngot:      %v", test.Rule, test.Errors, got)
			}
		})
	}
}

func sortLocations(errs []*errors.QueryError) {
	for _, err := range errs {
		locs := err.Locations
		sort.Slice(locs, func(i, j int) bool { return locs[i].Before(locs[j]) })
	}

	// FIXME: Sorting errors seems necessary, as mutations being validated before queries, but order reversed in expectations?
	sort.Slice(errs, func(i, j int) bool { return errs[i].Locations[0].Before(errs[j].Locations[0]) })
}
