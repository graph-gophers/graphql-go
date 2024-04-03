package schema

import (
	"reflect"
	"testing"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

func TestParseSchemaDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.SchemaDefinition
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "Parses sdl without schema definition",
			definition: `
			type Query {
				hello: String!
			}
			`,
			expected: &ast.SchemaDefinition{Present: false},
		},
		{
			description: "Schema definition present",
			definition: `
			schema {
				query: Query
			}
			type Query{
				hello: String!
			}
			`,
			expected: &ast.SchemaDefinition{Present: true, Loc: errors.Location{Line: 2, Column: 11}},
		},
		{
			description: "Schema definition present and has comment",
			definition: `
			"""
			My cool schema.
			"""
			schema {
				query: Query
			}
			type Query{
				hello: String!
			}
			`,
			expected: &ast.SchemaDefinition{
				Desc:    "My cool schema.",
				Present: true,
				Loc:     errors.Location{Line: 5, Column: 11},
			},
		},
		{
			description: "Schema definition present with comment and directives",
			definition: `
			"""
			My cool schema.
			"""
			schema @dir1(arg1: "Val1", arg2: 5) {
				query: Query
			}
			type Query{
				hello: String!
			}
			`,
			expected: &ast.SchemaDefinition{
				Desc: "My cool schema.",
				Directives: ast.DirectiveList{
					&ast.Directive{
						Arguments: ast.ArgumentList{
							{
								Name: ast.Ident{
									Name: "arg1",
									Loc:  errors.Location{Line: 5, Column: 17},
								},
								Value: &ast.PrimitiveValue{
									Type: scanner.String,
									Text: `"Val1"`,
									Loc:  errors.Location{Line: 5, Column: 23},
								},
							},
							{
								Name: ast.Ident{
									Name: "arg2",
									Loc:  errors.Location{Line: 5, Column: 31},
								},
								Value: &ast.PrimitiveValue{
									Type: scanner.Int,
									Text: "5",
									Loc:  errors.Location{Line: 5, Column: 37},
								},
							},
						},
						Name: ast.Ident{
							Name: "dir1",
							Loc:  errors.Location{Line: 5, Column: 11},
						},
					},
				},
				Loc:     errors.Location{Line: 5, Column: 11},
				Present: true,
			},
		},
		{
			description: "Schema definition present with directives",
			definition: `
			schema @dir3(a: 5) @dir4(b: 1) {
				query: Query
			}
			type Query{
				hello: String!
			}
			`,
			expected: &ast.SchemaDefinition{
				Directives: ast.DirectiveList{
					&ast.Directive{
						Arguments: ast.ArgumentList{
							{
								Name: ast.Ident{
									Name: "a",
									Loc:  errors.Location{Line: 2, Column: 17},
								},
								Value: &ast.PrimitiveValue{
									Type: scanner.Int,
									Text: "5",
									Loc:  errors.Location{Line: 2, Column: 20},
								},
							},
						},
						Name: ast.Ident{
							Name: "dir3",
							Loc:  errors.Location{Line: 2, Column: 11},
						},
					},
					&ast.Directive{
						Arguments: ast.ArgumentList{
							{
								Name: ast.Ident{
									Name: "b",
									Loc:  errors.Location{Line: 2, Column: 29},
								},
								Value: &ast.PrimitiveValue{
									Type: scanner.Int,
									Text: "1",
									Loc:  errors.Location{Line: 2, Column: 32},
								},
							},
						},
						Name: ast.Ident{
							Name: "dir4",
							Loc:  errors.Location{Line: 2, Column: 23},
						},
					},
				},
				Loc:     errors.Location{Line: 2, Column: 11},
				Present: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.SchemaDefinition
			lex := common.NewLexer(test.definition, true)
			parse := func() {
				s := New()
				parseSchema(s, lex)
				actual = &s.SchemaDefinition

			}
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareSchemaDefinitions(t, test.expected, actual)
		})
	}
}

func TestParseInterfaceDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.InterfaceTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses simple interface",
		definition:  "Greeting { field: String }",
		expected: &ast.InterfaceTypeDefinition{
			Name:   "Greeting",
			Loc:    errors.Location{Line: 1, Column: 1},
			Fields: ast.FieldsDefinition{&ast.FieldDefinition{Name: "field"}}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.InterfaceTypeDefinition
			lex := setup(t, test.definition)

			parse := func() { actual = parseInterfaceDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareInterfaces(t, test.expected, actual)
		})
	}
}

// TestParseObjectDef tests the logic for parsing object types from the schema definition as
// written in `parseObjectDef()`.
func TestParseObjectDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.ObjectTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses type inheriting single interface",
		definition:  "Hello implements World { field: String }",
		expected:    &ast.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{Line: 1, Column: 1}, InterfaceNames: []string{"World"}},
	}, {
		description: "Parses type inheriting multiple interfaces",
		definition:  "Hello implements Wo & rld { field: String }",
		expected:    &ast.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{Line: 1, Column: 1}, InterfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Parses type inheriting multiple interfaces with leading ampersand",
		definition:  "Hello implements & Wo & rld { field: String }",
		expected:    &ast.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{Line: 1, Column: 1}, InterfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Allows legacy SDL interfaces",
		definition:  "Hello implements Wo, rld { field: String }",
		expected:    &ast.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{Line: 1, Column: 1}, InterfaceNames: []string{"Wo", "rld"}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.ObjectTypeDefinition
			lex := setup(t, test.definition)

			parse := func() { actual = parseObjectDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareObjects(t, test.expected, actual)
		})
	}
}

func TestParseUnionDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.Union
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "Parses a union",
			definition:  "Foo = Bar | Qux | Quux",
			expected: &ast.Union{
				Name:      "Foo",
				TypeNames: []string{"Bar", "Qux", "Quux"},
				Loc:       errors.Location{Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.Union
			lex := setup(t, test.definition)

			parse := func() { actual = parseUnionDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareUnions(t, test.expected, actual)
		})
	}
}

func TestParseEnumDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.EnumTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "parses EnumTypeDefinition on single line",
			definition:  "Foo { BAR QUX }",
			expected: &ast.EnumTypeDefinition{
				Name: "Foo",
				EnumValuesDefinition: []*ast.EnumValueDefinition{
					{
						EnumValue: "BAR",
						Loc:       errors.Location{Line: 1, Column: 7},
					},
					{
						EnumValue: "QUX",
						Loc:       errors.Location{Line: 1, Column: 11},
					},
				},
				Loc: errors.Location{Line: 1, Column: 1},
			},
		},
		{
			description: "parses EnumtypeDefinition with new lines",
			definition: `Foo { 
				BAR
				QUX
			}`,
			expected: &ast.EnumTypeDefinition{
				Name: "Foo",
				EnumValuesDefinition: []*ast.EnumValueDefinition{
					{
						EnumValue: "BAR",
						Loc:       errors.Location{Line: 2, Column: 5},
					},
					{
						EnumValue: "QUX",
						Loc:       errors.Location{Line: 3, Column: 5},
					},
				},
				Loc: errors.Location{Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.EnumTypeDefinition
			lex := setup(t, test.definition)

			parse := func() { actual = parseEnumDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareEnumTypeDefs(t, test.expected, actual)
		})
	}
}

func TestParseDirectiveDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.DirectiveDefinition
		err         *errors.QueryError
	}

	tests := []*testCase{
		{
			description: "parses DirectiveDefinition",
			definition:  "@Foo on FIELD",
			expected: &ast.DirectiveDefinition{
				Name:      "Foo",
				Loc:       errors.Location{Line: 1, Column: 2},
				Locations: []string{"FIELD"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.DirectiveDefinition
			lex := setup(t, test.definition)

			parse := func() { actual = parseDirectiveDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareDirectiveDefinitions(t, test.expected, actual)
		})
	}
}

func TestParseInputDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *ast.InputObject
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "parses an input object type definition",
			definition:  "Foo { qux: String }",
			expected: &ast.InputObject{
				Name:   "Foo",
				Values: nil,
				Loc:    errors.Location{Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *ast.InputObject
			lex := setup(t, test.definition)

			parse := func() { actual = parseInputDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareInputObjectTypeDefinition(t, test.expected, actual)
		})
	}
}

func compareDirectiveDefinitions(t *testing.T, expected *ast.DirectiveDefinition, actual *ast.DirectiveDefinition) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Fatalf("wrong DirectiveDefinition name: want %q, got %q", expected.Name, actual.Name)
	}

	if !reflect.DeepEqual(expected.Locations, actual.Locations) {
		t.Errorf("wrong DirectiveDefinition locations: want %v, got %v", expected.Locations, actual.Locations)
	}

	compareLoc(t, "DirectiveDefinition", expected.Loc, actual.Loc)
}

func compareInputObjectTypeDefinition(t *testing.T, expected, actual *ast.InputObject) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Fatalf("wrong InputObject name: want %q, got %q", expected.Name, actual.Name)
	}

	compareLoc(t, "InputObjectTypeDefinition", expected.Loc, actual.Loc)
}

func compareEnumTypeDefs(t *testing.T, expected, actual *ast.EnumTypeDefinition) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Fatalf("wrong EnumTypeDefinition name: want %q, got %q", expected.Name, actual.Name)
	}

	compareLoc(t, "EnumValueDefinition", expected.Loc, actual.Loc)

	for i, definition := range expected.EnumValuesDefinition {
		expectedValue, expectedLoc := definition.EnumValue, definition.Loc
		actualDef := actual.EnumValuesDefinition[i]

		if expectedValue != actualDef.EnumValue {
			t.Fatalf("wrong EnumValue: want %q, got %q", expectedValue, actualDef.EnumValue)
		}

		compareLoc(t, "EnumValue "+expectedValue, expectedLoc, actualDef.Loc)
	}
}

func compareLoc(t *testing.T, typeName string, expected, actual errors.Location) {
	t.Helper()
	if expected != actual {
		t.Errorf("wrong location on %s: want %v, got %v", typeName, expected, actual)
	}
}

func compareErrors(t *testing.T, expected, actual *errors.QueryError) {
	t.Helper()

	switch {
	case expected != nil && actual != nil:
		if expected.Message != actual.Message {
			t.Fatalf("wanted error message %q, got %q", expected.Message, actual.Message)
		}
		// TODO: Check error locations are as expected.

	case expected != nil && actual == nil:
		t.Fatalf("missing expected error: %q", expected)

	case expected == nil && actual != nil:
		t.Fatalf("got unexpected error: %q", actual)
	}
}

func compareInterfaces(t *testing.T, expected, actual *ast.InterfaceTypeDefinition) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Errorf("wrong interface name: want %q, got %q", expected.Name, actual.Name)
	}

	compareLoc(t, "InterfaceTypeDefinition", expected.Loc, actual.Loc)

	if len(expected.Fields) != len(actual.Fields) {
		t.Fatalf("wanted %d field definitions, got %d", len(expected.Fields), len(actual.Fields))
	}

	for i, f := range expected.Fields {
		if f.Name != actual.Fields[i].Name {
			t.Errorf("fields[%d]: wrong field name: want %q, got %q", i, f.Name, actual.Fields[i].Name)
		}
	}
}

func compareUnions(t *testing.T, expected, actual *ast.Union) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Errorf("wrong object name: want %q, got %q", expected.Name, actual.Name)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("wrong type names: want %v, got %v", expected.TypeNames, actual.TypeNames)
	}
}

