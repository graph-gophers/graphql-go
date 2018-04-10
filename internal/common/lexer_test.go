package common_test

import (
	"testing"

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
,,,,,, # Commas are insignificant
type Hello {
	world: String!
}`,
	expected: "Comment line 1\nComment line 2\nCommas are insignificant",
}}

func TestConsume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.description, func(t *testing.T) {
			lex := common.NewLexer(test.definition)

			err := lex.CatchSyntaxError(lex.Consume)
			if err != nil {
				t.Fatal(err)
			}

			if test.expected != lex.DescComment() {
				t.Errorf("wrong description value:\nwant: %q\ngot : %q", test.expected, lex.DescComment())
			}
		})
	}
}
