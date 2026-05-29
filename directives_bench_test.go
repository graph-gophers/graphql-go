package graphql_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/example/starwars"
)

const starwarsCostBenchmarkSchema = `
	directive @cost(weight: Int!, multiplier: String) on FIELD_DEFINITION

	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		hero(episode: Episode = NEWHOPE): Character @cost(weight: 1)
		reviews(episode: Episode!): [Review]! @cost(weight: 2)
		search(text: String!): [SearchResult]! @cost(weight: 3)
		character(id: ID!): Character @cost(weight: 1)
		droid(id: ID!): Droid @cost(weight: 1)
		human(id: ID!): Human @cost(weight: 1)
		starship(id: ID!): Starship @cost(weight: 1)
	}

	type Mutation {
		createReview(episode: Episode!, review: ReviewInput!): Review @cost(weight: 2)
	}

	enum Episode {
		NEWHOPE
		EMPIRE
		JEDI
	}

	interface Character {
		id: ID! @cost(weight: 1)
		name: String! @cost(weight: 1)
		friends: [Character] @cost(weight: 2)
		friendsConnection(first: Int, after: ID): FriendsConnection! @cost(weight: 2, multiplier: "first")
		appearsIn: [Episode!]! @cost(weight: 1)
	}

	enum LengthUnit {
		METER
		FOOT
	}

	type Human implements Character {
		id: ID! @cost(weight: 1)
		name: String! @cost(weight: 1)
		height(unit: LengthUnit = METER): Float! @cost(weight: 1)
		mass: Float @cost(weight: 1)
		friends: [Character] @cost(weight: 2)
		friendsConnection(first: Int, after: ID): FriendsConnection! @cost(weight: 2, multiplier: "first")
		appearsIn: [Episode!]! @cost(weight: 1)
		starships: [Starship] @cost(weight: 2)
	}

	type Droid implements Character {
		id: ID! @cost(weight: 1)
		name: String! @cost(weight: 1)
		friends: [Character] @cost(weight: 2)
		friendsConnection(first: Int, after: ID): FriendsConnection! @cost(weight: 2, multiplier: "first")
		appearsIn: [Episode!]! @cost(weight: 1)
		primaryFunction: String @cost(weight: 1)
	}

	type FriendsConnection {
		totalCount: Int! @cost(weight: 1)
		edges: [FriendsEdge] @cost(weight: 2)
		friends: [Character] @cost(weight: 2)
		pageInfo: PageInfo! @cost(weight: 1)
	}

	type FriendsEdge {
		cursor: ID! @cost(weight: 1)
		node: Character @cost(weight: 1)
	}

	type PageInfo {
		startCursor: ID @cost(weight: 1)
		endCursor: ID @cost(weight: 1)
		hasNextPage: Boolean! @cost(weight: 1)
	}

	type Review {
		stars: Int! @cost(weight: 1)
		commentary: String @cost(weight: 1)
	}

	input ReviewInput {
		stars: Int!
		commentary: String
	}

	type Starship {
		id: ID! @cost(weight: 1)
		name: String! @cost(weight: 1)
		length(unit: LengthUnit = METER): Float! @cost(weight: 1)
	}

	union SearchResult = Human | Droid | Starship
`

type benchmarkCostDirective struct{}

func (d *benchmarkCostDirective) Name() string {
	return "cost"
}

type benchmarkCostArgs struct {
	Weight     int32
	Multiplier graphql.NullString
}

type benchmarkCostKey struct{}

func (d *benchmarkCostDirective) Visit(ctx context.Context, dc graphql.DirectiveContext) error {
	var args benchmarkCostArgs
	if err := dc.DecodeArgs(&args); err != nil {
		return err
	}
	if args.Weight < 0 {
		return fmt.Errorf("invalid weight %d", args.Weight)
	}

	multiplier := int32(1)
	if args.Multiplier.Value != nil {
		if err := dc.FieldArg(*args.Multiplier.Value, &multiplier); err != nil {
			return err
		}
		if multiplier < 0 {
			return fmt.Errorf("invalid multiplier %d", multiplier)
		}
	}

	total, _ := ctx.Value(benchmarkCostKey{}).(*uint)
	if total != nil {
		*total += uint(args.Weight) * uint(multiplier)
	}
	return nil
}

func benchmarkThresholdHook(threshold uint) graphql.PreExecHookFunc {
	return func(ctx context.Context, _ *ast.ExecutableDefinition, _ *ast.OperationDefinition, _ map[string]any) error {
		total, _ := ctx.Value(benchmarkCostKey{}).(*uint)
		if total != nil && *total > threshold {
			return fmt.Errorf("query cost %d exceeds threshold %d", *total, threshold)
		}
		return nil
	}
}

const benchmarkCostThreshold = 80

var benchmarkCostResponseSink *graphql.Response

func BenchmarkDirectiveVisitors_CostComplexityThreshold(b *testing.B) {
	schema := graphql.MustParseSchema(
		starwarsCostBenchmarkSchema,
		&starwars.Resolver{},
		graphql.DirectiveVisitors(&benchmarkCostDirective{}),
		graphql.PreExecHook(benchmarkThresholdHook(benchmarkCostThreshold)),
	)

	queries := []struct {
		name            string
		query           string
		expectRejection bool
	}{
		{
			name:            "SmallAllowed",
			query:           `{ hero { id name } }`,
			expectRejection: false,
		},
		{
			name: "ConnectionAllowed",
			query: `{
				hero {
					friendsConnection(first: 10) {
						totalCount
						edges { node { id name } }
					}
				}
			}`,
			expectRejection: false,
		},
		{
			name: "SearchAllowed",
			query: `{
				search(text: "R2") {
					... on Droid { id name primaryFunction }
					... on Human { id name starships { id name } }
				}
			}`,
			expectRejection: false,
		},
		{
			name: "TooExpensiveRejected",
			query: `{
				hero {
					friendsConnection(first: 500) {
						totalCount
						edges { node { id name } }
					}
				}
			}`,
			expectRejection: true,
		},
	}

	for _, tc := range queries {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var total uint
				ctx := context.WithValue(context.Background(), benchmarkCostKey{}, &total)
				resp := schema.Exec(ctx, tc.query, "", nil)
				benchmarkCostResponseSink = resp

				rejected := len(resp.Errors) != 0
				if rejected != tc.expectRejection {
					b.Fatalf("unexpected rejection=%t errors=%v", rejected, resp.Errors)
				}
				if tc.expectRejection && !strings.Contains(resp.Errors[0].Message, "exceeds threshold") {
					b.Fatalf("unexpected rejection error: %+v", resp.Errors)
				}
			}
		})
	}
}
