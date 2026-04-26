package query

import (
	"errors"
	"testing"

	gqlerrors "github.com/graph-gophers/graphql-go/errors"
)

// TestParse_UnicodeLexerParserParity covers graphql-js PR3117 Unicode lexer/parser parity behavior.
// Reference: https://github.com/graphql/graphql-js/pull/3117.
func TestParse_UnicodeLexerParserParity(t *testing.T) {
	t.Parallel()

	t.Run("StringAndBlockStringValidCases", func(t *testing.T) {
		t.Parallel()

		valid := []string{
			`query { f(arg: "unescaped unicode outside BMP 😀") }`,
			`query { f(arg: "unescaped maximal unicode outside BMP 􏿿") }`,
			`query { f(arg: "unicode \u{1234}\u{5678}\u{90AB}\u{CDEF}") }`,
			`query { f(arg: "string with unicode escape outside BMP \u{1F600}") }`,
			`query { f(arg: "string with minimal unicode escape \u{0}") }`,
			`query { f(arg: "string with maximal unicode escape \u{10FFFF}") }`,
			`query { f(arg: "string with maximal minimal unicode escape \u{00000000}") }`,
			`query { f(arg: "string with unicode surrogate pair escape \uD83D\uDE00") }`,
			`query { f(arg: "string with minimal surrogate pair escape \uD800\uDC00") }`,
			`query { f(arg: "string with maximal surrogate pair escape \uDBFF\uDFFF") }`,
			`query { f(arg: """unescaped unicode outside BMP 😀""") }`,
		}

		for _, q := range valid {
			t.Run(q, func(t *testing.T) {
				t.Parallel()
				_, err := Parse(q)
				if err != nil {
					t.Fatalf("expected parse success, got: %v", err)
				}
			})
		}
	})

	t.Run("InvalidUnicodeEscapeCases", func(t *testing.T) {
		t.Parallel()

		invalid := []string{
			`query { f(arg: "bad surrogate \uDEAD") }`,
			`query { f(arg: "bad high surrogate pair \uDEAD\uDEAD") }`,
			`query { f(arg: "bad low surrogate pair \uD800\uD800") }`,
			`query { f(arg: "bad \u{} esc") }`,
			`query { f(arg: "bad \u{FXXX} esc") }`,
			`query { f(arg: "bad \u{FFFF esc") }`,
			`query { f(arg: "bad \u{FFFF") }`,
			`query { f(arg: "too high \u{110000} esc") }`,
			`query { f(arg: "way too high \u{12345678} esc") }`,
			`query { f(arg: "too long \u{000000000} esc") }`,
			`query { f(arg: "bad surrogate \uDEAD esc") }`,
			`query { f(arg: "bad surrogate \u{DEAD} esc") }`,
			`query { f(arg: "cannot use braces for surrogate pair \u{D83D}\u{DE00} esc") }`,
			`query { f(arg: "bad high surrogate pair \uDEAD\uDEAD esc") }`,
			`query { f(arg: "bad low surrogate pair \uD800\uD800 esc") }`,
			`query { f(arg: "bad \uD83D\not an escape") }`,
		}

		for _, q := range invalid {
			t.Run(q, func(t *testing.T) {
				t.Parallel()
				_, err := Parse(q)
				if err == nil {
					t.Fatal("expected syntax error, got nil")
				}
			})
		}
	})

	t.Run("CommentAndSourceCharacterCases", func(t *testing.T) {
		t.Parallel()

		_, err := Parse("# Comment 😀\nquery { f }")
		if err != nil {
			t.Fatalf("expected comment with supplementary code point to parse, got: %v", err)
		}

		_, err = Parse("😀")
		if err == nil {
			t.Fatal("expected syntax error for unexpected top-level supplementary character")
		}

		_, err = Parse("\x08")
		if err == nil {
			t.Fatal("expected syntax error for backspace control character")
		}

		nul := string([]byte{0})
		_, err = Parse(nul)
		if err == nil {
			t.Fatal("expected syntax error for NUL control character")
		}
	})

	t.Run("ErrorTextForUnicodeEscapeReportsInvalidCharEscape", func(t *testing.T) {
		t.Parallel()

		_, err := Parse(`query { f(arg: "too high \u{110000} esc") }`)
		if err == nil {
			t.Fatal("expected syntax error")
		}
		if !errors.Is(err, gqlerrors.ErrSyntax) {
			t.Fatalf("expected syntax sentinel error, got: %v", err)
		}
		if got, want := err.Message, "syntax error: invalid char escape"; got != want {
			t.Fatalf("expected %q, got: %v", want, err)
		}
	})
}
