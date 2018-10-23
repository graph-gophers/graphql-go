package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	qerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type rootResolver struct {
	*helloResolver
	*helloSaidResolver
}

type helloResolver struct{}

func (r *helloResolver) Hello() string {
	return "Hello world!"
}

var resolverErr = errors.New("resolver error")

type helloSaidResolver struct {
	err      error
	upstream <-chan *helloSaidEventResolver
}

type helloSaidEventResolver struct {
	msg string
	err error
}

func (r *helloSaidResolver) HelloSaid(ctx context.Context) (chan *helloSaidEventResolver, error) {
	if r.err != nil {
		return nil, r.err
	}

	c := make(chan *helloSaidEventResolver)
	go func() {
		for r := range r.upstream {
			select {
			case <-ctx.Done():
				close(c)
				return
			case c <- r:
			}
		}
		close(c)
	}()

	return c, nil
}

func (r *helloSaidEventResolver) Msg() (string, error) {
	return r.msg, r.err
}

func closedUpstream(rr ...*helloSaidEventResolver) <-chan *helloSaidEventResolver {
	c := make(chan *helloSaidEventResolver, len(rr))
	for _, r := range rr {
		c <- r
	}
	close(c)
	return c
}

func TestSchemaSubscribe(t *testing.T) {
	gqltesting.RunSubscribes(t, []*gqltesting.TestSubscription{
		{
			Name: "ok",
			Schema: graphql.MustParseSchema(schema, &rootResolver{
				helloSaidResolver: &helloSaidResolver{
					upstream: closedUpstream(
						&helloSaidEventResolver{msg: "Hello world!"},
						&helloSaidEventResolver{err: resolverErr},
						&helloSaidEventResolver{msg: "Hello again!"},
					),
				},
			}),
			Query: `
				subscription onHelloSaid {
					helloSaid {
						msg
					}
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`
						{
							"helloSaid": {
								"msg": "Hello world!"
							}
						}
					`),
				},
				{
					Data: json.RawMessage(`
						null
					`),
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
				{
					Data: json.RawMessage(`
						{
							"helloSaid": {
								"msg": "Hello again!"
							}
						}
					`),
				},
			},
		},
		{
			Name:   "parse_errors",
			Schema: graphql.MustParseSchema(schema, &rootResolver{}),
			Query:  `invalid graphQL query`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", `syntax error: unexpected "invalid", expecting "fragment" (line 1, column 9)`)},
				},
			},
		},
		{
			Name:   "subscribe_to_query_succeeds",
			Schema: graphql.MustParseSchema(schema, &rootResolver{}),
			Query: `
				query Hello {
					hello
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`
						{
							"hello": "Hello world!"
						}
					`),
				},
			},
		},
		{
			Name: "subscription_resolver_can_error",
			Schema: graphql.MustParseSchema(schema, &rootResolver{
				helloSaidResolver: &helloSaidResolver{err: resolverErr},
			}),
			Query: `
				subscription onHelloSaid {
					helloSaid {
						msg
					}
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
			},
		},
		{
			Name:   "schema_without_resolver_errors",
			Schema: &graphql.Schema{},
			Query: `
				subscription onHelloSaid {
					helloSaid {
						msg
					}
				}
			`,
			ExpectedErr: errors.New("schema created without resolver, can not subscribe"),
		},
	})
}

const schema = `
	schema {
		subscription: Subscription,
		query: Query
	}

	type Subscription {
		helloSaid: HelloSaidEvent!
	}

	type HelloSaidEvent {
		msg: String!
	}

	type Query {
		hello: String!
	}
`
