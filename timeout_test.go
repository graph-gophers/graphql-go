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

func (r *TimeoutTest) MetaMessage() MetaMessage {
	m := MetaMessage{}
	return m
}

type MetaMessage struct {
}

func (m MetaMessage) Msg(args struct{ Timeout int32 }) string {
	time.Sleep(time.Duration(args.Timeout) * time.Millisecond)
	s := "MetaSuccess!"
	return s
}

func TestPartialSuccessTimeout(t *testing.T) {
	cxt, _ := context.WithDeadline(context.Background(), time.Now().Add(20*time.Millisecond)) // This test now depends on the simplest resolver returning within 10 milliseconds
	gqltesting.RunTest(t,
		&gqltesting.Test{ // test that one feature will sucessfully return
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
		})
}

func TestPartialSuccessCancel(t *testing.T) {

	cxt2, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(50*time.Millisecond))
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancelFunc()
	}()
	gqltesting.RunTest(t,
		&gqltesting.Test{
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
			ExpectedErrors: []*qerrors.QueryError{qerrors.Errorf("context canceled")},
			Context:        cxt2,
		})
}

func TestTotalFailureOnNonNull(t *testing.T) {
	cxt3, _ := context.WithDeadline(context.Background(), time.Now().Add(500*time.Millisecond)) // This test now depends on the simplest resolver returning within 10 milliseconds
	gqltesting.RunTest(t,
		&gqltesting.Test{ // test that when we timeout on non-nullable fields we return valid JSON
			Schema: graphql.MustParseSchema(`
			schema {
				query: Query
			}

			type MetaMessage {
				Msg(Timeout: Int!): String!
			}

			type Query {
				MetaMessage: MetaMessage!
			}
		`, &TimeoutTest{}),
			Query: `
			{
			MetaMessage {
					m1: Msg(Timeout: 1000)
					m2: Msg(Timeout: 1000)
				}
			}
		`,
			ExpectedResult: ` { "MetaMessage": {"m1":null, "m2": null }}`,
			ExpectedErrors: []*qerrors.QueryError{qerrors.Errorf("context deadline exceeded")},
			Context:        cxt3,
		})
}
