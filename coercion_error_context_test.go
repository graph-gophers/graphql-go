package graphql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

type coercionCtxResolver struct{}

func (r *coercionCtxResolver) Hello(args struct{ Arg string }) *string { return &args.Arg }

func TestNonNullArgCoercedNullErrorMessage(t *testing.T) {
	s := graphql.MustParseSchema(`type Query { hello(arg: String!): String }`, &coercionCtxResolver{})
	resp := s.Exec(context.Background(), `query($v: String = "hello") { hello(arg: $v) }`, "", map[string]interface{}{"v": nil})
	if len(resp.Errors) == 0 {
		t.Fatal("expected an error for null non-null argument")
	}
	msg := resp.Errors[0].Error()
	if !strings.Contains(msg, `"arg"`) || !strings.Contains(msg, "String!") {
		t.Fatalf("error lacks argument context: %s", msg)
	}
}
