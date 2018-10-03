package schema

import (
	"testing"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

func TestParseInterfaceDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *Interface
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses simple interface",
		definition:  "Greeting { field: String }",
		expected:    &Interface{Name: "Greeting", Fields: []*Field{{Name: "field"}}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *Interface
			lex := setup(t, test.definition)

			parse := func() { actual = parseInterfaceDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareInterfaces(t, test.expected, actual)
		})
	}
}

// TestParseObjectDef tests the logic for parsing object types from the schema definition as
// written in `parseObjectDef()`.
func TestParseObjectDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *Object
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses type inheriting single interface",
		definition:  "Hello implements World { field: String }",
		expected:    &Object{Name: "Hello", interfaceNames: []string{"World"}},
	}, {
		description: "Parses type inheriting multiple interfaces",
		definition:  "Hello implements Wo & rld { field: String }",
		expected:    &Object{Name: "Hello", interfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Parses type inheriting multiple interfaces with leading ampersand",
		definition:  "Hello implements & Wo & rld { field: String }",
		expected:    &Object{Name: "Hello", interfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Allows legacy SDL interfaces",
		definition:  "Hello implements Wo, rld { field: String }",
		expected:    &Object{Name: "Hello", interfaceNames: []string{"Wo", "rld"}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *Object
			lex := setup(t, test.definition)

			parse := func() { actual = parseObjectDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareObjects(t, test.expected, actual)
		})
	}
}

func compareErrors(t *testing.T, expected, actual *errors.QueryError) {
	t.Helper()

	switch {
	case expected != nil && actual != nil:
		if expected.Message != actual.Message {
			t.Fatalf("wanted error message %q, got %q", expected.Message, actual.Message)
		}
		// TODO: Check error locations are as expected.

	case expected != nil && actual == nil:
		t.Fatalf("missing expected error: %q", expected)

	case expected == nil && actual != nil:
		t.Fatalf("got unexpected error: %q", actual)
	}
}

func compareInterfaces(t *testing.T, expected, actual *Interface) {
	t.Helper()

	// TODO: We can probably extract this switch statement into its own function.
	switch {
	case expected == nil && actual == nil:
		return
	case expected == nil && actual != nil:
		t.Fatalf("wanted nil, got an unexpected result: %#v", actual)
	case expected != nil && actual == nil:
		t.Fatalf("wanted non-nil result, got nil")
	}

	if expected.Name != actual.Name {
		t.Errorf("wrong interface name: want %q, got %q", expected.Name, actual.Name)
	}

	if len(expected.Fields) != len(actual.Fields) {
		t.Fatalf("wanted %d field definitions, got %d", len(expected.Fields), len(actual.Fields))
	}

	for i, f := range expected.Fields {
		if f.Name != actual.Fields[i].Name {
			t.Errorf("fields[%d]: wrong field name: want %q, got %q", i, f.Name, actual.Fields[i].Name)
		}
	}
}

func compareObjects(t *testing.T, expected, actual *Object) {
	t.Helper()

	switch {
	case expected == nil && expected == actual:
		return
	case expected == nil && actual != nil:
		t.Fatalf("wanted nil, got an unexpected result: %#v", actual)
	case expected != nil && actual == nil:
		t.Fatalf("wanted non-nil result, got nil")
	}

	if expected.Name != actual.Name {
		t.Errorf("wrong object name: want %q, got %q", expected.Name, actual.Name)
	}

	if len(expected.interfaceNames) != len(actual.interfaceNames) {
		t.Fatalf(
			"wrong number of interface names: want %s, got %s",
			expected.interfaceNames,
			actual.interfaceNames,
		)
	}

	for i, expectedName := range expected.interfaceNames {
		actualName := actual.interfaceNames[i]
		if expectedName != actualName {
			t.Errorf("wrong interface name: want %q, got %q", expectedName, actualName)
		}
	}
}

func setup(t *testing.T, def string) *common.Lexer {
	t.Helper()

	lex := common.NewLexer(def, false)
	lex.Consume(true)

	return lex
}
