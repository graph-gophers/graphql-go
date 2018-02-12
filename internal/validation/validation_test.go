package validation_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"encoding/json"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/internal/validation"
)

type Test struct {
	Name   string
	Rule   string
	Schema int
	Query  string
	Errors []*errors.QueryError
}

func TestValidation(t *testing.T) {
	f, err := os.Open("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData struct {
		Schemas []string
		Tests   []*Test
	}
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		t.Fatalf("test data JSON decode failed: %v", err)
	}

	schemas := make([]*schema.Schema, len(testData.Schemas))
	for i, schemaStr := range testData.Schemas {
		schemas[i] = schema.New()
		if err := schemas[i].Parse(schemaStr); err != nil {
			t.Fatalf("schema parse failed: %v", err)
		}
	}

	for _, test := range testData.Tests {
		t.Run(test.Name, func(t *testing.T) {
			d, err := query.Parse(test.Query)
			if err != nil {
				t.Fatalf("query parse failed: %v", err)
			}
			errs := validation.Validate(schemas[test.Schema], d)
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
				t.Errorf("wrong errors\nexpected: %v\ngot:      %v", test.Errors, got)
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
