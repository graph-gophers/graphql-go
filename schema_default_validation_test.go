package graphql_test

import (
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

func TestSchemaRejectsInvalidTypedDefaults(t *testing.T) {
	cases := []struct {
		name   string
		schema string
	}{
		{
			name:   "non-null argument default null",
			schema: `type Query { f1(arg1: Int! = null): Int! }`,
		},
		{
			name: "list item non-null default null",
			schema: `
				input InputObj { arg1: [String!] = [null] }
				type Query { f1(input: InputObj): Int }
			`,
		},
		{
			name: "unknown enum value default",
			schema: `
				enum Language { EN JA }
				directive @localize(lang: Language = ZH) on FIELD
				type Query { f1: Int }
			`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := graphql.ParseSchema(tt.schema, nil); err == nil {
				t.Fatal("expected schema parse to reject invalid default value")
			}
		})
	}
}
