package graphql_test

import (
	"encoding/json"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestInterfaceInheritanceIntrospection(t *testing.T) {
	s := graphql.MustParseSchema(`
		interface Named { name: String }
		interface Node implements Named { id: ID! name: String }
		type Query { node: Node }
	`, nil)

	j, err := s.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Schema struct {
			Types []struct {
				Name       string `json:"name"`
				Interfaces []struct {
					Name string `json:"name"`
				} `json:"interfaces"`
			} `json:"types"`
		} `json:"__schema"`
	}
	if err := json.Unmarshal(j, &doc); err != nil {
		t.Fatal(err)
	}
	for _, typ := range doc.Schema.Types {
		if typ.Name == "Node" {
			if len(typ.Interfaces) != 1 || typ.Interfaces[0].Name != "Named" {
				t.Fatalf("Node.interfaces = %v, want [Named]", typ.Interfaces)
			}
			return
		}
	}
	t.Fatal("Node type not found in introspection output")
}
