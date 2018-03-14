package schema

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

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

	setup := func(def string) *common.Lexer {
		sc := &scanner.Scanner{
			Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
		}
		sc.Init(strings.NewReader(def))

		lex := common.NewLexer(sc)
		lex.Consume()

		return lex
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *Object
			lex := setup(test.definition)

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

func compareObjects(t *testing.T, expected, actual *Object) {
	t.Helper()

	switch {
	case expected == nil && expected == actual:
		return
	case expected == nil && actual != nil:
		t.Fatalf("got an unexpected object: %#v", actual)
	case expected != nil && actual == nil:
		t.Fatalf("wanted non-nil object, got nil")
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
