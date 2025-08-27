package graphql_test

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

const selectionTestSchema = `
	schema { query: Query }
	type Query { customer: Customer }
	type Customer { id: ID! name: String items: [Item!]! }
	type Item { id: ID! name: String category: Category }
	type Category { id: ID! }
`

type selectionRoot struct {
	t            *testing.T
	expectNames  []string
	expectSorted []string
	hasChecks    map[string]bool
}
type selectionCustomer struct {
	t        *testing.T
	id, name string
}

func (r *selectionRoot) Customer(ctx context.Context) *selectionCustomer {
	if r.expectNames != nil {
		names := graphql.SelectedFieldNames(ctx)
		if !equalStringSlices(names, r.expectNames) {
			r.t.Errorf("SelectedFieldNames = %v, want %v", names, r.expectNames)
		}
	}
	if r.expectSorted != nil {
		sorted := graphql.SortedSelectedFieldNames(ctx)
		if !equalStringSlices(sorted, r.expectSorted) {
			r.t.Errorf("SortedSelectedFieldNames = %v, want %v", sorted, r.expectSorted)
		}
	}
	for n, want := range r.hasChecks {
		if got := graphql.HasSelectedField(ctx, n); got != want {
			r.t.Errorf("HasSelectedField(%q) = %v, want %v", n, got, want)
		}
	}
	return &selectionCustomer{t: r.t, id: "c1", name: "Alice"}
}

func (h *selectionCustomer) ID() graphql.ID { return graphql.ID(h.id) }
func (h *selectionCustomer) Name(ctx context.Context) *string {
	if len(graphql.SelectedFieldNames(ctx)) != 0 {
		h.t.Errorf("leaf selections should be empty")
	}
	if graphql.HasSelectedField(ctx, "anything") {
		h.t.Errorf("unexpected leaf HasSelectedField true")
	}
	if len(graphql.SortedSelectedFieldNames(ctx)) != 0 {
		h.t.Errorf("leaf sorted selections should be empty")
	}
	return &h.name
}

// nested types for extended schema
type selectionItem struct {
	id, name string
	category *selectionCategory
}
type selectionCategory struct{ id string }

func (h *selectionCustomer) Items() []*selectionItem {
	return []*selectionItem{{id: "i1", name: "Item", category: &selectionCategory{id: "cat1"}}}
}
func (p *selectionItem) ID() graphql.ID               { return graphql.ID(p.id) }
func (p *selectionItem) Name() *string                { return &p.name }
func (p *selectionItem) Category() *selectionCategory { return p.category }
func (c *selectionCategory) ID() graphql.ID           { return graphql.ID(c.id) }

func TestFieldSelectionHelpers(t *testing.T) {
	tests := []struct {
		name         string
		schemaOpts   []graphql.SchemaOpt
		query        string
		expectNames  []string
		expectSorted []string
		hasChecks    map[string]bool
	}{
		{
			name:         "enabled order",
			query:        `query { customer { name id } }`,
			expectNames:  []string{"name", "id"},
			expectSorted: []string{"id", "name"},
			hasChecks:    map[string]bool{"id": true, "name": true},
		},
		{
			name:         "one field",
			query:        `query { customer { id } }`,
			expectNames:  []string{"id"},
			expectSorted: []string{"id"},
			hasChecks:    map[string]bool{"id": true, "name": false},
		},
		{
			name:         "nested paths",
			query:        `query { customer { items { id name category { id } } id } }`,
			expectNames:  []string{"items", "items.id", "items.name", "items.category", "items.category.id", "id"},
			expectSorted: []string{"id", "items", "items.category", "items.category.id", "items.id", "items.name"},
			hasChecks:    map[string]bool{"items": true, "items.id": true, "items.name": true, "items.category": true, "items.category.id": true, "id": true},
		},
		{
			name:         "disabled",
			schemaOpts:   []graphql.SchemaOpt{graphql.DisableFieldSelections()},
			query:        `query { customer { name id } }`,
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
		name, query string
		expectNames []string
		hasChecks   map[string]bool
	}{
		{
			name:        "alias ignored",
			query:       `query { customer { idAlias: id name } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "idAlias": false, "name": true},
		},
		{
			name:        "fragment spread",
			query:       `fragment CFields on Customer { id name } query { customer { ...CFields } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "name": true},
		},
		{
			name:        "inline fragment",
			query:       `query { customer { id ... on Customer { id name } } }`,
			expectNames: []string{"id", "name"},
			hasChecks:   map[string]bool{"id": true, "name": true},
		},
		{
			name:        "meta excluded",
			query:       `query { customer { id __typename name } }`,
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
