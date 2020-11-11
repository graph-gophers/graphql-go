package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	graphql "github.com/graph-gophers/graphql-go"
	qerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type rootResolver struct {
	*helloResolver
	*helloSaidResolver
	*helloSaidNullableResolver
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

func (r *rootResolver) OtherField(ctx context.Context) <-chan int32 {
	return make(chan int32)
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

type helloSaidNullableResolver struct {
	err      error
	upstream <-chan *helloSaidNullableEventResolver
}

type helloSaidNullableEventResolver struct {
	msg *string
	err error
}

func (r *helloSaidNullableResolver) HelloSaidNullable(ctx context.Context) (chan *helloSaidNullableEventResolver, error) {
	if r.err != nil {
		return nil, r.err
	}

	c := make(chan *helloSaidNullableEventResolver)
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

func (r *helloSaidNullableEventResolver) Msg() (*string, error) {
	return r.msg, r.err
}

func closedUpstreamNullable(rr ...*helloSaidNullableEventResolver) <-chan *helloSaidNullableEventResolver {
	c := make(chan *helloSaidNullableEventResolver, len(rr))
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
					Data: json.RawMessage(`
						null
					`),
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
			},
		},
		{
			Name: "subscription_resolver_can_error_optional_msg",
			Schema: graphql.MustParseSchema(schema, &rootResolver{
				helloSaidNullableResolver: &helloSaidNullableResolver{
					upstream: closedUpstreamNullable(
						&helloSaidNullableEventResolver{err: resolverErr},
					),
				},
			}),
			Query: `
				subscription onHelloSaid {
					helloSaidNullable {
						msg
					}
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`
						{
							"helloSaidNullable": {
								"msg": null
							}
						}
					`),
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
			},
		},
		{
			Name: "subscription_resolver_can_error_optional_event",
			Schema: graphql.MustParseSchema(schema, &rootResolver{
				helloSaidNullableResolver: &helloSaidNullableResolver{err: resolverErr},
			}),
			Query: `
				subscription onHelloSaid {
					helloSaidNullable {
						msg
					}
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`
						{
							"helloSaidNullable": null
						}
					`),
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
			},
		},
		{
			Name:   "schema_without_resolver_errors",
			Schema: graphql.MustParseSchema(schema, nil),
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

func TestRootOperations_invalidSubscriptionSchema(t *testing.T) {
	type args struct {
		Schema string
	}
	type want struct {
		Error string
	}
	testTable := map[string]struct {
		Args args
		Want want
	}{
		"Subscription as incorrect type": {
			Args: args{
				Schema: `
					schema {
						query: Query
						subscription: String
					}
					type Query {
						thing: String
					}
				`,
			},
			Want: want{Error: `root operation "subscription" must be an OBJECT`},
		},
		"Subscription declared by schema, but type not present": {
			Args: args{
				Schema: `
					schema {
						query: Query
						subscription: Subscription
					}
					type Query {
						hello: String!
					}
				`,
			},
			Want: want{Error: `graphql: type "Subscription" not found`},
		},
	}

	for name, tt := range testTable {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := graphql.ParseSchema(tt.Args.Schema, nil)
			if err == nil || err.Error() != tt.Want.Error {
				t.Logf("got:  %v", err)
				t.Logf("want: %s", tt.Want.Error)
				t.Fail()
			}
		})
	}
}

