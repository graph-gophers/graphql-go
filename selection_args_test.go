package graphql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

// Date is a custom scalar implementing decode.Unmarshaler.
type Date struct{ Value string }

func (d *Date) ImplementsGraphQLType(name string) bool { return name == "Date" }
func (d *Date) UnmarshalGraphQL(input any) error {
	s, ok := input.(string)
	if !ok {
		return fmt.Errorf("Date expects string got %T", input)
	}
	d.Value = s
	return nil
}

// harness captures decoded argument structs from inside resolvers.
type harness struct {
	got any
}

type parentResolver struct{}

func (p *parentResolver) ScalarField(ctx context.Context, args struct{ X int32 }) int32 {
	return args.X
}

func (p *parentResolver) StringField(ctx context.Context, args struct{ S string }) string {
	return args.S
}

func (p *parentResolver) EnumField(ctx context.Context, args struct{ Color string }) string {
	return args.Color
}

func (p *parentResolver) CustomField(ctx context.Context, args struct{ D Date }) string {
	return args.D.Value
}

func (p *parentResolver) ComplexField(ctx context.Context, args complexArgs) string {
	return "ok"
}

// decoded argument holder structs
type scalarArgs struct{ X int32 }

type stringArgs struct{ S string }

type enumArgs struct{ Color string }

type customArgs struct{ D Date }

type complexArgs struct {
	R struct {
		Start int32
		End   int32
	}
	Colors []string
}

type queryResolver struct {
	h    *harness
	path string
}

func (q *queryResolver) Parent(ctx context.Context) *parentResolver {
	// decode any child arguments and assign to the harness
	dec := func(path string, dst any) {
		if ok, _ := graphql.DecodeSelectedFieldArgs(ctx, path, dst); ok && q.h.got == nil {
			q.h.got = dst
		}
	}
	switch q.path {
	case "scalarField":
		dec(q.path, &scalarArgs{})
	case "stringField":
		dec(q.path, &stringArgs{})
	case "enumField":
		dec(q.path, &enumArgs{})
	case "customField":
		dec(q.path, &customArgs{})
	case "complexField":
		dec(q.path, &complexArgs{})
	}
	return &parentResolver{}
}

func TestDecodeSelectedFieldArgs(t *testing.T) {
	schemaSDL := `
		scalar Date
		enum Color { RED GREEN BLUE }
		input Range { start: Int! end: Int! }
		type Query { parent: Parent! }
		type Parent {
			scalarField(x: Int!): Int!
			stringField(s: String!): String!
			enumField(color: Color!): String!
			customField(d: Date!): String!
			complexField(r: Range!, colors: [Color!]!): String!
		}
	`

	tests := []struct {
		name   string
		query  string
		path   string
		expect func(t *testing.T, v any)
	}{
		{
			name:  "scalar int",
			query: `query { parent { scalarField(x: 42) } }`,
			path:  "scalarField",
			expect: func(t *testing.T, v any) {
				got := v.(*scalarArgs)
				if got.X != 42 {
					t.Errorf("want 42 got %d", got.X)
				}
			},
		},
		{
			name:  "string",
			query: `query { parent { stringField(s: "abc") } }`,
			path:  "stringField",
			expect: func(t *testing.T, v any) {
				got := v.(*stringArgs)
				if got.S != "abc" {
					t.Errorf("want abc got %s", got.S)
				}
			},
		},
		{
			name:  "custom scalar",
			query: `query { parent { customField(d: "2025-01-02") } }`,
			path:  "customField",
			expect: func(t *testing.T, v any) {
				got := v.(*customArgs)
				if got.D.Value != "2025-01-02" {
					t.Errorf("want date got %s", got.D.Value)
				}
			},
		},
		{
			name:  "complex",
			query: `query { parent { complexField(r: { start: 1, end: 5 }, colors: [GREEN, BLUE]) } }`,
			path:  "complexField",
			expect: func(t *testing.T, v any) {
				got := v.(*complexArgs)
				if got.R.Start != 1 || got.R.End != 5 {
					t.Errorf("range mismatch: %+v", got.R)
				}
				if len(got.Colors) != 2 || got.Colors[0] != "GREEN" || got.Colors[1] != "BLUE" {
					t.Errorf("colors mismatch: %#v", got.Colors)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &harness{}
			q := &queryResolver{h: h, path: tt.path}
			schema := graphql.MustParseSchema(schemaSDL, q)
			res := schema.Exec(context.Background(), tt.query, "", nil)
			if len(res.Errors) > 0 {
				t.Fatalf("unexpected errors: %+v", res.Errors)
			}
			if h.got == nil {
				t.Errorf("resolver did not capture decoded args (path %s)", tt.path)
				return
			}
			tt.expect(t, h.got)
		})
	}
}
