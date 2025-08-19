package graphql_test

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

const selectionTestSchema = `
    schema { query: Query }
    type Query { hero: Human }
    type Human { id: ID! name: String }
`

type selectionRoot struct {
	t            *testing.T
	expectNames  []string
	expectSorted []string
	hasChecks    map[string]bool
}

type selectionHuman struct {
	t    *testing.T
	id   string
	name string
}

func (r *selectionRoot) Hero(ctx context.Context) *selectionHuman {
	names := graphql.SelectedFieldNames(ctx)
	sorted := graphql.SortedSelectedFieldNames(ctx)
	if !equalStringSlices(names, r.expectNames) {
		r.t.Errorf("SelectedFieldNames = %v, want %v", names, r.expectNames)
	}
	if !equalStringSlices(sorted, r.expectSorted) {
		r.t.Errorf("SortedSelectedFieldNames = %v, want %v", sorted, r.expectSorted)
	}
	for name, want := range r.hasChecks {
		if got := graphql.HasSelectedField(ctx, name); got != want {
			r.t.Errorf("HasSelectedField(%q) = %v, want %v", name, got, want)
		}
	}
	return &selectionHuman{t: r.t, id: "h1", name: "Luke"}
}

// Object-level assertions happen in Hero via a wrapper test function; leaf behavior tested here.
func (h *selectionHuman) ID() graphql.ID { return graphql.ID(h.id) }

func (h *selectionHuman) Name(ctx context.Context) *string {
	// Leaf field: should always produce empty selections regardless of enable/disable.
	if got := graphql.SelectedFieldNames(ctx); len(got) != 0 {
		h.t.Errorf("leaf field SelectedFieldNames = %v, want empty", got)
	}
	if graphql.HasSelectedField(ctx, "anything") {
		h.t.Errorf("leaf field HasSelectedField unexpectedly true")
	}
	if sorted := graphql.SortedSelectedFieldNames(ctx); len(sorted) != 0 {
		h.t.Errorf("leaf field SortedSelectedFieldNames = %v, want empty", sorted)
	}
	return &h.name
}

func TestFieldSelectionHelpers(t *testing.T) {
	tests := []struct {
		name         string
		schemaOpts   []graphql.SchemaOpt
		query        string
		expectNames  []string // expected order from SelectedFieldNames at object boundary
		expectSorted []string // expected from SortedSelectedFieldNames at object boundary
		hasChecks    map[string]bool
	}{
		{
			name:         "enabled object order preserved and sorted copy",
			query:        `query { hero { name id } }`, // order intentionally name,id
			expectNames:  []string{"name", "id"},
			expectSorted: []string{"id", "name"},
			hasChecks:    map[string]bool{"id": true, "name": true, "missing": false},
		},
		{
			name:         "enabled only one field selected",
			query:        `query { hero { id } }`, // order intentionally name,id
			expectNames:  []string{"id"},
			expectSorted: []string{"id"},
			hasChecks:    map[string]bool{"id": true, "name": false, "missing": false},
		},
		{
			name:         "disabled object returns empty",
			schemaOpts:   []graphql.SchemaOpt{graphql.DisableFieldSelections()},
			query:        `query { hero { name id } }`,
			expectNames:  []string{},
			expectSorted: []string{},
			hasChecks:    map[string]bool{"id": false, "name": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &selectionRoot{t: t, expectNames: tt.expectNames, expectSorted: tt.expectSorted, hasChecks: tt.hasChecks}
			s := graphql.MustParseSchema(selectionTestSchema, root, tt.schemaOpts...)
			resp := s.Exec(context.Background(), tt.query, "", nil)
			if len(resp.Errors) > 0 {
				t.Fatalf("execution errors: %v", resp.Errors)
			}
		})
	}
}

func TestSelectedFieldNames_FragmentsAliasesMeta(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectNames []string
		hasChecks   map[string]bool
	}{
		{
			name:        "alias ignored order preserved",
			query:       `query { hero { idAlias: id name } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "idAlias": false, "name": true},
		},
		{
			name:        "fragment spread flattened",
			query:       `fragment HFields on Human { id name } query { hero { ...HFields } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "name": true},
		},
		{
			name:        "inline fragment dedup",
			query:       `query { hero { id ... on Human { id name } } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "name": true},
		},
		{
			name:        "meta field excluded",
			query:       `query { hero { id __typename name } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "name": true, "__typename": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &selectionRoot{t: t, expectNames: tt.expectNames, expectSorted: tt.expectNames, hasChecks: tt.hasChecks}
			s := graphql.MustParseSchema(selectionTestSchema, root)
			resp := s.Exec(context.Background(), tt.query, "", nil)
			if len(resp.Errors) > 0 {
				t.Fatalf("execution errors: %v", resp.Errors)
			}
		})
	}
}

// equalStringSlices compares content and order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
