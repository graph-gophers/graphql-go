package graphql_test

import (
	"context"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type helloWorldResolver1 struct{}

func (r *helloWorldResolver1) Hello() string {
	return "Hello world!"
}

type helloWorldResolver2 struct{}

func (r *helloWorldResolver2) Hello(ctx context.Context) (string, error) {
	return "Hello world!", nil
}

type helloSnakeResolver1 struct{}

func (r *helloSnakeResolver1) HelloHTML() string {
	return "Hello snake!"
}

func (r *helloSnakeResolver1) SayHello(args struct{ FullName string }) string {
	return "Hello " + args.FullName + "!"
}

type helloSnakeResolver2 struct{}

func (r *helloSnakeResolver2) HelloHTML(ctx context.Context) (string, error) {
	return "Hello snake!", nil
}

func (r *helloSnakeResolver2) SayHello(ctx context.Context, args struct{ FullName string }) (string, error) {
	return "Hello " + args.FullName + "!", nil
}

type theNumberResolver struct {
	number int32
}

func (r *theNumberResolver) TheNumber() int32 {
	return r.number
}

func (r *theNumberResolver) ChangeTheNumber(args struct{ NewNumber int32 }) *theNumberResolver {
	r.number = args.NewNumber
	return r
}

type timeResolver struct{}

func (r *timeResolver) AddHour(args struct{ Time graphql.Time }) graphql.Time {
	return graphql.Time{Time: args.Time.Add(time.Hour)}
}

func TestHelloWorld_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello: String!
				}
			`, &helloWorldResolver1{}),
			Query: `
				{
					hello
				}
			`,
			ExpectedResult: `
				{
					"hello": "Hello world!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello: String!
				}
			`, &helloWorldResolver2{}),
			Query: `
				{
					hello
				}
			`,
			ExpectedResult: `
				{
					"hello": "Hello world!"
				}
			`,
		},
	})
}

func TestHelloSnake_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello_html: String!
				}
			`, &helloSnakeResolver1{}),
			Query: `
				{
					hello_html
				}
			`,
			ExpectedResult: `
				{
					"hello_html": "Hello snake!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					hello_html: String!
				}
			`, &helloSnakeResolver2{}),
			Query: `
				{
					hello_html
				}
			`,
			ExpectedResult: `
				{
					"hello_html": "Hello snake!"
				}
			`,
		},
	})
}

func TestHelloSnakeArguments_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					say_hello(full_name: String!): String!
				}
			`, &helloSnakeResolver1{}),
			Query: `
				{
					say_hello(full_name: "Rob Pike")
				}
			`,
			ExpectedResult: `
				{
					"say_hello": "Hello Rob Pike!"
				}
			`,
		},

		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					say_hello(full_name: String!): String!
				}
			`, &helloSnakeResolver2{}),
			Query: `
				{
					say_hello(full_name: "Rob Pike")
				}
			`,
			ExpectedResult: `
				{
					"say_hello": "Hello Rob Pike!"
				}
			`,
		},
	})
}

type testDeprecatedDirectiveResolver struct{}

func (r *testDeprecatedDirectiveResolver) A() int32 {
	return 0
}

func (r *testDeprecatedDirectiveResolver) B() int32 {
	return 0
}

func (r *testDeprecatedDirectiveResolver) C() int32 {
	return 0
}

func TestDeprecatedDirective_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					a: Int!
					b: Int! @deprecated
					c: Int! @deprecated(reason: "We don't like it")
				}
			`, &testDeprecatedDirectiveResolver{}),
			Query: `
				{
					__type(name: "Query") {
						fields {
							name
						}
						allFields: fields(includeDeprecated: true) {
							name
							isDeprecated
							deprecationReason
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"__type": {
						"fields": [
							{ "name": "a" }
						],
						"allFields": [
							{ "name": "a", "isDeprecated": false, "deprecationReason": null },
							{ "name": "b", "isDeprecated": true, "deprecationReason": "No longer supported" },
							{ "name": "c", "isDeprecated": true, "deprecationReason": "We don't like it" }
						]
					}
				}
			`,
		},
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
				}

				enum Test {
					A
					B @deprecated
					C @deprecated(reason: "We don't like it")
				}
			`, &testDeprecatedDirectiveResolver{}),
			Query: `
				{
					__type(name: "Test") {
						enumValues {
							name
						}
						allEnumValues: enumValues(includeDeprecated: true) {
							name
							isDeprecated
							deprecationReason
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"__type": {
						"enumValues": [
							{ "name": "A" }
						],
						"allEnumValues": [
							{ "name": "A", "isDeprecated": false, "deprecationReason": null },
							{ "name": "B", "isDeprecated": true, "deprecationReason": "No longer supported" },
							{ "name": "C", "isDeprecated": true, "deprecationReason": "We don't like it" }
						]
					}
				}
			`,
		},
	})
}

func TestMutationOrder_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
					mutation: Mutation
				}

				type Query {
					theNumber: Int!
				}

				type Mutation {
					changeTheNumber(newNumber: Int!): Query
				}
			`, &theNumberResolver{}),
			Query: `
				mutation {
					first: changeTheNumber(newNumber: 1) {
						theNumber
					}
					second: changeTheNumber(newNumber: 3) {
						theNumber
					}
					third: changeTheNumber(newNumber: 2) {
						theNumber
					}
				}
			`,
			ExpectedResult: `
				{
					"first": {
						"theNumber": 1
					},
					"second": {
						"theNumber": 3
					},
					"third": {
						"theNumber": 2
					}
				}
			`,
		},
	})
}

func TestTime_Legacy(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				schema {
					query: Query
				}

				type Query {
					addHour(time: Time = "2001-02-03T04:05:06Z"): Time!
				}

				scalar Time
			`, &timeResolver{}),
			Query: `
				query($t: Time!) {
					a: addHour(time: $t)
					b: addHour
				}
			`,
			Variables: map[string]interface{}{
				"t": time.Date(2000, 2, 3, 4, 5, 6, 0, time.UTC),
			},
			ExpectedResult: `
				{
					"a": "2000-02-03T05:05:06Z",
					"b": "2001-02-03T05:05:06Z"
				}
			`,
		},
	})
}

type resolverWithUnexportedMethod struct{}

func (r *resolverWithUnexportedMethod) changeTheNumber(args struct{ NewNumber int32 }) int32 {
	return args.NewNumber
}

func TestUnexportedMethod_Legacy(t *testing.T) {
	_, err := graphql.ParseSchema(`
		schema {
			mutation: Mutation
		}

		type Mutation {
			changeTheNumber(newNumber: Int!): Int!
		}
	`, &resolverWithUnexportedMethod{})
	if err == nil {
		t.Error("error expected")
	}
}

type resolverWithUnexportedField struct{}

func (r *resolverWithUnexportedField) ChangeTheNumber(args struct{ newNumber int32 }) int32 {
	return args.newNumber
}

func TestUnexportedField(t *testing.T) {
	_, err := graphql.ParseSchema(`
		schema {
			mutation: Mutation
		}

		type Mutation {
			changeTheNumber(newNumber: Int!): Int!
		}
	`, &resolverWithUnexportedField{})
	if err == nil {
		t.Error("error expected")
	}
}