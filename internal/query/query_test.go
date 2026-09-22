package query

import (
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/internal/common"
)

func TestParseRejectsEmptyOperationSelectionSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{name: "query", query: "query {}"},
		{name: "mutation", query: "mutation {}"},
		{name: "subscription", query: "subscription {}"},
		{name: "anonymous query", query: "{}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tc.query)
			if err == nil {
				t.Fatalf("Parse(%q): expected syntax error, got nil", tc.query)
			}
		})
	}
}

func TestParseRejectsEmptyNestedSelectionSet(t *testing.T) {
	t.Parallel()

	_, err := Parse("query { user {} }")
	if err == nil {
		t.Fatalf("expected syntax error for empty nested selection set, got nil")
	}
}

func TestParseAcceptsFullUnicodeEscapeSyntax(t *testing.T) {
	t.Parallel()

	_, err := Parse(`query { user(id: "\u{1F37A}") { id } }`)
	if err != nil {
		t.Fatalf("expected query to parse, got error: %v", err)
	}
}

func TestParseAcceptsSupplementaryPlaneCodePoint(t *testing.T) {
	t.Parallel()

	_, err := Parse(`query { user(id: "🍺") { id } }`)
	if err != nil {
		t.Fatalf("expected query to parse, got error: %v", err)
	}
}

func TestParseAcceptsValidSurrogatePairEscape(t *testing.T) {
	t.Parallel()

	_, err := Parse(`query { user(id: "\uD83C\uDF7A") { id } }`)
	if err != nil {
		t.Fatalf("expected query to parse, got error: %v", err)
	}
}

func TestParseRejectsInvalidSurrogatePairEscape(t *testing.T) {
	t.Parallel()

	_, err := Parse(`query { user(id: "\uD83C\u0041") { id } }`)
	if err == nil {
		t.Fatal("expected syntax error for invalid surrogate pair, got nil")
	}
}

func TestParseNestingDepth(t *testing.T) {
	tests := []struct {
		name  string
		query string
		valid bool
	}{
		{
			name:  "selection set at limit",
			query: nestedSelectionSet(common.MaxParserDepth),
			valid: true,
		},
		{
			name:  "selection set above limit",
			query: nestedSelectionSet(common.MaxParserDepth + 1),
		},
		{
			name:  "inline fragment at limit",
			query: nestedInlineFragment(common.MaxParserDepth),
			valid: true,
		},
		{
			name:  "inline fragment above limit",
			query: nestedInlineFragment(common.MaxParserDepth + 1),
		},
		{
			name:  "list literal at limit",
			query: nestedListLiteral(common.MaxParserDepth - 1),
			valid: true,
		},
		{
			name:  "list literal above limit",
			query: nestedListLiteral(common.MaxParserDepth),
		},
		{
			name:  "object literal at limit",
			query: nestedObjectLiteral(common.MaxParserDepth - 1),
			valid: true,
		},
		{
			name:  "object literal above limit",
			query: nestedObjectLiteral(common.MaxParserDepth),
		},
		{
			name:  "list type at limit",
			query: nestedListType(common.MaxParserDepth),
			valid: true,
		},
		{
			name:  "list type above limit",
			query: nestedListType(common.MaxParserDepth + 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.query)
			if tt.valid {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected nesting depth error")
			}
			if !strings.Contains(err.Message, "maximum nesting depth exceeded") {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func nestedSelectionSet(depth int) string {
	return strings.Repeat("{field", depth) + strings.Repeat("}", depth)
}

func nestedInlineFragment(depth int) string {
	return "{" + strings.Repeat("... on Query {", depth-1) + "field" + strings.Repeat("}", depth)
}

func nestedListLiteral(depth int) string {
	return "{field(value:" + strings.Repeat("[", depth) + "1" + strings.Repeat("]", depth) + ")}"
}

func nestedObjectLiteral(depth int) string {
	return "{field(value:" + strings.Repeat("{value:", depth) + "1" + strings.Repeat("}", depth) + ")}"
}

func nestedListType(depth int) string {
	return "query($value:" + strings.Repeat("[", depth) + "Int" + strings.Repeat("]", depth) + "){field}"
}
