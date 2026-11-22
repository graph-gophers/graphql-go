package graphql_test

import (
	"encoding/json"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestToJSON_IncludesSpecifiedByURL(t *testing.T) {
	t.Parallel()

	schema := graphql.MustParseSchema(`
		schema {
			query: Query
		}
		type Query {
			hello: String
		}
		scalar UUID @specifiedBy(
			url: "https://tools.ietf.org/html/rfc4122"
		)
	`, nil)

	j, err := schema.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %s", err)
	}

	var result struct {
		Schema struct {
			Types []struct {
				Name           string  `json:"name"`
				SpecifiedByURL *string `json:"specifiedByURL"`
			} `json:"types"`
		} `json:"__schema"`
	}
	if err := json.Unmarshal(j, &result); err != nil {
		t.Fatalf("failed to unmarshal ToJSON output: %s", err)
	}

	for _, typ := range result.Schema.Types {
		if typ.Name == "UUID" {
			if typ.SpecifiedByURL == nil || *typ.SpecifiedByURL != "https://tools.ietf.org/html/rfc4122" {
				t.Fatalf("specifiedByURL = %v, want the RFC 4122 URL", typ.SpecifiedByURL)
			}
			return
		}
	}
	t.Fatal("UUID scalar type not found in ToJSON output")
}
