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
	value, ok := input.(int64)
	if !ok {
		return fmt.Errorf("Int64 expects int64 got %T", input)
	}
	u.Value = value
	return nil
}

type issue305Resolver struct{}

func (r *issue305Resolver) Custom(args struct{ Hash int64Scalar }) string {
	return fmt.Sprintf("%d", args.Hash.Value)
}

func (r *issue305Resolver) Regular(args struct{ X int32 }) int32 {
	return args.X
}

func TestIssue305IntegerLiteralBehavior(t *testing.T) {
	schema := graphql.MustParseSchema(`
		scalar Int64
		type Query {
			custom(hash: Int64!): String!
			regular(x: Int!): Int!
		}
	`, &issue305Resolver{})

	const large = "3626262620"

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema:         schema,
			Query:          fmt.Sprintf(`{ custom(hash: %s) }`, large),
			ExpectedResult: fmt.Sprintf(`{"custom":%q}`, large),
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
