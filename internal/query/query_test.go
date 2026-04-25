package query

import "testing"

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
