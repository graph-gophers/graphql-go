package schema_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/internal/schema"
)

type parseTestCase struct {
	description string
	sdl         string
	expected    *schema.Schema
	err         error
}

var parseTests = []parseTestCase{{
	description: "Parses interface definition",
	sdl:         "interface Greeting { message: String! }",
	expected: &schema.Schema{
		Types: map[string]schema.NamedType{
			"Greeting": &schema.Interface{
				Name:   "Greeting",
				Fields: []*schema.Field{{Name: "message"}},
			}},
	}}, {
	description: "Parses type with description string",
	sdl: `
	"Single line description."
	type Type {
		field: String
	}`,
	expected: &schema.Schema{
		Types: map[string]schema.NamedType{
			"Type": &schema.Object{
				Name: "Type",
				Desc: "Single line description.",
			}},
	}}, {
	description: "Parses type with multi-line description string",
	sdl: `
	"""
	Multi-line description.
	"""
	type Type {
		field: String
	}`,
	expected: &schema.Schema{
		Types: map[string]schema.NamedType{
			"Type": &schema.Object{
				Name: "Type",
				Desc: "Multi-line description.",
			}},
	}}, {
	description: "Parses type with multi-line description and ignores comments",
	sdl: `
	"""
	Multi-line description with ignored comments.
	"""
	# This comment should be ignored.
	type Type {
		field: String
	}`,
	expected: &schema.Schema{
		Types: map[string]schema.NamedType{
			"Type": &schema.Object{
				Name: "Type",
				Desc: "Multi-line description with ignored comments.",
			}},
	}},
}

func TestParse(t *testing.T) {
	setup := func(t *testing.T) *schema.Schema {
		t.Helper()
		return schema.New()
	}

	for _, test := range parseTests {
		t.Run(test.description, func(t *testing.T) {
			t.Skip("TODO: add support for descriptions")
			schema := setup(t)

			err := schema.Parse(test.sdl, false)
			if err != nil {
				t.Fatal(err)
			}

			// TODO: verify schema is the same as expected.
		})
	}
}
