package schema

import (
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/types"
)

func TestParseInterfaceDef(t *testing.T) {
	type testCase struct {
		description string
		definition  string
		expected    *types.InterfaceTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses simple interface",
		definition:  "Greeting { field: String }",
		expected: &types.InterfaceTypeDefinition{
			Name:   "Greeting",
			Loc:    errors.Location{1, 1},
			Fields: types.FieldsDefinition{&types.FieldDefinition{Name: "field"}}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.InterfaceTypeDefinition
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
		expected    *types.ObjectTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{{
		description: "Parses type inheriting single interface",
		definition:  "Hello implements World { field: String }",
		expected:    &types.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{1, 1}, InterfaceNames: []string{"World"}},
	}, {
		description: "Parses type inheriting multiple interfaces",
		definition:  "Hello implements Wo & rld { field: String }",
		expected:    &types.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{1, 1}, InterfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Parses type inheriting multiple interfaces with leading ampersand",
		definition:  "Hello implements & Wo & rld { field: String }",
		expected:    &types.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{1, 1}, InterfaceNames: []string{"Wo", "rld"}},
	}, {
		description: "Allows legacy SDL interfaces",
		definition:  "Hello implements Wo, rld { field: String }",
		expected:    &types.ObjectTypeDefinition{Name: "Hello", Loc: errors.Location{1, 1}, InterfaceNames: []string{"Wo", "rld"}},
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.ObjectTypeDefinition
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
		expected    *types.Union
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "Parses a union",
			definition:  "Foo = Bar | Qux | Quux",
			expected: &types.Union{
				Name:      "Foo",
				TypeNames: []string{"Bar", "Qux", "Quux"},
				Loc:       errors.Location{1, 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.Union
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
		expected    *types.EnumTypeDefinition
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "parses EnumTypeDefinition on single line",
			definition:  "Foo { BAR QUX }",
			expected: &types.EnumTypeDefinition{
				Name: "Foo",
				EnumValuesDefinition: []*types.EnumValueDefinition{
					{
						EnumValue: "BAR",
						Loc:       errors.Location{1, 7},
					},
					{
						EnumValue: "QUX",
						Loc:       errors.Location{1, 11},
					},
				},
				Loc: errors.Location{1, 1},
			},
		},
		{
			description: "parses EnumtypeDefinition with new lines",
			definition: `Foo { 
				BAR
				QUX
			}`,
			expected: &types.EnumTypeDefinition{
				Name: "Foo",
				EnumValuesDefinition: []*types.EnumValueDefinition{
					{
						EnumValue: "BAR",
						Loc:       errors.Location{2, 5},
					},
					{
						EnumValue: "QUX",
						Loc:       errors.Location{3, 5},
					},
				},
				Loc: errors.Location{1, 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.EnumTypeDefinition
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
		expected    *types.DirectiveDefinition
		err         *errors.QueryError
	}

	tests := []*testCase{
		{
			description: "parses DirectiveDefinition",
			definition:  "@Foo on FIELD",
			expected: &types.DirectiveDefinition{
				Name:      "Foo",
				Loc:       errors.Location{1, 2},
				Locations: []string{"FIELD"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.DirectiveDefinition
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
		expected    *types.InputObject
		err         *errors.QueryError
	}

	tests := []testCase{
		{
			description: "parses an input object type definition",
			definition:  "Foo { qux: String }",
			expected: &types.InputObject{
				Name:   "Foo",
				Values: nil,
				Loc:    errors.Location{1, 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var actual *types.InputObject
			lex := setup(t, test.definition)

			parse := func() { actual = parseInputDef(lex) }
			err := lex.CatchSyntaxError(parse)

			compareErrors(t, test.err, err)
			compareInputObjectTypeDefinition(t, test.expected, actual)
		})
	}
}

func compareDirectiveDefinitions(t *testing.T, expected *types.DirectiveDefinition, actual *types.DirectiveDefinition) {
	t.Helper()

	checkNilCase(t, expected, actual)

	if expected.Name != actual.Name {
		t.Fatalf("wrong DirectiveDefinition name: want %q, got %q", expected.Name, actual.Name)
	}

	if !reflect.DeepEqual(expected.Locations, actual.Locations) {
		t.Errorf("wrong DirectiveDefinition locations: want %v, got %v", expected.Locations, actual.Locations)
	}

	compareLoc(t, "DirectiveDefinition", expected.Loc, actual.Loc)
}

func compareInputObjectTypeDefinition(t *testing.T, expected, actual *types.InputObject) {
	t.Helper()

	checkNilCase(t, expected, actual)

	if expected.Name != actual.Name {
		t.Fatalf("wrong InputObject name: want %q, got %q", expected.Name, actual.Name)
	}

	compareLoc(t, "InputObjectTypeDefinition", expected.Loc, actual.Loc)
}

func compareEnumTypeDefs(t *testing.T, expected, actual *types.EnumTypeDefinition) {
	t.Helper()

	checkNilCase(t, expected, actual)

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

func compareInterfaces(t *testing.T, expected, actual *types.InterfaceTypeDefinition) {
	t.Helper()

	checkNilCase(t, expected, actual)

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

func compareUnions(t *testing.T, expected, actual *types.Union) {
	t.Helper()

	checkNilCase(t, expected, actual)

	if expected.Name != actual.Name {
		t.Errorf("wrong object name: want %q, got %q", expected.Name, actual.Name)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("wrong type names: want %v, got %v", expected.TypeNames, actual.TypeNames)
	}
}

func compareObjects(t *testing.T, expected, actual *types.ObjectTypeDefinition) {
	t.Helper()

	checkNilCase(t, expected, actual)

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

func checkNilCase(t *testing.T, a, b interface{}) {
	t.Helper()

	switch {
	case a == nil && a == b:
		return
	case a == nil && b != nil:
		t.Fatalf("wanted nil, got an unexpected result: %#v", b)
	case a != nil && b == nil:
		t.Fatalf("wanted non-nil result, got nil")
	}
}

func setup(t *testing.T, def string) *common.Lexer {
	t.Helper()

	lex := common.NewLexer(def, false)
	lex.ConsumeWhitespace()

	return lex
}
