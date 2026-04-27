package norm_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/internal/common/norm"
)

type readStep struct {
	n    int
	eof  bool
	data string
}

func runReadSize(r io.Reader, readSizes []int) []readStep {
	steps := make([]readStep, 0, len(readSizes))
	for _, size := range readSizes {
		buf := make([]byte, size)
		n, err := r.Read(buf)
		steps = append(steps, readStep{
			n:    n,
			eof:  errors.Is(err, io.EOF),
			data: string(buf[:n]),
		})
	}
	return steps
}

func assertStringsReaderParity(t *testing.T, input string, readSizes []int) {
	t.Helper()

	got := runReadSize(norm.NewReader(input), readSizes)
	want := runReadSize(strings.NewReader(input), readSizes)

	if len(got) != len(want) {
		t.Fatalf("step count mismatch: got=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i].n != want[i].n || got[i].eof != want[i].eof || got[i].data != want[i].data {
			t.Fatalf("step %d mismatch:\n got: n=%d eof=%v data=%q\nwant: n=%d eof=%v data=%q",
				i, got[i].n, got[i].eof, got[i].data,
				want[i].n, want[i].eof, want[i].data,
			)
		}
	}
}

func readAllChunked(r io.Reader, chunk int) (string, error) {
	var b strings.Builder
	buf := make([]byte, chunk)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}
		if errors.Is(err, io.EOF) {
			return b.String(), nil
		}
		if err != nil {
			return "", err
		}
	}
}

func TestReader_ReadParity_NonTransformingInputs(t *testing.T) {
	t.Parallel()

	// readSizes drives per-call Read buffer lengths to stress io.Reader parity (zero-length, tiny, and oversized reads) against strings.NewReader.
	readSizes := []int{
		0, 1, 2, 3, 5, 8, 0, 13, 21, 1, 1, 34, 0, 55, 89, 0, 144, 1, 1, 1, 0, 233,
	}

	inputs := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii", "abc"},
		{"plain_ascii", "hello world"},
		{"graphql_plain", `{ f(arg: "plain") }`},
		{"graphql_escape_sequences", "{ f(arg: \"slashes \\n \\t \\u \\u{\") } }"},
		{"graphql_block_string_unicode", `{ f(arg: """block with \u{1F37A} stays literal""") }`},
		{"emoji_and_combining", "emoji 🍺 and combining e\u0301 and composed é"},
		{"invalid_utf8_with_newline", string([]byte{0xff, 'a', 0xc3, 0x28, '\n', 'z'})},
		{"comment_only", "# just a comment"},
		{"comment_with_crlf", "# comment\r\nquery { id }"},
		{"block_string_with_escaped_delim", `{ f(arg: """line1\n"""other""") }`},
		{"multiple_block_strings", `"""first""" """second"""`},
		{"whitespace_only", "   \t  \n  "},
		{"mixed_newlines", "line1\rline2\nline3\r\nline4"},
		{"long_plain_text", "The quick brown fox jumps over the lazy dog. " + "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " + "This is a longer test string to stress buffering."},
		{"tabs_and_spaces", "query\t{\n  field\t\t{\n    id\n  }\n}"},
		{"quote_chars_outside_strings", `query { id } # "not a string" 'also not'`},
		{"unicode_edge_cases", string([]rune{
			0x0000, 0x0001, 0x0002, 0x0003, 0x0004, 0x0005, 0x0006, 0x0007, 0x0008, 0x0009,
			0x000A, 0x000B, 0x000C, 0x000D, 0x000E, 0x000F, 0x0010, 0x0011, 0x0012, 0x0013,
			// omitting characters for brevity
			0x10FFFD, 0x10FFFE, 0x10FFFF,
		})},
		{"combined", "# comment\nquery { field(arg: \"value with unicode 🍺 and escapes \\n\") }\n" + `"""block string with \u{1F37A} and "quotes""""`},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			assertStringsReaderParity(t, tt.in, readSizes)
		})
	}
}

