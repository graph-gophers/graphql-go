package schema

import (
	"reflect"
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
		entryPoint *EntryPoint
		err        error
	}{
		{
			entryPoint: &EntryPoint{Name: "subscription", Type: "Subscription", Loc: errors.Location{Line: 3, Column: 6}},
		},
		{
			entryPoint: &EntryPoint{Name: "foo", Type: "Query", Loc: errors.Location{Line: 322, Column: 3}},
			err: &errors.QueryError{
				Message:   `unexpected "foo", expected "query", "mutation" or "subscription"`,
				Locations: []errors.Location{{Line: 322, Column: 3}},
			},
		},
		{
			entryPoint: &EntryPoint{Name: "query", Type: "Query", Loc: errors.Location{Line: 322, Column: 6}},
			err: &errors.QueryError{
				Message:   `"query" operation provided more than once`,
				Locations: []errors.Location{{Line: 1, Column: 6}, {Line: 322, Column: 6}},
			},
		},
	}

	for _, test := range testCases {
		err := validateEntryPointName(schema, test.entryPoint)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("wrong error\nexpected: %v\ngot:      %v", test.err, err)
		}
	}
}

func TestValidateTypeName(t *testing.T) {
	testCases := []struct {
		schema    *Schema
		namedType NamedType
		err       error
	}{
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Foo":  &Object{Name: "Foo", Loc: errors.Location{Line: 1, Column: 6}},
					"Time": &Scalar{Name: "Time", Loc: errors.Location{Line: 2, Column: 8}},
				},
			},
			namedType: &Object{Name: "Baz", Loc: errors.Location{Line: 322, Column: 6}},
		},
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Foo": &Object{Name: "Foo", Loc: errors.Location{Line: 1, Column: 6}},
					"Baz": &Object{Name: "Baz", Loc: errors.Location{Line: 2, Column: 6}},
				},
			},
			namedType: &Object{
				Name: "Foo",
				Loc:  errors.Location{Line: 322, Column: 6},
			},
			err: &errors.QueryError{
				Message:   `"Foo" defined more than once`,
				Locations: []errors.Location{{Line: 1, Column: 6}, {Line: 322, Column: 6}},
			},
		},
		{
			schema: &Schema{
				Types: map[string]NamedType{
					"Time": &Scalar{Name: "Time", Loc: errors.Location{Line: 1, Column: 8}},
				},
			},
			namedType: &Scalar{
				Name: "Time",
				Loc:  errors.Location{Line: 322, Column: 8},
			},
			err: &errors.QueryError{
				Message:   `"Time" defined more than once`,
				Locations: []errors.Location{{Line: 1, Column: 8}, {Line: 322, Column: 8}},
			},
		},
		{
			schema: &Schema{},
			namedType: &Scalar{
				Name: "Boolean",
				Loc:  errors.Location{Line: 322, Column: 8},
			},
			err: &errors.QueryError{
				Message:   `built-in type "Boolean" redefined`,
				Locations: []errors.Location{{Line: 322, Column: 8}},
			},
		},
		{
			schema: &Schema{},
			namedType: &Object{
				Name: "String",
				Loc:  errors.Location{Line: 322, Column: 6},
			},
			err: &errors.QueryError{
				Message:   `built-in type "String" redefined`,
				Locations: []errors.Location{{Line: 322, Column: 6}},
			},
		},
		{
			schema: &Schema{},
			namedType: &Object{
				Name: "__Foo",
				Loc:  errors.Location{Line: 322, Column: 6},
			},
			err: &errors.QueryError{
				Message:   `"__Foo" must not begin with "__", reserved for introspection types`,
				Locations: []errors.Location{{Line: 322, Column: 6}},
			},
		},
	}

	for _, test := range testCases {
		err := validateTypeName(test.schema, test.namedType)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("wrong error\nexpected: %v\ngot:      %v", test.err, err)
		}
	}
}

func TestValidateDirectiveName(t *testing.T) {
	testCases := []struct {
		schema    *Schema
		directive *DirectiveDecl
		err       error
	}{
		{
			schema:    &Schema{},
			directive: &DirectiveDecl{Name: "ignore", Loc: errors.Location{Line: 322, Column: 12}},
		},
		{
			schema: &Schema{
				Directives: map[string]*DirectiveDecl{
					"ignore": {Name: "ignore", Loc: errors.Location{Line: 1, Column: 12}},
				},
			},
			directive: &DirectiveDecl{Name: "ignore", Loc: errors.Location{Line: 322, Column: 12}},
			err: &errors.QueryError{
				Message:   `"ignore" defined more than once`,
				Locations: []errors.Location{{Line: 1, Column: 12}, {Line: 322, Column: 12}},
			},
		},
		{
			schema:    &Schema{},
			directive: &DirectiveDecl{Name: "deprecated", Loc: errors.Location{Line: 322, Column: 12}},
			err: &errors.QueryError{
				Message:   `built-in directive "deprecated" redefined`,
				Locations: []errors.Location{{Line: 322, Column: 12}},
			},
		},
		{
			schema:    &Schema{},
			directive: &DirectiveDecl{Name: "__foo", Loc: errors.Location{Line: 322, Column: 12}},
			err: &errors.QueryError{
				Message:   `"__foo" must not begin with "__", reserved for introspection types`,
				Locations: []errors.Location{{Line: 322, Column: 12}},
			},
		},
	}

	for _, test := range testCases {
		err := validateDirectiveName(test.schema, test.directive)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("wrong error\nexpected: %v\ngot:      %v", test.err, err)
		}
	}
}
