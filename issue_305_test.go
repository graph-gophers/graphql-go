package graphql_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/graph-gophers/graphql-go"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type int64Scalar struct {
	Value int64
}

func (u *int64Scalar) ImplementsGraphQLType(name string) bool { return name == "Int64" }

func (u *int64Scalar) UnmarshalGraphQL(input any) error {
	switch value := input.(type) {
	case int64:
		u.Value = value
		return nil
	case int32:
		u.Value = int64(value)
		return nil
	default:
		return fmt.Errorf("Int64 expects int32 or int64 got %T", input)
	}
}

type issue305Resolver struct{}

func (r *issue305Resolver) Custom(args struct{ Hash int64Scalar }) string {
	return fmt.Sprintf("%d", args.Hash.Value)
}

func (r *issue305Resolver) Regular(args struct{ X int32 }) int32 {
	return args.X
}

func (r *issue305Resolver) Floaty(args struct{ X float64 }) float64 {
	return args.X
}

func TestIssue305IntegerLiteralBehavior(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar Int64
		type Query {
			custom(hash: Int64!): String!
			regular(x: Int!): Int!
			floaty(x: Float!): Float!
		}
	`, &issue305Resolver{})

	const large = "3626262620"
	const largeInt int64 = 3626262620
	const negLargeInt int64 = -largeInt

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema:         schema,
			Query:          `{ custom(hash: 123) }`,
			ExpectedResult: `{"custom":"123"}`,
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ custom(hash: -%s) }`, large),
			ExpectedResult: fmt.Sprintf(`{"custom":%q}`, fmt.Sprintf("-%s", large)),
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ custom(hash: %s) }`, large),
			ExpectedResult: fmt.Sprintf(`{"custom":%q}`, large),
		},
		{
			Schema:        schema,
			Query:         `query($hash: Int64!) { custom(hash: $hash) }`,
			Variables:     map[string]any{"hash": int32(123)},
			ExpectedResult: `{"custom":"123"}`,
		},
		{
			Schema:         schema,
			Query:          `query($hash: Int64!) { custom(hash: $hash) }`,
			Variables:      map[string]any{"hash": largeInt},
			ExpectedResult: fmt.Sprintf(`{"custom":%q}`, large),
		},
		{
			Schema:         schema,
			Query:          `query($hash: Int64!) { custom(hash: $hash) }`,
			Variables:      map[string]any{"hash": negLargeInt},
			ExpectedResult: fmt.Sprintf(`{"custom":%q}`, fmt.Sprintf("-%s", large)),
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ floaty(x: %s) }`, large),
			ExpectedResult: fmt.Sprintf(`{"floaty":%s}`, large),
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ floaty(x: -%s) }`, large),
			ExpectedResult: fmt.Sprintf(`{"floaty":%s}`, fmt.Sprintf("-%s", large)),
		},
		{
			Schema:         schema,
			Query:          `query($x: Float!) { floaty(x: $x) }`,
			Variables:      map[string]any{"x": largeInt},
			ExpectedResult: fmt.Sprintf(`{"floaty":%s}`, large),
		},
		{
			Schema:         schema,
			Query:          `query($x: Float!) { floaty(x: $x) }`,
			Variables:      map[string]any{"x": negLargeInt},
			ExpectedResult: fmt.Sprintf(`{"floaty":%s}`, fmt.Sprintf("-%s", large)),
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ regular(x: %d) }`, math.MaxInt32),
			ExpectedResult: fmt.Sprintf(`{"regular":%d}`, math.MaxInt32),
		},
		{
			Schema: schema,
			Query:  fmt.Sprintf(`query { regular(x: %d) }`, int64(math.MaxInt32)+1),
			ExpectedErrors: []*gqlerrors.QueryError{{
				Message:   fmt.Sprintf("Int cannot represent non 32-bit signed integer value: %d", int64(math.MaxInt32)+1),
				Locations: []gqlerrors.Location{{Line: 1, Column: 20}},
				Rule:      "ValuesOfCorrectTypeRule",
			}},
		},
		{
			Schema: schema,
			Query:  fmt.Sprintf(`query { regular(x: %d) }`, int64(math.MinInt32)-1),
			ExpectedErrors: []*gqlerrors.QueryError{{
				Message:   fmt.Sprintf("Int cannot represent non 32-bit signed integer value: %d", int64(math.MinInt32)-1),
				Locations: []gqlerrors.Location{{Line: 1, Column: 20}},
				Rule:      "ValuesOfCorrectTypeRule",
			}},
		},
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ regular(x: %d) }`, math.MinInt32),
			ExpectedResult: fmt.Sprintf(`{"regular":%d}`, math.MinInt32),
		},
		{
			Schema: schema,
			Query:  `query { regular(x: 3626262620) }`,
			ExpectedErrors: []*gqlerrors.QueryError{{
				Message:   "Int cannot represent non 32-bit signed integer value: 3626262620",
				Locations: []gqlerrors.Location{{Line: 1, Column: 20}},
				Rule:      "ValuesOfCorrectTypeRule",
			}},
		},
	})
}
