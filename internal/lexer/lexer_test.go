package lexer_test

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

func TestLexer_ConsumeFloat(t *testing.T) {
	cases := map[string]struct {
		given    string
		expected float64
	}{
		"integer": {given: "0", expected: 0.0},
		"decimal": {given: "1.5", expected: 1.5},
	}

	for hint, c := range cases {
		t.Run(hint, func(t *testing.T) {
			s := &scanner.Scanner{}
			s.Init(strings.NewReader(c.given))
			l := lexer.New(s)
			var got float64

			err := l.CatchSyntaxError(func() {
				got = l.ConsumeFloat()
			})
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			if c.expected != got {
				t.Errorf("wrong output, expected %f but got %f", c.expected, got)
			}
		})
	}
}
