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

func TestValidate(t *testing.T) {
	skip := map[string]struct{}{
		// Minor issue: reporting extra error under PossibleFragmentSpreadsRule which is not intended
		"Validate: Possible fragment spreads/ignores incorrect type (caught by FragmentsOnCompositeTypesRule)": {},
		// graphql-js test case parses SDL as if it was a query here, which fails since we only accept a query
		"Validate: Directives Are Unique Per Location/unknown directives must be ignored": {},
		// The meta schema always includes the standard types, so this isn't applicable
		"Validate: Known type names/references to standard scalars that are missing in schema": {},
		// Ignore tests using experimental @stream
		"Validate: Overlapping fields can be merged/different stream directive label":               {},
		"Validate: Overlapping fields can be merged/different stream directive initialCount":        {},
		"Validate: Overlapping fields can be merged/different stream directive first missing args":  {},
		"Validate: Overlapping fields can be merged/different stream directive second missing args": {},
		"Validate: Overlapping fields can be merged/different stream directive extra argument":      {},
		"Validate: Overlapping fields can be merged/mix of stream and no stream":                    {},
		// ValuesOfCorrectTypeRule: Error message format differs from graphql-js. graphql-go includes argument name and type details, while graphql-js uses simpler coercion messages.
		// TODO: Align error message format with graphql-js or update tests.json to match graphql-go's implementation.
		"Validate: Values of correct type/Invalid String values/Int into String":                                                           {},
		"Validate: Values of correct type/Invalid String values/Float into String":                                                         {},
		"Validate: Values of correct type/Invalid String values/Boolean into String":                                                       {},
		"Validate: Values of correct type/Invalid String values/Unquoted String into String":                                               {},
		"Validate: Values of correct type/Invalid Int values/String into Int":                                                              {},
		"Validate: Values of correct type/Invalid Int values/Big Int into Int":                                                             {},
		"Validate: Values of correct type/Invalid Int values/Unquoted String into Int":                                                     {},
		"Validate: Values of correct type/Invalid Int values/Simple Float into Int":                                                        {},
		"Validate: Values of correct type/Invalid Int values/Float into Int":                                                               {},
		"Validate: Values of correct type/Invalid Float values/String into Float":                                                          {},
		"Validate: Values of correct type/Invalid Float values/Boolean into Float":                                                         {},
		"Validate: Values of correct type/Invalid Float values/Unquoted into Float":                                                        {},
		"Validate: Values of correct type/Invalid Boolean value/Int into Boolean":                                                          {},
		"Validate: Values of correct type/Invalid Boolean value/Float into Boolean":                                                        {},
		"Validate: Values of correct type/Invalid Boolean value/String into Boolean":                                                       {},
		"Validate: Values of correct type/Invalid Boolean value/Unquoted into Boolean":                                                     {},
		"Validate: Values of correct type/Invalid ID value/Float into ID":                                                                  {},
		"Validate: Values of correct type/Invalid ID value/Boolean into ID":                                                                {},
		"Validate: Values of correct type/Invalid ID value/Unquoted into ID":                                                               {},
		"Validate: Values of correct type/Invalid Enum value/Int into Enum":                                                                {},
		"Validate: Values of correct type/Invalid Enum value/Float into Enum":                                                              {},
		"Validate: Values of correct type/Invalid Enum value/String into Enum":                                                             {},
		"Validate: Values of correct type/Invalid Enum value/Boolean into Enum":                                                            {},
		"Validate: Values of correct type/Invalid Enum value/Unknown Enum Value into Enum":                                                 {},
		"Validate: Values of correct type/Invalid Enum value/Different case Enum Value into Enum":                                          {},
		"Validate: Values of correct type/Invalid List value/Incorrect item type":                                                          {},
		"Validate: Values of correct type/Invalid List value/Single value of incorrect type":                                               {},
		"Validate: Values of correct type/Invalid non-nullable value/Incorrect value type":                                                 {},
		"Validate: Values of correct type/Invalid non-nullable value/Incorrect value and missing argument (ProvidedRequiredArgumentsRule)": {},
		"Validate: Values of correct type/Invalid non-nullable value/Null value":                                                           {},
		"Validate: Values of correct type/Invalid input object value/Partial object, missing required":                                     {},
		"Validate: Values of correct type/Invalid input object value/Partial object, invalid field type":                                   {},
		"Validate: Values of correct type/Invalid input object value/Partial object, null to non-null field":                               {},
		"Validate: Values of correct type/Invalid input object value/Partial object, unknown field arg":                                    {},
		"Validate: Values of correct type/Invalid input object value/reports error for custom scalar that returns undefined":               {},
		"Validate: Values of correct type/Invalid oneOf input object value/Invalid field type":                                             {},
		"Validate: Values of correct type/Directive arguments/with directive with incorrect types":                                         {},
		"Validate: Values of correct type/Variable default values/variables with invalid default null values":                              {},
		"Validate: Values of correct type/Variable default values/variables with invalid default values":                                   {},
		"Validate: Values of correct type/Variable default values/variables with complex invalid default values":                           {},
		"Validate: Values of correct type/Variable default values/complex variables missing required field":                                {},
		"Validate: Values of correct type/Variable default values/list variables with invalid item":                                        {},
	}

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
}
