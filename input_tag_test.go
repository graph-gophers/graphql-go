package graphql_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

type inputTagResolver struct{}

func (r *inputTagResolver) Echo(args struct {
	Input *struct {
		Renamed string `graphql:"original_name"`
	}
}) *string {
	return &args.Input.Renamed
}

func TestInputFieldGraphQLTag(t *testing.T) {
	s := graphql.MustParseSchema(`input Filter { original_name: String! } type Query { echo(input: Filter): String }`, &inputTagResolver{})
	resp := s.Exec(context.Background(), `query($input: Filter) { echo(input: $input) }`, "", map[string]interface{}{
		"input": map[string]interface{}{"original_name": "hello"},
	})
	if len(resp.Errors) != 0 {
		t.Fatalf("query failed: %v", resp.Errors)
	}
	j, _ := json.Marshal(resp.Data)
	if !strings.Contains(string(j), "hello") {
		t.Fatalf("input tag was not honored: %s", j)
	}
}
