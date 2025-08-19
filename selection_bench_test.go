package graphql_test

import (
	"context"
	"fmt"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

// This benchmark compares query execution when resolvers do NOT call the
// selection helpers vs when they call SelectedFieldNames at object boundaries.
// It documents the lazy overhead of computing child field selections.

const lazyBenchSchema = `
			schema { query: Query }
			type Query { hero: Human }
			type Human { id: ID! name: String friends: [Human!]! }
		`

// Simple in-memory data graph.
type human struct {
	id, name string
	friends  []*human
}

// Build a small graph once outside the benchmark loops.
var benchHero *human

func init() {
	// Create 5 friends (no recursive friends to keep size stable).
	friends := make([]*human, 5)
	for i := range friends {
		friends[i] = &human{id: fmt.Sprintf("F%d", i), name: "Friend"}
	}
	benchHero = &human{id: "H1", name: "Hero", friends: friends}
}

// Baseline resolvers (do NOT invoke selection helpers).
type (
	rootBaseline          struct{}
	humanResolverBaseline struct{ h *human }
)

func (r *rootBaseline) Hero(ctx context.Context) *humanResolverBaseline {
	return &humanResolverBaseline{h: benchHero}
}
func (h *humanResolverBaseline) ID() graphql.ID { return graphql.ID(h.h.id) }
func (h *humanResolverBaseline) Name() *string  { return &h.h.name }
func (h *humanResolverBaseline) Friends(ctx context.Context) []*humanResolverBaseline {
	out := make([]*humanResolverBaseline, len(h.h.friends))
	for i, f := range h.h.friends {
		out[i] = &humanResolverBaseline{h: f}
	}
	return out
}

// Instrumented resolvers (CALL selection helpers once per object-level resolver).
type (
	rootWithSel          struct{}
	humanResolverWithSel struct{ h *human }
)

func (r *rootWithSel) Hero(ctx context.Context) *humanResolverWithSel {
	// Selection list for hero object (id, name, friends)
	_ = graphql.SelectedFieldNames(ctx)
	return &humanResolverWithSel{h: benchHero}
}

func (h *humanResolverWithSel) ID(ctx context.Context) graphql.ID { // leaf: expecting empty slice
	return graphql.ID(h.h.id)
}

func (h *humanResolverWithSel) Name(ctx context.Context) *string { // leaf
	return &h.h.name
}

func (h *humanResolverWithSel) Friends(ctx context.Context) []*humanResolverWithSel {
	// Selection list on list field: children of Human inside list items.
	_ = graphql.SelectedFieldNames(ctx)
	out := make([]*humanResolverWithSel, len(h.h.friends))
	for i, f := range h.h.friends {
		// For each friend object we also call once at the object resolver boundary.
		out[i] = &humanResolverWithSel{h: f}
	}
	return out
}

// Query used for both benchmarks.
const lazyBenchQuery = `query { hero { id name friends { id name } } }`

func BenchmarkFieldSelections_NoUsage(b *testing.B) {
	schema := graphql.MustParseSchema(lazyBenchSchema, &rootBaseline{})
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Exec(ctx, lazyBenchQuery, "", nil)
	}
}

func BenchmarkFieldSelections_Disabled_NoUsage(b *testing.B) {
	schema := graphql.MustParseSchema(lazyBenchSchema, &rootBaseline{}, graphql.DisableFieldSelections())
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Exec(ctx, lazyBenchQuery, "", nil)
	}
}

func BenchmarkFieldSelections_WithSelectedFieldNames(b *testing.B) {
	schema := graphql.MustParseSchema(lazyBenchSchema, &rootWithSel{})
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Exec(ctx, lazyBenchQuery, "", nil)
	}
}
