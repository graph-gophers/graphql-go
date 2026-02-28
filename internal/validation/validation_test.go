package validation_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/internal/validation"
)

type Schema struct {
	ID  string
	SDL string
}

type Test struct {
	Name   string
	Rule   string
	Schema string
	Query  string
	Vars   map[string]any
	Errors []*errors.QueryError
}

var skippedValidationTests = map[string]struct{}{
	// known-rule-overlap: extra error under PossibleFragmentSpreadsRule.
	"Validate: Possible fragment spreads/ignores incorrect type (caught by FragmentsOnCompositeTypesRule)": {},
	// parser-mismatch: graphql-js fixture parses SDL as executable query.
	"Validate: Directives Are Unique Per Location/unknown directives must be ignored": {},
	"Validate: Executable definitions/with schema definition":                         {},
	"Validate: Executable definitions/with type definition":                           {},
	// schema-difference: meta schema always includes standard scalar types.
	"Validate: Known type names/references to standard scalars that are missing in schema": {},
	// unsupported-feature: experimental @stream directives.
	"Validate: Overlapping fields can be merged/different stream directive label":               {},
	"Validate: Overlapping fields can be merged/different stream directive initialCount":        {},
	"Validate: Overlapping fields can be merged/different stream directive first missing args":  {},
	"Validate: Overlapping fields can be merged/different stream directive second missing args": {},
	"Validate: Overlapping fields can be merged/different stream directive extra argument":      {},
	"Validate: Overlapping fields can be merged/mix of stream and no stream":                    {},
	// unresolved-scalar-parse-literal: graphql-js can reject literal via scalar parseLiteral(undefined) behavior.
	"Validate: Values of correct type/Invalid input object value/reports error for custom scalar that returns undefined": {},
}

func TestValidate(t *testing.T) {
	skip := skippedValidationTests

	f, err := os.Open("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData struct {
		Schemas []Schema
		Tests   []*Test
	}
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		t.Fatal(err)
	}

	schemas := make(map[string]*ast.Schema, len(testData.Schemas))
	for _, sc := range testData.Schemas {
		s := schema.New()
		err := schema.Parse(s, sc.SDL, false)
		if err != nil {
			t.Fatal(err)
		}
		schemas[sc.ID] = s
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
			errs := validation.Validate(schemas[test.Schema], d, test.Vars, 0, 0)
			got := []*errors.QueryError{}
			for _, err := range errs {
				if err.Rule == test.Rule {
					err.Rule = ""
					got = append(got, err)
				}
			}
			normalizeErrors(test.Errors)
			normalizeErrors(got)
			if !reflect.DeepEqual(test.Errors, got) {
				t.Errorf("wrong errors for rule %q\nexpected: %v\ngot:      %v", test.Rule, test.Errors, got)
			}
		})
	}
}

func normalizeErrors(errs []*errors.QueryError) {
	for _, err := range errs {
		locs := err.Locations
		sort.Slice(locs, func(i, j int) bool { return locs[i].Before(locs[j]) })
	}

	sort.Slice(errs, func(i, j int) bool {
		if errs[i].Message != errs[j].Message {
			return errs[i].Message < errs[j].Message
		}

		il := errs[i].Locations
		jl := errs[j].Locations
		if len(il) == 0 || len(jl) == 0 {
			return len(il) < len(jl)
		}

		if il[0].Line != jl[0].Line {
			return il[0].Line < jl[0].Line
		}
		if il[0].Column != jl[0].Column {
			return il[0].Column < jl[0].Column
		}

		return len(il) < len(jl)
	})
}
