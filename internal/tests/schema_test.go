package tests

import (
	"reflect"
	"testing"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/schema"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		schema string
		err    error
	}{
		{
			schema: `type Foo {
  baz: Baz
}
type Baz {
	id: ID!
}
`,
		},
		{
			schema: `directive @foo on FIELD_DEFINITION
type Foo {
	foo: Foo @foo
}
`,
		},
		{
			schema: `schema {
  query: Query
  mutation: Mutation
  subscription: Subscription
}
type Query {
}
type Mutation {
}
type Subscription {
}
`,
		},
		{
			schema: `schema {
  query: Query
  foo: Query
}
type Query {
}
`,
			err: &errors.QueryError{
				Message:   `unexpected "foo", expected "query", "mutation" or "subscription"`,
				Locations: []errors.Location{{Line: 3, Column: 3}},
			},
		},
		{
			schema: `schema {
  query: Query
	mutation: Mutation
  query: Mutation
}
type Query {
}
type Mutation {
}
`,
			err: &errors.QueryError{
				Message: `"query" type provided more than once`,
				Locations: []errors.Location{
					{Line: 2, Column: 3},
					{Line: 4, Column: 3},
				},
			},
		},
		{
			schema: `type Foo {
  id: ID!
}
type Baz {
  id: ID!
}
type Foo {
  baz: Baz
}
`,
			err: &errors.QueryError{
				Message: `"Foo" defined more than once`,
				Locations: []errors.Location{
					{Line: 1, Column: 6},
					{Line: 7, Column: 6},
				},
			},
		},
		{
			schema: `scalar Time
scalar CountryCode
scalar Time
`,
			err: &errors.QueryError{
				Message: `"Time" defined more than once`,
				Locations: []errors.Location{
					{Line: 1, Column: 8},
					{Line: 3, Column: 8},
				},
			},
		},
		{
			schema: `scalar Time
scalar Boolean
			`,
			err: &errors.QueryError{
				Message:   `built-in type "Boolean" redefined`,
				Locations: []errors.Location{{Line: 2, Column: 8}},
			},
		},
		{
			schema: `type String {
}
`,
			err: &errors.QueryError{
				Message:   `built-in type "String" redefined`,
				Locations: []errors.Location{{Line: 1, Column: 6}},
			},
		},
		{
			schema: `type __Foo {
}
`,
			err: &errors.QueryError{
				Message:   `"__Foo" must not begin with "__", reserved for introspection types`,
				Locations: []errors.Location{{Line: 1, Column: 6}},
			},
		},
		{
			schema: `directive @ignore on FIELD
directive @ignore on FIELD_DEFINITION
`,
			err: &errors.QueryError{
				Message: `"ignore" defined more than once`,
				Locations: []errors.Location{
					{Line: 1, Column: 12},
					{Line: 2, Column: 12},
				},
			},
		},
		{
			schema: `directive @deprecated on FIELD_DEFINITION | ENUM_VALUE`,
			err: &errors.QueryError{
				Message:   `built-in directive "deprecated" redefined`,
				Locations: []errors.Location{{Line: 1, Column: 12}},
			},
		},
		{
			schema: `directive @__foo on FIELD_DEFINITION`,
			err: &errors.QueryError{
				Message:   `"__foo" must not begin with "__", reserved for introspection types`,
				Locations: []errors.Location{{Line: 1, Column: 12}},
			},
		},
	}
	for _, test := range testCases {
		s := schema.New()
		err := s.Parse(test.schema)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("wrong error\nexpected: %v\ngot:      %v", test.err, err)
		}
	}
}
