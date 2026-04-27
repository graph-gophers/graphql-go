// Package norm provides a reader that normalizes Unicode escape sequences in GraphQL string literals.
//
// Summary of behavior and motivation:
//   - Newly supported executable GraphQL documents include string literals using GraphQL-specific
//     Unicode escapes, especially braced escapes (for example, \u{1F600}, \u{10FFFF}, \u{0})
//     and valid UTF-16 surrogate pairs (for example, \uD83D\uDE00).
//   - Raw UTF-8 characters (for example, 😀) were already supported by text/scanner because it can
//     decode UTF-8 source text.
//   - The gap was escape grammar compatibility, not UTF-8 decoding: GraphQL escape forms are not a
//     strict match for Go string escape handling in text/scanner.
//   - This reader rewrites GraphQL escapes inside normal string literals into scanner-compatible
//     forms while leaving block strings, comments, and non-string source text unchanged.
//
// Example query form enabled by normalization:
//
//	mutation {
//	  createReview(episode: JEDI, review: { stars: 5, commentary: "Loved it \u{1F600}" }) {
//	    commentary
//	  }
//	}
//
// Required as of Sep 2025 GraphQL spec: https://github.com/graphql/graphql-spec/pull/849.
package norm

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// reader normalizes Unicode escape sequences in GraphQL string literals.
// It implements [io.Reader] and rewrites escape sequences like \u{1F37A} and \uD83C\uDF7A to Go's \U format for parser compatibility.
type reader struct {
	src           string
	i             int
	pending       string
	pendingOffset int
	inString      bool
	inBlockString bool
	inComment     bool
}

// NewReader returns an [io.Reader] for src.
// For sources without any \u escapes it returns [strings.Reader] directly.
// Otherwise it returns a normalizing reader that rewrites GraphQL Unicode escapes.
func NewReader(src string) io.Reader {
	if !strings.Contains(src, `\u`) {
		return strings.NewReader(src)
	}

	return &reader{src: src}
}

// Reader reads from the source stream, normalizing Unicode escape sequences within GraphQL string literals.
func (r *reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		if r.pendingOffset >= len(r.pending) && r.i >= len(r.src) {
			return 0, io.EOF
		}
		return 0, nil
	}

	var n int
	for n < len(p) {
		if r.pendingOffset < len(r.pending) {
			k := copy(p[n:], r.pending[r.pendingOffset:])
			r.pendingOffset += k
			n += k
			continue
		}

		r.pending = ""
		r.pendingOffset = 0

		if r.i >= len(r.src) {
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}

		r.produceNext()
	}

	return n, nil
}

// produceNext fetches and processes the next character, escape sequence, or string delimiter from the source.
// It tracks whether we are inside a normal string, block string, or unquoted region, and rewrites Unicode escapes accordingly.
func (r *reader) produceNext() {
	if r.inComment {
		if r.src[r.i] == '\n' || r.src[r.i] == '\r' {
			r.inComment = false
		}
		r.emitNextRune()
		return
	}

	if !r.inString && !r.inBlockString {
		if r.src[r.i] == '#' {
			r.inComment = true
			r.pending = "#"
			r.i++
			return
		}

		if hasPrefixAt(r.src, r.i, `"""`) {
			r.inBlockString = true
			r.pending = `"""`
			r.i += 3
			return
		}
		if r.src[r.i] == '"' {
			r.inString = true
			r.pending = `"`
			r.i++
			return
		}
		r.emitNextRune()
		return
	}

	if r.inBlockString {
		if hasPrefixAt(r.src, r.i, `\"""`) {
			r.pending = `\"""`
			r.i += 4
			return
		}
		if hasPrefixAt(r.src, r.i, `"""`) {
			r.inBlockString = false
			r.pending = `"""`
			r.i += 3
			return
		}
		r.emitNextRune()
		return
	}

	// Normal string mode.
	if r.src[r.i] == '\\' {
		if rewritten, consumed, ok := rewriteUnicodeEscape(r.src[r.i:]); ok {
			r.pending = rewritten
			r.i += consumed
			return
		}

		if r.i+1 < len(r.src) {
			r.pending = r.src[r.i : r.i+2]
			r.i += 2
			return
		}

		r.pending = r.src[r.i : r.i+1]
		r.i++
		return
	}

	if r.src[r.i] == '"' {
		r.inString = false
		r.pending = `"`
		r.i++
		return
	}

	r.emitNextRune()
}

// emitNextRune outputs the next UTF-8 encoded rune from the source without transformation.
func (r *reader) emitNextRune() {
	_, size := utf8.DecodeRuneInString(r.src[r.i:])
	r.pending = r.src[r.i : r.i+size]
	r.i += size
}

// rewriteUnicodeEscape converts GraphQL Unicode escape syntax to Go's \U format.
// It handles \u{...} braced form, \uXXXX surrogate pairs, and invalid sequences.
// It returns (rewritten string, bytes consumed, ok). Invalid forms return empty string and false.
func rewriteUnicodeEscape(input string) (string, int, bool) {
	if len(input) < 2 || input[0] != '\\' || input[1] != 'u' {
		return "", 0, false
	}

	// \u{1F37A}
	if len(input) >= 4 && input[2] == '{' {
		j := 3
		for j < len(input) && isHexDigit(input[j]) {
			j++
		}
		if j-3 > 8 {
			if j < len(input) && input[j] == '}' {
				return `\u{}`, j + 1, true
			}
			return `\u{}`, j, true
		}
		if j == 3 || j >= len(input) || input[j] != '}' {
			return "", 0, false
		}

		cp, err := strconv.ParseUint(input[3:j], 16, 32)
		if err != nil || cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF) {
			return "", 0, false
		}

		return fmt.Sprintf("\\U%08X", cp), j + 1, true
	}

	if len(input) < 6 {
		return "", 0, false
	}

	// \uD83C\uDF7A
	hex := input[2:6]
	if !allHex(hex) {
		return "", 0, false
	}
	code, err := strconv.ParseUint(hex, 16, 16)
	if err != nil {
		return "", 0, false
	}

	if code >= 0xDC00 && code <= 0xDFFF {
		// Force a syntax error later for an unpaired low surrogate.
		return fmt.Sprintf("\\u{%04X}", code), 6, true
	}

	if code < 0xD800 || code > 0xDBFF {
		return "", 0, false
	}

	if len(input) < 12 || input[6] != '\\' || input[7] != 'u' || !allHex(input[8:12]) {
		// Force a syntax error later for an unpaired high surrogate.
		return fmt.Sprintf("\\u{%04X}", code), 6, true
	}

	low, err := strconv.ParseUint(input[8:12], 16, 16)
	if err != nil || low < 0xDC00 || low > 0xDFFF {
		// Force a syntax error later for an unpaired high surrogate.
		return fmt.Sprintf("\\u{%04X}", code), 6, true
	}

	cp := ((code-0xD800)<<10 | (low - 0xDC00)) + 0x10000
	return fmt.Sprintf("\\U%08X", cp), 12, true
}

func hasPrefixAt(s string, i int, prefix string) bool {
	return i+len(prefix) <= len(s) && s[i:i+len(prefix)] == prefix
}

func isHexDigit(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}

func allHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isHexDigit(s[i]) {
			return false
		}
	}
	return true
}
