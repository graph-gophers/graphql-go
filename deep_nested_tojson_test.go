package graphql_test

import (
	"encoding/json"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

type deepListResolver struct{}

func (r *deepListResolver) Value() *[]*[]*[]*[]*[]*[]*[]*[]*string { return nil }

// ToJSON's introspection query must describe arbitrarily deep list wrappers.
// A fixed ofType nesting truncates the chain for deeply wrapped types.
func TestToJSONDeeplyWrappedType(t *testing.T) {
	s := graphql.MustParseSchema(`
		type Query {
			value: [[[[[[[[String]]]]]]]]
		}
	`, &deepListResolver{})

	j, err := s.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Schema struct {
			Types []struct {
				Name string          `json:"name"`
				Fields []struct {
					Name string `json:"name"`
					Type map[string]interface{} `json:"type"`
				} `json:"fields"`
			} `json:"types"`
		} `json:"__schema"`
	}
	if err := json.Unmarshal(j, &doc); err != nil {
		t.Fatal(err)
	}

	for _, typ := range doc.Schema.Types {
		if typ.Name != "Query" {
			continue
		}
		for _, f := range typ.Fields {
			if f.Name != "value" {
				continue
			}
			depth := 0
			for cur := f.Type; cur != nil; {
				depth++
				next, ok := cur["ofType"].(map[string]interface{})
				if !ok {
					break
				}
				cur = next
			}
			// 8 list wrappers + the String leaf = 9 type refs.
			if depth < 9 {
				t.Fatalf("ofType chain has depth %d, want >= 9; JSON: %s", depth, truncate(string(j)))
			}
			return
		}
	}
	t.Fatal("Query.value type not found in introspection JSON")
}

func truncate(s string) string {
	if len(s) > 400 {
		return s[:400]
	}
	return s
}
