package graphql_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

const overlapBenchSchema = `schema { query: Query } type Query { root: Thing } type Thing { id: ID! name: String value: String }`

func buildLargeQuery(count int) string {
	var b strings.Builder
	b.Grow(20 + count*8)
	b.WriteString("query{root{")
	for i := 0; i < count; i++ {
		b.WriteString("f")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(":id ")
	}
	b.WriteString("}}")
	return b.String()
}

func buildFragmentedQuery(total int) string {
	if total < 4 {
		return buildLargeQuery(total)
	}
	top := total / 2
	rest := total - top
	fragA := rest / 2
	fragB := rest - fragA
	var topSel strings.Builder
	topSel.Grow(32 + top*8)
	topSel.WriteString("query{root{")
	for i := 0; i < top; i++ {
		topSel.WriteString("t")
		topSel.WriteString(strconv.Itoa(i))
		topSel.WriteString(":id ")
	}
	topSel.WriteString(" ...FragA ...FragB }}")
	var frags strings.Builder
	frags.Grow(32 + (fragA+fragB)*8)
	frags.WriteString(" fragment FragA on Thing {")
	for i := 0; i < fragA; i++ {
		frags.WriteString(" a")
		frags.WriteString(strconv.Itoa(i))
		frags.WriteString(":id")
	}
	frags.WriteString(" }")
	frags.WriteString(" fragment FragB on Thing {")
	for i := 0; i < fragB; i++ {
		frags.WriteString(" b")
		frags.WriteString(strconv.Itoa(i))
		frags.WriteString(":id")
	}
	frags.WriteString(" }")
	return topSel.String() + frags.String()
}

type overlapRoot struct{}

func (r *overlapRoot) Root() *thingResolver { return &thingResolver{} }

type thingResolver struct{}

func (t *thingResolver) ID() graphql.ID { return graphql.ID("1") }
func (t *thingResolver) Name() *string  { s := "n"; return &s }
func (t *thingResolver) Value() *string { s := "v"; return &s }

func BenchmarkValidateOverlap(b *testing.B) {
	sizes := []int{500, 1000, 2000, 5000}
	for _, n := range sizes {
		b.Run("fields_"+strconv.Itoa(n), func(b *testing.B) {
			schema := graphql.MustParseSchema(overlapBenchSchema, &overlapRoot{})
			query := buildLargeQuery(n)
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				_ = schema.Exec(ctx, query, "", nil)
			}
		})
	}
	b.Run("fragments_1000", func(b *testing.B) {
		schema := graphql.MustParseSchema(overlapBenchSchema, &overlapRoot{})
		query := buildFragmentedQuery(1000)
		ctx := context.Background()
		b.ReportAllocs()
		for b.Loop() {
			_ = schema.Exec(ctx, query, "", nil)
		}
	})
}