func TestRootOperations_validSubscriptionSchema(t *testing.T) {
	gqltesting.RunSubscribes(t, []*gqltesting.TestSubscription{
		{
			Name: "Default name, schema omitted",
			Schema: graphql.MustParseSchema(`
				type Query {
					hello: String!
				}
				type Subscription {
					helloSaid: HelloSaidEvent!
				}
				type HelloSaidEvent {
					msg: String!
				}
			`, &rootResolver{helloSaidResolver: &helloSaidResolver{upstream: closedUpstream(&helloSaidEventResolver{msg: "Hello world!"})}}),
			Query: `subscription { helloSaid { msg } }`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`{"helloSaid": {"msg": "Hello world!"}}`),
				},
			},
		},
		{
			Name: "Custom name, schema omitted",
			Schema: graphql.MustParseSchema(`
				type Query {
					hello: String!
				}
				type SubscriptionType {
					helloSaid: HelloSaidEvent!
				}
				type HelloSaidEvent {
					msg: String!
				}
			`, &rootResolver{}),
			Query:       `subscription { helloSaid { msg } }`,
			ExpectedErr: errors.New("no subscriptions are offered by the schema"),
		},
		{
			Name: "Custom name, schema required",
			Schema: graphql.MustParseSchema(`
					schema {
						query: Query
						subscription: SubscriptionType
					}
					type Query {
						hello: String!
					}
					type SubscriptionType {
						helloSaid: HelloSaidEvent!
					}
					type HelloSaidEvent {
						msg: String!
					}
			`, &rootResolver{helloSaidResolver: &helloSaidResolver{upstream: closedUpstream(&helloSaidEventResolver{msg: "Hello world!"})}}),
			Query: `subscription { helloSaid { msg } }`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`{"helloSaid": {"msg": "Hello world!"}}`),
				},
			},
		},
		{
			Name: "Explicit schema without subscription field",
			Schema: graphql.MustParseSchema(`
					schema {
						query: Query
					}
					type Query {
						hello: String!
					}
					type Subscription {
						helloSaid: HelloSaidEvent!
					}
					type HelloSaidEvent {
						msg: String!
					}
			`, &rootResolver{helloSaidResolver: &helloSaidResolver{upstream: closedUpstream(&helloSaidEventResolver{msg: "Hello world!"})}}),
			Query:       `subscription { helloSaid { msg } }`,
			ExpectedErr: errors.New("no subscriptions are offered by the schema"),
		},
	})
}

func TestError_multiple_subscription_fields(t *testing.T) {
	gqltesting.RunSubscribes(t, []*gqltesting.TestSubscription{
		{
			Name: "Explicit schema without subscription field",
			Schema: graphql.MustParseSchema(`
					schema {
						query: Query
						subscription: Subscription
					}
					type Query {
						hello: String!
					}
					type Subscription {
						helloSaid: HelloSaidEvent!
						otherField: Int!
					}
					type HelloSaidEvent {
						msg: String!
					}
			`, &rootResolver{helloSaidResolver: &helloSaidResolver{upstream: closedUpstream(&helloSaidEventResolver{msg: "Hello world!"})}}),
			Query: `subscription { helloSaid { msg } otherField }`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Errors: []*qerrors.QueryError{qerrors.Errorf("can subscribe to at most one subscription at a time")},
				},
			},
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
		helloSaidNullable: HelloSaidEventNullable
	}

	type HelloSaidEvent {
		msg: String!
	}

	type HelloSaidEventNullable {
		msg: String
	}

	type Query {
		hello: String!
	}
`

type subscriptionsCustomTimeout struct{}

type messageResolver struct{}

func (r messageResolver) Msg() string {
	time.Sleep(5 * time.Millisecond)
	return "failed!"
}

func (r *subscriptionsCustomTimeout) OnTimeout() <-chan *messageResolver {
	c := make(chan *messageResolver)
	go func() {
		c <- &messageResolver{}
		close(c)
	}()

	return c
}

func TestSchemaSubscribe_CustomResolverTimeout(t *testing.T) {
	r := &struct {
		*subscriptionsCustomTimeout
	}{
		subscriptionsCustomTimeout: &subscriptionsCustomTimeout{},
	}
	gqltesting.RunSubscribe(t, &gqltesting.TestSubscription{
		Schema: graphql.MustParseSchema(`
			type Query {}
			type Subscription {
				onTimeout : Message!
			}

			type Message {
				msg: String!
			}
		`, r, graphql.SubscribeResolverTimeout(1*time.Millisecond)),
		Query: `
			subscription {
				onTimeout { msg }
			}
		`,
		ExpectedResults: []gqltesting.TestResponse{
			{Errors: []*qerrors.QueryError{{Message: "context deadline exceeded"}}},
		},
	})
}
