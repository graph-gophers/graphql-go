package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/graph-gophers/graphql-go"
	qerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

const separateSchemaString = `
	schema {
		query: Query
		mutation: Mutation
		subscription: Subscription
	}

	type Subscription {
		hello: HelloEvent!
	}

	type HelloEvent {
		msg: String!
	}

	type Query {
		hello: String!
	}

	type Mutation {
		hello: String!
	}
`

type RootResolver struct{}
type QueryResolver struct{}
type MutationResolver struct{}
type SubscriptionResolver struct {
	err      error
	upstream <-chan *helloEventResolver
}

func (r *RootResolver) Query() interface{} {
	return &QueryResolver{}
}

func (r *RootResolver) Mutation() interface{} {
	return &MutationResolver{}
}

type helloEventResolver struct {
	msg string
	err error
}

func (r *helloEventResolver) Msg() (string, error) {
	return r.msg, r.err
}

func closedHelloEventUpstream(rr ...*helloEventResolver) <-chan *helloEventResolver {
	c := make(chan *helloEventResolver, len(rr))
	for _, r := range rr {
		c <- r
	}
	close(c)
	return c
}

func (r *RootResolver) Subscription() interface{} {
	return &SubscriptionResolver{
		upstream: closedHelloEventUpstream(
			&helloEventResolver{msg: "Hello world!"},
			&helloEventResolver{err: errors.New("resolver error")},
			&helloEventResolver{msg: "Hello again!"},
		),
	}
}

func (qr *QueryResolver) Hello() string {
	return "Hello world!"
}

func (mr *MutationResolver) Hello() string {
	return "Hello world!"
}

func (sr *SubscriptionResolver) Hello(ctx context.Context) (chan *helloEventResolver, error) {
	if sr.err != nil {
		return nil, sr.err
	}

	c := make(chan *helloEventResolver)
	go func() {
		for r := range sr.upstream {
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

var separateSchema = graphql.MustParseSchema(separateSchemaString, &RootResolver{})

func TestSeparateQuery(t *testing.T) {
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: separateSchema,
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
			Schema: graphql.MustParseSchema(separateSchemaString, &RootResolver{}),
			Query: `
				mutation {
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

func TestSeparateSubscription(t *testing.T) {
	gqltesting.RunSubscribes(t, []*gqltesting.TestSubscription{
		{
			Name:   "ok",
			Schema: separateSchema,
			Query: `
				subscription onHello {
					hello {
						msg
					}
				}
			`,
			ExpectedResults: []gqltesting.TestResponse{
				{
					Data: json.RawMessage(`
						{
							"hello": {
								"msg": "Hello world!"
							}
						}
					`),
				},
				{
					Data: json.RawMessage(`
						{
							"hello": {
								"msg":null
							}
						}
					`),
					Errors: []*qerrors.QueryError{qerrors.Errorf("%s", resolverErr)},
				},
				{
					Data: json.RawMessage(`
						{
							"hello": {
								"msg": "Hello again!"
							}
						}
					`),
				},
			},
		},
	})
}
