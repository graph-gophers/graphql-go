package common_test

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/internal/common"
)

type consumeTestCase struct {
	description string
	definition  string
	expected    string // expected description
}

var consumeTests = []consumeTestCase{{
	description: "initial test",
	definition: `

# Comment line 1
# Comment line 2
type Hello {
world: String!
}`,
	expected: "Comment line 1\nComment line 2",
}}

func TestConsume(t *testing.T) {
	setup := func(t *testing.T, def string) *common.Lexer {
		t.Helper()

		sc := &scanner.Scanner{
			Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
		}
		sc.Init(strings.NewReader(def))
		return common.NewLexer(sc)
	}

	for _, test := range consumeTests {
		t.Run(test.description, func(t *testing.T) {
			lex := setup(t, test.definition)

			lex.Consume()

			if test.expected != lex.DescComment() {
				t.Errorf("wanted: %q\ngot: %q", test.expected, lex.DescComment())
			}
		})
	}
}
