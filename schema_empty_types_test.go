package graphql_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestSchemaEmptyTypeDefinitions(t *testing.T) {
	cases := []struct {
		name    string
		sdl     string
		wantErr bool
	}{
		{
			name:    "empty object type",
			sdl:     `type Query { dummy: Int } type Empty { }`,
			wantErr: true,
		},
		{
			name:    "empty interface type",
			sdl:     `type Query { dummy: Int } interface EmptyInterface { }`,
			wantErr: true,
		},
		{
			name:    "empty input object type",
			sdl:     `type Query { dummy(arg: EmptyInput): Int } input EmptyInput { }`,
			wantErr: true,
		},
		{
			name:    "valid types (controls)",
			sdl:     `type Query { dummy: Int } interface Node { id: ID! } input Something { v: Int }`,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := graphql.ParseSchema(tc.sdl, nil)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %s, got none", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
		})
	}
}
