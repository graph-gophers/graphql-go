package graphql_test

import (
	"context"
	"fmt"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
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

const (
	memoryPoolSmallQuery = `query SmallQuery { hero { id name } }`
	memoryPoolWideQuery  = `query WideQuery { hero { id name appearsIn friends { id name appearsIn } } }`
	memoryPoolNestQuery  = `query NestedQuery { hero { id name friends { id name friends { id name } } } }`
)

func memoryPoolBenchmarkSchema(enableMemoryPooling bool) *graphql.Schema {
	if enableMemoryPooling {
		return graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})
	}
	return graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{}, graphql.DisableMemoryPooling())
}

func benchmarkQueryForShape(shape string) string {
	switch shape {
	case "Small":
		return memoryPoolSmallQuery
	case "Wide":
		return memoryPoolWideQuery
	default:
		return memoryPoolNestQuery
	}
}

var memoryPoolBenchSink *graphql.Response // prevent compiler optimizations

func BenchmarkQueryExecution_MemoryPooling(b *testing.B) {
	ctx := context.Background()
	modes := []struct {
		name    string
		enabled bool
	}{
		{name: "WithPool", enabled: true},
		{name: "WithoutPool", enabled: false},
	}
	shapes := []string{"Small", "Wide", "Nested"}

	for _, mode := range modes {
		schema := memoryPoolBenchmarkSchema(mode.enabled)
		for _, shape := range shapes {
			query := benchmarkQueryForShape(shape)
			b.Run(mode.name+"/"+shape, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					memoryPoolBenchSink = schema.Exec(ctx, query, "", nil)
				}
			})
		}
	}
}

func BenchmarkQueryExecution_MemoryPoolingParallel(b *testing.B) {
	ctx := context.Background()
	modes := []struct {
		name    string
		enabled bool
	}{
		{name: "WithPool", enabled: true},
		{name: "WithoutPool", enabled: false},
	}
	shapes := []string{"Small", "Wide", "Nested"}

	for _, mode := range modes {
		schema := memoryPoolBenchmarkSchema(mode.enabled)
		for _, shape := range shapes {
			query := benchmarkQueryForShape(shape)
			b.Run(mode.name+"/"+shape, func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						memoryPoolBenchSink = schema.Exec(ctx, query, "", nil)
					}
				})
			})
		}
	}
}
