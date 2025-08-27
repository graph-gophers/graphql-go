package graphql

import (
	"context"
	"sort"

	"github.com/graph-gophers/graphql-go/internal/selections"
)

// SelectedFieldNames returns the set of selected field paths underneath the
// the current resolver. Paths are dot-delimited for nested structures (e.g.
// "products", "products.id", "products.category.id"). Immediate child field
// names are always present (even when they have further children). Order preserves
// the first appearance in the query after fragment flattening, performing a
// depth-first traversal.
// It returns an empty slice when the current field's return type is a leaf
// (scalar / enum) or when DisableFieldSelections was used at schema creation.
// The returned slice is a copy safe for caller modification.
//
// Notes:
//   - Fragment spreads & inline fragments are flattened.
//   - Field aliases are ignored; original schema field names are used.
//   - Meta fields beginning with "__" (including __typename) are excluded.
//   - Duplicate paths are removed, preserving the earliest occurrence.
func SelectedFieldNames(ctx context.Context) []string {
	// If no selection info is present (leaf field or no child selections), return empty slice.
	lazy := selections.FromContext(ctx)
	if lazy == nil {
		return []string{}
	}
	return lazy.Names()
}

// HasSelectedField returns true if the child selection list contains the provided
// (possibly nested) path (case sensitive). It returns false for leaf resolvers
// and when DisableFieldSelections was used.
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
