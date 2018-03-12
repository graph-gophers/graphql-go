package schema

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/internal/common"
)

type testCase struct {
	description string
	declaration string
	expected    *Object
}

func TestParseObjectDeclaration(t *testing.T) {
	tests := []testCase{
		{
			"allows '&' separator",
			"Alien implements Being & Intelligent { name: String, iq: Int }",
			&Object{
				Name:           "Alien",
				interfaceNames: []string{"Being", "Intelligent"},
			},
		},
		{
			"allows legacy ',' separator",
			"Alien implements Being, Intelligent { name: String, iq: Int }",
			&Object{
				Name:           "Alien",
				interfaceNames: []string{"Being", "Intelligent"},
			},
		},
	}

	setup := func(schema string) *common.Lexer {
		sc := &scanner.Scanner{
			Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
		}
		sc.Init(strings.NewReader(schema))
		return common.NewLexer(sc)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			lex := setup(test.declaration)
			var actual *Object

			parse := func() { actual = parseObjectDeclaration(lex) }
			if err := lex.CatchSyntaxError(parse); err != nil {
				t.Fatal(err)
			}

			if test.expected.Name != actual.Name {
				t.Errorf("wrong object name: want %q, got %q", test.expected.Name, actual.Name)
			}

			if len(test.expected.interfaceNames) != len(actual.interfaceNames) {
				t.Fatalf("wrong number of interface names: want %s, got %s", test.expected.interfaceNames, actual.interfaceNames)
			}

			for i, expectedName := range test.expected.interfaceNames {
				actualName := actual.interfaceNames[i]
				if expectedName != actualName {
					t.Errorf("wrong interface name: want %q, got %q", expectedName, actualName)
				}
			}
		})
	}
}
