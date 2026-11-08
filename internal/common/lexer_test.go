package common_test

import (
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
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

func TestLexerRejectsDotPrefixedNumber(t *testing.T) {
	for _, input := range []string{".123", "-.5", "-.5e2"} {
		t.Run(input, func(t *testing.T) {
			lex := common.NewLexer(input, false)

			err := lex.CatchSyntaxError(func() {
				lex.ConsumeWhitespace()
			})
			if err == nil {
				t.Fatal("expected syntax error for a number without an integer part")
			}
		})
	}
}

func TestLexerRejectsNegativeDotPrefixedNumberAfterToken(t *testing.T) {
	lex := common.NewLexer("value # comment\n -.5", false)
	err := lex.CatchSyntaxError(func() {
		lex.ConsumeWhitespace()
		lex.ConsumeToken(scanner.Ident)
	})
	if err == nil {
		t.Fatal("expected syntax error for a number without an integer part")
	}
}

func TestLexerAcceptsNumbersWithIntegerPart(t *testing.T) {
	for _, input := range []string{"0", "-0", "0.123", "1e3", "-0.5", "-1", "-1e3", "9223372036854775807", "-9223372036854775808", "1.7976931348623157e308"} {
		t.Run(input, func(t *testing.T) {
			lex := common.NewLexer(input, false)
			err := lex.CatchSyntaxError(func() {
				lex.ConsumeWhitespace()
				value := common.ParseLiteral(lex, false)
				lit, ok := value.(*ast.PrimitiveValue)
				if !ok || lit.Text != input {
					t.Fatalf("got literal %#v, want %q", value, input)
				}
				if lex.Peek() != scanner.EOF {
					t.Fatalf("unexpected token after %q", input)
				}
			})
			if err != nil {
				t.Fatalf("unexpected syntax error for %q: %v", input, err)
			}
		})
	}
}

func TestParseLiteralRejectsInvalidNumbers(t *testing.T) {
	for _, input := range []string{
		"0x10", "0b11", "0o17", "1_0", "1.0_0", "0x1p2",
		"01", "1.", "9223372036854775808", "-9223372036854775809",
		"1e999", "-1e999", "-true", `-"text"`, "-]",
	} {
		t.Run(input, func(t *testing.T) {
			lex := common.NewLexer(input, false)
			err := lex.CatchSyntaxError(func() {
				lex.ConsumeWhitespace()
				common.ParseLiteral(lex, false)
			})
			if err == nil {
				t.Fatalf("expected syntax error for %q", input)
			}
		})
	}
}

func TestParseLiteralRejectsNonNumberAfterMinus(t *testing.T) {
	lex := common.NewLexer("-true", false)
	err := lex.CatchSyntaxError(func() {
		lex.ConsumeWhitespace()
		common.ParseLiteral(lex, false)
	})
	if err == nil || err.Message != "syntax error: expected number" {
		t.Fatalf("expected number syntax error, got %v", err)
	}
}
