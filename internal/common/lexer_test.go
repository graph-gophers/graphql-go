package common_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/internal/common"
)

type consumeTestCase struct {
	description              string
	definition               string
	expected                 string // expected description
	failureExpected          bool
	noCommentsAsDescriptions bool
}

// Note that these tests stop as soon as they parse the comments, so even though the rest of the file will fail to parse sometimes, the tests still pass
var consumeTests = []consumeTestCase{{
	description: "no string descriptions allowed in old mode",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
"New style comments"
type Hello {
	world: String!
}`,
	expected:                 "Comment line 1\nComment line 2\nCommas are insignificant",
	noCommentsAsDescriptions: false,
}, {
	description: "simple string descriptions allowed in old mode",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
"New style comments"
type Hello {
	world: String!
}`,
	expected:                 "New style comments",
	noCommentsAsDescriptions: true,
}, {
	description: "triple quote descriptions allowed in old mode",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
"""
New style comments
"""
type Hello {
	world: String!
}`,
	expected:                 "New style comments",
	noCommentsAsDescriptions: true,
}}

func TestConsume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.description, func(t *testing.T) {
			lex := common.NewLexer(test.definition, test.noCommentsAsDescriptions)

			err := lex.CatchSyntaxError(func() { lex.ConsumeWhitespace() })
			if test.failureExpected {
				if err == nil {
					t.Fatalf("schema should have been invalid; comment: %s", lex.DescComment())
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
			}

			if test.expected != lex.DescComment() {
				t.Errorf("wrong description value:\nwant: %q\ngot : %q", test.expected, lex.DescComment())
			}
		})
	}
}
