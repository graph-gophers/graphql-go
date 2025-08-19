package graphql

import (
	"context"
	"sort"

	"github.com/graph-gophers/graphql-go/internal/selections"
)

// SelectedFieldNames returns the set of immediate child field names selected
// on the value returned by the current resolver. It returns an empty slice
// when the current field's return type is a leaf (scalar / enum) or when the
// feature was disabled at schema construction via DisableFieldSelections.
// The returned slice is a copy and is safe for the caller to modify.
//
// It is intentionally simple and does not expose the internal AST. If more
// detailed information is needed in the future (e.g. arguments per child,
// nested trees) a separate API can be added without breaking this one.
//
// Notes:
//   - Fragment spreads & inline fragments are flattened; the union of all
//     possible child fields is returned (deduplicated, preserving first
//     appearance order in the query document).
//   - Field aliases are ignored; the original schema field names are returned.
//   - Meta fields beginning with "__" (including __typename) are excluded.
func SelectedFieldNames(ctx context.Context) []string {
	// If no selection info is present (leaf field or no child selections), return empty slice.
	lazy := selections.FromContext(ctx)
	if lazy == nil {
		return []string{}
	}
	return lazy.Names()
}

// HasSelectedField returns true if the immediate child selection list contains
// the provided field name (case sensitive). It returns false for leaf return
// types and when DisableFieldSelections was used.
func HasSelectedField(ctx context.Context, name string) bool {
	lazy := selections.FromContext(ctx)
	if lazy == nil {
		return false
	}
	return lazy.Has(name)
}

// SortedSelectedFieldNames returns the same data as SelectedFieldNames but
// sorted lexicographically for deterministic ordering scenarios (e.g. cache
// key generation). It will also return an empty slice when selections are
// disabled.
func SortedSelectedFieldNames(ctx context.Context) []string {
	names := SelectedFieldNames(ctx)
	if len(names) <= 1 {
		return names
	}
	out := make([]string, len(names))
	copy(out, names)
	sort.Strings(out)
	return out
}
