package lexer_test

import (
	"strings"
	"testing"
	"text/scanner"

	"strconv"

	"github.com/engoengine/math"
	"github.com/neelance/graphql-go/internal/lexer"
)

func TestLexer_ConsumeFloat(t *testing.T) {
	cases := map[string]struct {
		given    string
		expected float64
	}{
		"zero": {
			given:    "0",
			expected: 0.0,
		},
		"regular": {
			given:    "1.5",
			expected: 1.5,
		},
		"max-int64": {
			given:    strconv.FormatInt(math.MaxInt64, 10),
			expected: float64(math.MaxInt64),
		},
		"max-float64": {
			given:    strconv.FormatFloat(math.MaxFloat64, 'g', 100000000, 64),
			expected: math.MaxFloat64,
		},
		"smallest-nonzero-float64": {
			given:    strconv.FormatFloat(math.SmallestNonzeroFloat64, 'g', 100000000, 64),
			expected: math.SmallestNonzeroFloat64,
		},
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