func TestReader_Read_TransformingInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "brace escape",
			in:   `{ f(arg: "\u{1F37A}") }`,
			want: `{ f(arg: "\U0001F37A") }`,
		},
		{
			name: "valid surrogate pair",
			in:   `{ f(arg: "\uD83C\uDF7A") }`,
			want: `{ f(arg: "\U0001F37A") }`,
		},
		{
			name: "unpaired low surrogate becomes sentinel",
			in:   `{ f(arg: "\uDEAD") }`,
			want: `{ f(arg: "\u{DEAD}") }`,
		},
		{
			name: "unpaired high surrogate becomes sentinel",
			in:   `{ f(arg: "\uD83C\u0041") }`,
			want: `{ f(arg: "\u{D83C}\u0041") }`,
		},
		{
			name: "too long brace escape becomes sentinel",
			in:   `{ f(arg: "\u{000000000}") }`,
			want: `{ f(arg: "\u{}") }`,
		},
		{
			name: "only inside normal string transforms",
			in:   `\u{1F37A} { f(arg: "\u{1F37A}") }`,
			want: `\u{1F37A} { f(arg: "\U0001F37A") }`,
		},
		{
			name: "block string unchanged",
			in:   `{ f(arg: """\u{1F37A}""") }`,
			want: `{ f(arg: """\u{1F37A}""") }`,
		},
		{
			name: "braced minimum code point",
			in:   `{ f(arg: "\u{0}") }`,
			want: `{ f(arg: "\U00000000") }`,
		},
		{
			name: "braced maximum code point",
			in:   `{ f(arg: "\u{10FFFF}") }`,
			want: `{ f(arg: "\U0010FFFF") }`,
		},
		{
			name: "braced leading zeros",
			in:   `{ f(arg: "\u{00000000}") }`,
			want: `{ f(arg: "\U00000000") }`,
		},
		{
			name: "valid surrogate pair minimum",
			in:   `{ f(arg: "\uD800\uDC00") }`,
			want: `{ f(arg: "\U00010000") }`,
		},
		{
			name: "valid surrogate pair maximum",
			in:   `{ f(arg: "\uDBFF\uDFFF") }`,
			want: `{ f(arg: "\U0010FFFF") }`,
		},
		{
			name: "high surrogate at end of string",
			in:   `{ f(arg: "\uD83C") }`,
			want: `{ f(arg: "\u{D83C}") }`,
		},
		{
			name: "high surrogate followed by plain char",
			in:   `{ f(arg: "\uD83Cx") }`,
			want: `{ f(arg: "\u{D83C}x") }`,
		},
		{
			name: "two high surrogates in a row",
			in:   `{ f(arg: "\uD800\uD800") }`,
			want: `{ f(arg: "\u{D800}\u{D800}") }`,
		},
		{
			name: "two low surrogates in a row",
			in:   `{ f(arg: "\uDC00\uDC00") }`,
			want: `{ f(arg: "\u{DC00}\u{DC00}") }`,
		},
		{
			name: "braced surrogate is not allowed",
			in:   `{ f(arg: "\u{DEAD}") }`,
			want: `{ f(arg: "\u{DEAD}") }`,
		},
		{
			name: "invalid braced non-hex unchanged",
			in:   `{ f(arg: "\u{FXXX}") }`,
			want: `{ f(arg: "\u{FXXX}") }`,
		},
		{
			name: "incomplete braced escape unchanged",
			in:   `{ f(arg: "\u{FFFF") }`,
			want: `{ f(arg: "\u{FFFF") }`,
		},
		{
			name: "escape before closing quote",
			in:   `{ f(arg: "x\u{1F37A}") }`,
			want: `{ f(arg: "x\U0001F37A") }`,
		},
		{
			name: "multiple escapes in one string",
			in:   `{ f(arg: "\u{41}\uD83C\uDF7A\u{42}") }`,
			want: `{ f(arg: "\U00000041\U0001F37A\U00000042") }`,
		},
		{
			name: "escaped block delimiter sequence remains in block string",
			in:   `{ f(arg: """x\"""\u{1F37A}y""") }`,
			want: `{ f(arg: """x\"""\u{1F37A}y""") }`,
		},
		{
			name: "comment with quotes does not enter string mode",
			in:   `# " """ \u{1F37A}` + "\n" + `{ f(arg: "\u{41}") }`,
			want: `# " """ \u{1F37A}` + "\n" + `{ f(arg: "\U00000041") }`,
		},
		{
			name: "comment ending with carriage return resumes normal lexing",
			in:   `# "\u{1F37A}` + "\r\n" + `{ f(arg: "\u{42}") }`,
			want: `# "\u{1F37A}` + "\r\n" + `{ f(arg: "\U00000042") }`,
		},
	}

	chunks := []int{1, 2, 3, 7, 16, 64}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, chunk := range chunks {
				got, err := readAllChunked(norm.NewReader(tt.in), chunk)
				if err != nil {
					t.Fatalf("chunk=%d: unexpected err: %v", chunk, err)
				}
				if got != tt.want {
					t.Fatalf("chunk=%d\ngot =%q\nwant=%q", chunk, got, tt.want)
				}
			}
		})
	}
}
