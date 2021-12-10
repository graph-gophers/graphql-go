package common_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go/internal/common"
)

type consumeTestCase struct {
	description           string
	definition            string
	expected              string // expected description
	failureExpected       bool
	useStringDescriptions bool
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
	expected:              "Comment line 1\nComment line 2\nCommas are insignificant",
	useStringDescriptions: false,
}, {
	description: "simple string descriptions allowed in new mode",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
"New style comments"
type Hello {
	world: String!
}`,
	expected:              "New style comments",
	useStringDescriptions: true,
}, {
	description: "comment after description works",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
type Hello {
	world: String!
}`,
	expected:              "",
	useStringDescriptions: true,
}, {
	description: "triple quote descriptions allowed in new mode",
	definition: `

# Comment line 1
#Comment line 2
,,,,,, # Commas are insignificant
"""
New style comments
Another line
"""
type Hello {
	world: String!
}`,
	expected:              "New style comments\nAnother line",
	useStringDescriptions: true,
}}

func TestConsume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.description, func(t *testing.T) {
			lex := common.NewLexer(test.definition, test.useStringDescriptions)

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

var multilineStringTests = []consumeTestCase{
	{
		description:           "Oneline strings are okay",
		definition:            `"Hello World"`,
		expected:              "",
		failureExpected:       false,
		useStringDescriptions: true,
	},
	{
		description: "Multiline strings are not allowed",
		definition: `"Hello
				 World"`,
		expected:              `graphql: syntax error: literal not terminated (line 1, column 1)`,
		failureExpected:       true,
		useStringDescriptions: true,
	},
}

func TestMultilineString(t *testing.T) {
	for _, test := range multilineStringTests {
		t.Run(test.description, func(t *testing.T) {
			lex := common.NewLexer(test.definition, test.useStringDescriptions)

			err := lex.CatchSyntaxError(func() { lex.ConsumeWhitespace() })
			if test.failureExpected && err == nil {
				t.Fatalf("Test '%s' should fail", test.description)
			} else if test.failureExpected && err != nil {
				if test.expected != err.Error() {
					t.Fatalf("Test '%s' failed with wrong error: '%s'. Error should be: '%s'", test.description, err.Error(), test.expected)
				}
			}

			if !test.failureExpected && err != nil {
				t.Fatalf("Test '%s' failed with error: '%s'", test.description, err.Error())
			}
		})
	}
}
