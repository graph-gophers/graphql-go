package graphql_test

import (
	"context"
	"testing"
	"time"

	graphql "github.com/graph-gophers/graphql-go"
	qerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type TimeoutTest struct {
}

func (r *TimeoutTest) Message(args struct{ Timeout int32 }) *string {
	time.Sleep(time.Duration(args.Timeout) * time.Millisecond)
	s := "Success!"
	return &s
}

func TestSchemaSubscribe_CustomResolverTimeout_(t *testing.T) {
	cxt, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Millisecond)) // This test now depends on the simplest resolver returning within 10 milliseconds
	cxt2, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(50*time.Millisecond))
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancelFunc()
	}()
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
			schema {
				query: Query
			}

			type Query {
				Message(Timeout: Int!): String
			}
		`, &TimeoutTest{}),
			Query: `
			{
			m1: Message(Timeout: 1) 
			m2: Message(Timeout: 2000) 
			}
		`,
			ExpectedResult: ` { "m1": "Success!", "m2": null }`,
			ExpectedErrors: []*qerrors.QueryError{qerrors.Errorf("context deadline exceeded")},
			Context:        cxt,
		},
		{
			Schema: graphql.MustParseSchema(`
			schema {
				query: Query
			}

			type Query {
				Message(Timeout: Int!): String
			}
		`, &TimeoutTest{}),
			Query: `
			{
			m1: Message(Timeout: 20) 
			m2: Message(Timeout: 2000) 
			}
		`,
			ExpectedResult: ` { "m1": null, "m2": null }`,
			ExpectedErrors: []*qerrors.QueryError{qerrors.Errorf("context canceled")},
			Context:        cxt2,
		},
	})
}