func compareObjects(t *testing.T, expected, actual *ast.ObjectTypeDefinition) {
	t.Helper()

	if expected.Name != actual.Name {
		t.Errorf("wrong object name: want %q, got %q", expected.Name, actual.Name)
	}

	if len(expected.InterfaceNames) != len(actual.InterfaceNames) {
		t.Fatalf(
			"wrong number of interface names: want %s, got %s",
			expected.InterfaceNames,
			actual.InterfaceNames,
		)
	}

	for i, expectedName := range expected.InterfaceNames {
		actualName := actual.InterfaceNames[i]
		if expectedName != actualName {
			t.Errorf("wrong interface name: want %q, got %q", expectedName, actualName)
		}
	}
}

func compareSchemaDefinitions(t *testing.T, expected, actual *ast.SchemaDefinition) {
	t.Helper()

	if expected.Present != actual.Present {
		t.Errorf("wrong boolean Present: want %v, got %v", expected.Present, actual.Present)
	}

	if expected.Desc != actual.Desc {
		t.Errorf("wrong schema Desc: want %q, got %q", expected.Desc, actual.Desc)
	}

	if len(expected.RootOperationTypes) != len(actual.RootOperationTypes) {
		t.Fatalf(
			"wrong number of root operations: want %d, got %d",
			len(expected.RootOperationTypes),
			len(actual.RootOperationTypes),
		)
	}

	for name, expectedOp := range expected.RootOperationTypes {
		actualOp := actual.RootOperationTypes[name]
		if actualOp != expectedOp {
			t.Errorf("wrong root operation name: want %q, got %q", actualOp, expectedOp)
		}
	}

	compareDirectiveList(t, "SchemaDef", expected.Directives, actual.Directives)

	compareLoc(t, "SchemaDef ", expected.Loc, actual.Loc)
}

func compareDirectiveList(t *testing.T, target string, expectedList, actualList ast.DirectiveList) {
	if len(expectedList) != len(actualList) {
		t.Fatalf(
			"wrong number of schema directives on %s: want %d, got %d",
			target,
			len(expectedList),
			len(actualList),
		)
	}

	for i, expected := range expectedList {
		actual := actualList[i]
		if !reflect.DeepEqual(expectedList, actualList) {
			if expected.Name.Name != actual.Name.Name {
				t.Errorf("wrong directive name: want %q, got %q", expected.Name.Name, actual.Name.Name)
			}

			target := "directive " + expected.Name.Name + " on SchemaDefinition"
			compareLoc(t, target, expected.Name.Loc, actual.Name.Loc)
			compareArgumentList(t, target, expected.Arguments, actual.Arguments)
		}
	}
}

func compareArgumentList(t *testing.T, target string, expectedList, actualList ast.ArgumentList) {
	if len(expectedList) != len(actualList) {
		t.Fatalf(
			"wrong number of arguments on %s: want %d, got %d",
			target,
			len(expectedList),
			len(actualList),
		)
	}

	for i, expected := range expectedList {
		actual := actualList[i]
		if !reflect.DeepEqual(expectedList, actualList) {
			if expected.Name.Name != actual.Name.Name {
				t.Errorf("wrong argument name on %s: want %q, got %q", target, expected.Name.Name, actual.Name.Name)
			}

			if expected.Value.String() != actual.Value.String() {
				t.Errorf("wrong argument value on %s: want %q, got %q", target, expected.Value, actual.Value)
			}

			compareDirectiveList(t, "argument "+expected.Name.Name+" on "+target, expected.Directives, actual.Directives)

			compareLoc(t, "argument "+expected.Name.Name+" on "+target, expected.Name.Loc, actual.Name.Loc)

			compareLoc(t, "value on argument "+expected.Name.Name+" on "+target, expected.Value.Location(), actual.Value.Location())
		}
	}
}

func setup(t *testing.T, def string) *common.Lexer {
	t.Helper()

	lex := common.NewLexer(def, false)
	lex.ConsumeWhitespace()

	return lex
}
