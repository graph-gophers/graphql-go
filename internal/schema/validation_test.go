package schema

import (
	"testing"

	"github.com/graph-gophers/graphql-go/errors"
)

func TestValidateEntryPointName(t *testing.T) {
	schema := &Schema{
		entryPointNames: map[string]*EntryPoint{
			"query": &EntryPoint{
				Name: "query",
				Type: "Query",
				Loc:  errors.Location{Line: 1, Column: 6},
			},
			"mutation": &EntryPoint{
				Name: " mutation",
				Type: "Mutation",
				Loc:  errors.Location{Line: 2, Column: 6},
			},
		},
	}

	testCases := []struct {
		entryPoint string
		err        *errors.QueryError
	}{
		{
			entryPoint: "subscription: Subscription",
		},
		{
			entryPoint: "foo: Query",
			err: &errors.QueryError{
				Message: `syntax error: unexpected "foo", expected "query", "mutation" or "subscription"`,
			},
		},
		{
			entryPoint: "query: Query",
			err: &errors.QueryError{
				Message: `syntax error: "query" provided more than once (line 1, column 6)`,
			},
		},
	}

	for _, test := range testCases {
		_, l := setup(t, test.entryPoint)
		err := l.CatchSyntaxError(func() {
			validateEntryPointName(schema, l)
		})
		compareErrors(t, test.err, err)
	}
}

func TestValidateTypeName(t *testing.T) {
	testCases := []struct {
		schema    *Schema
		namedType string
		err       *errors.QueryError
	}{
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Foo":  &Object{Name: "Foo", Loc: errors.Location{Line: 1, Column: 6}},
					"Time": &Scalar{Name: "Time", Loc: errors.Location{Line: 2, Column: 8}},
				},
			},
			namedType: "Baz",
		},
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Foo": &Object{Name: "Foo", Loc: errors.Location{Line: 1, Column: 6}},
					"Baz": &Object{Name: "Baz", Loc: errors.Location{Line: 2, Column: 6}},
				},
			},
			namedType: "Foo",
			err: &errors.QueryError{
				Message: `syntax error: "Foo" defined more than once (line 1, column 6)`,
			},
		},
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Time": &Scalar{Name: "Time", Loc: errors.Location{Line: 1, Column: 8}},
				},
			},
			namedType: "Time",
			err: &errors.QueryError{
				Message: `syntax error: "Time" defined more than once (line 1, column 8)`,
			},
		},
		{
			schema:    &Schema{},
			namedType: "Boolean",
			err: &errors.QueryError{
				Message: `syntax error: built-in type "Boolean" redefined`,
			},
		},
		{
			schema:    &Schema{},
			namedType: "String",
			err: &errors.QueryError{
				Message: `syntax error: built-in type "String" redefined`,
			},
		},
		{
			schema:    &Schema{},
			namedType: "__Foo",
			err: &errors.QueryError{
				Message: `syntax error: "__Foo" must not begin with "__", reserved for introspection types`,
			},
		},
	}

	for _, test := range testCases {
		_, l := setup(t, test.namedType)
		err := l.CatchSyntaxError(func() {
			validateTypeName(test.schema, l)
		})
		compareErrors(t, test.err, err)
	}
}

func TestValidateDirectiveName(t *testing.T) {
	testCases := []struct {
		schema    *Schema
		directive string
		err       *errors.QueryError
	}{
		{
			schema:    &Schema{},
			directive: "ignore",
		},
		{
			schema: &Schema{
				Directives: map[string]*DirectiveDecl{
					"ignore": {Name: "ignore", Loc: errors.Location{Line: 1, Column: 12}},
				},
			},
			directive: "ignore",
			err: &errors.QueryError{
				Message: `syntax error: "ignore" defined more than once (line 1, column 12)`,
			},
		},
		{
			schema:    &Schema{},
			directive: "deprecated",
			err: &errors.QueryError{
				Message: `syntax error: built-in directive "deprecated" redefined`,
			},
		},
		{
			schema:    &Schema{},
			directive: "__foo",
			err: &errors.QueryError{
				Message: `syntax error: "__foo" must not begin with "__", reserved for introspection types`,
			},
		},
	}

	for _, test := range testCases {
		_, l := setup(t, test.directive)
		err := l.CatchSyntaxError(func() {
			validateDirectiveName(test.schema, l)
		})
		compareErrors(t, test.err, err)
	}
}
