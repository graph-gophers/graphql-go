package graphql_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

type repeatableResolver struct{}

func (r *repeatableResolver) F() *string { s := "ok"; return &s }

func TestDirectiveIsRepeatableIntrospection(t *testing.T) {
	s := graphql.MustParseSchema(`type Query { f: String }`, &repeatableResolver{})
	resp := s.Exec(context.Background(), `{ __schema { directives { name isRepeatable } } }`, "", nil)
	if len(resp.Errors) != 0 {
		t.Fatalf("introspection query failed: %v", resp.Errors)
	}
	var doc struct {
		Schema struct {
			Directives []struct {
				Name         string `json:"name"`
				IsRepeatable bool   `json:"isRepeatable"`
			} `json:"directives"`
		} `json:"__schema"`
	}
	j, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(j, &doc); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range doc.Schema.Directives {
		if d.Name == "include" {
			found = true
		}
	}
	if !found {
		t.Fatal("built-in @include not found in introspection result")
	}
}
