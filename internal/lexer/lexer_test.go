package lexer_test

import (
	"fmt"
	"strings"
	"testing"
	"text/scanner"

	"reflect"

	"github.com/neelance/graphql-go/internal/lexer"
)

func TestLexer(t *testing.T) {
	cases := map[string]struct {
		given    string
		expected []interface{}
	}{
		"array of strings": {
			given:    `String ["a","b","c",""]`,
			expected: []interface{}{"a", "b", "c", ""},
		},
		"array of strings with spaces": {
			given:    `String ["a", "b" ,      "c",""    ]`,
			expected: []interface{}{"a", "b", "c", ""},
		},
		"array of integers": {
			given: `Int	[1,2,44,550125]`,
			expected: []interface{}{1, 2, 44, 550125},
		},
		"array of floats": {
			given: `Float	[0,0.5,1,1.2,55,4.012]`,
			expected: []interface{}{0.0, 0.5, 1.0, 1.2, 55.0, 4.012},
		},
	}
	readArray := func(l *lexer.Lexer, fn func(*lexer.Lexer)) {
		l.ConsumeToken('[')
		for l.Peek() != ']' {
			fn(l)
			if l.Peek() == ',' {
				l.ConsumeToken(',')
			}
		}
		l.ConsumeToken(']')
	}
	for hint, c := range cases {
		t.Run(hint, func(t *testing.T) {
			s := &scanner.Scanner{
				Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
			}
			s.Init(strings.NewReader(c.given))
			l := lexer.New(s)
			got := []interface{}{}

			err := l.CatchSyntaxError(func() {
				for l.Peek() != scanner.EOF {
					switch keyword := l.ConsumeIdent(); keyword {
					case "String":
						readArray(l, func(l *lexer.Lexer) {
							got = append(got, l.ConsumeString())
						})
					case "Int":
						readArray(l, func(l *lexer.Lexer) {
							got = append(got, l.ConsumeInt())
						})
					case "Float":
						readArray(l, func(l *lexer.Lexer) {
							got = append(got, l.ConsumeFloat())
						})
					default:
						l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "String", "Float", "Integer"`, keyword))
					}
				}
			})
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			if !reflect.DeepEqual(c.expected, got) {
				t.Errorf("wrong output, expected %v but got %v", c.expected, got)
			}
		})
	}

}
