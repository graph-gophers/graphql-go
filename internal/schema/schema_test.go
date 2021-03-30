package schema_test

import (
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/types"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name                  string
		sdl                   string
		useStringDescriptions bool
		validateError         func(err error) error
		validateSchema        func(s *types.Schema) error
	}{
		{
			name: "Parses interface definition",
			sdl:  "interface Greeting { message: String! }",
			validateSchema: func(s *types.Schema) error {
				const typeName = "Greeting"
				typ, ok := s.Types[typeName].(*types.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("interface %q not found", typeName)
				}
				if want, have := 1, len(typ.Fields); want != have {
					return fmt.Errorf("invalid number of fields: want %d, have %d", want, have)
				}
				const fieldName = "message"
				if typ.Fields[0].Name != fieldName {
					return fmt.Errorf("field %q not found", fieldName)
				}
				return nil
			},
		},
		{
			name: "Parses implementing type without providing required fields",
			sdl: `
			interface Greeting { 
				message: String!
			} 
			type Welcome implements Greeting {
			}`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `graphql: interface "Greeting" expects field "message" but "Welcome" does not provide it`, err.Error(); want != have {
					return fmt.Errorf("unexpected error: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with description string",
			sdl: `
			"Single line description."
			type Type { 
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				if want, have := "Single line description.", typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with simple multi-line 'BlockString' description",
			sdl: `
			"""
			Multi-line description.
			"""
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				if want, have := "Multi-line description.", typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with empty multi-line 'BlockString' description",
			sdl: `
			"""
			"""
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				if want, have := "", typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with multi-line 'BlockString' description",
			sdl: `
			"""
			First line of the description.

			Second line of the description.

				query {
					code {
						example
					}
				}

			Notes:

			 * First note
			 * Second note
			"""
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				want := "First line of the description.\n\nSecond line of the description.\n\n\tquery {\n\t\tcode {\n\t\t\texample\n\t\t}\n\t}\n\nNotes:\n\n * First note\n * Second note"
				if have := typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with un-indented multi-line 'BlockString' description",
			sdl: `
			"""
First line of the description.

Second line of the description.
			"""
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				want := "First line of the description.\n\nSecond line of the description."
				if have := typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with space-indented multi-line 'BlockString' description",
			sdl: `
            """
            First line of the description.

            Second line of the description.

                query {
                    code {
                        example
                    }
                }
            """
            type Type {
                field: String
            }`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				want := "First line of the description.\n\nSecond line of the description.\n\n    query {\n        code {\n            example\n        }\n    }"
				if have := typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Parses type with multi-line 'BlockString' description and ignores comments",
			sdl: `
			"""
			Multi-line description with ignored comments.
			"""
			# This comment should be ignored.
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", typeName)
				}
				if want, have := "Multi-line description with ignored comments.", typ.Description(); want != have {
					return fmt.Errorf("invalid description: want %q, have %q", want, have)
				}
				return nil
			},
		},
		{
			name: "Description is correctly parsed for non-described types",
			sdl: `
			"Some description."
			scalar MyInt 
			type Type { 
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Type"]
				if !ok {
					return fmt.Errorf("type %q not found", "Type")
				}
				if want, have := "", typ.Description(); want != have {
					return fmt.Errorf("description does not match: want %q, have %q ", want, have)
				}
				return nil
			},
		},
		{
			name: "Multi-line comment is correctly parsed",
			sdl: `
			# Multi-line
			# comment.
			" This description should be ignored. "
			scalar MyInt 
			type Type { 
				field: String
			}`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["MyInt"]
				if !ok {
					return fmt.Errorf("scalar %q not found", "MyInt")
				}
				if want, have := "Multi-line\ncomment.", typ.Description(); want != have {
					return fmt.Errorf("description does not match: want %q, have %q ", want, have)
				}
				typ, ok = s.Types["Type"]
				if !ok {
					return fmt.Errorf("type %q not found", "Type")
				}
				if want, have := "", typ.Description(); want != have {
					return fmt.Errorf("description does not match: want %q, have %q ", want, have)
				}
				return nil
			},
		},
		{
			name: "Default Root schema",
			sdl: `
			type Query {
				hello: String!
			}
			type Mutation {
				concat(a: String!, b: String!): String!
			}
			`,
			validateSchema: func(s *types.Schema) error {
				typq, ok := s.Types["Query"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Query")
				}
				helloField := typq.Fields.Get("hello")
				if helloField == nil {
					return fmt.Errorf("field %q not found", "hello")
				}
				if helloField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "hello", helloField.Type.String())
				}

				typm, ok := s.Types["Mutation"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Mutation")
				}
				concatField := typm.Fields.Get("concat")
				if concatField == nil {
					return fmt.Errorf("field %q not found", "concat")
				}
				if concatField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "concat", concatField.Type.String())
				}
				if len(concatField.Arguments) != 2 || concatField.Arguments[0] == nil || concatField.Arguments[1] == nil || concatField.Arguments[0].Type.String() != "String!" || concatField.Arguments[1].Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid args: %+v", "concat", concatField.Arguments)
				}
				return nil
			},
		},
		{
			name: "Extend type",
			sdl: `
			type Query {
				hello: String!
			}

			extend type Query {
				world: String!
			}`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Query"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Query")
				}

				helloField := typ.Fields.Get("hello")
				if helloField == nil {
					return fmt.Errorf("field %q not found", "hello")
				}
				if helloField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "hello", helloField.Type.String())
				}

				worldField := typ.Fields.Get("world")
				if worldField == nil {
					return fmt.Errorf("field %q not found", "world")
				}
				if worldField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "world", worldField.Type.String())
				}
				return nil
			},
		},
		{
			name: "Extend schema",
			sdl: `
			schema {
				query: Query
			}
			type Query {
				hello: String!
			}
			extend schema {
				mutation: Mutation
			}
			type Mutation {
				concat(a: String!, b: String!): String!
			}
			`,
			validateSchema: func(s *types.Schema) error {
				typq, ok := s.Types["Query"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Query")
				}
				helloField := typq.Fields.Get("hello")
				if helloField == nil {
					return fmt.Errorf("field %q not found", "hello")
				}
				if helloField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "hello", helloField.Type.String())
				}

				typm, ok := s.Types["Mutation"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Mutation")
				}
				concatField := typm.Fields.Get("concat")
				if concatField == nil {
					return fmt.Errorf("field %q not found", "concat")
				}
				if concatField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "concat", concatField.Type.String())
				}
				if len(concatField.Arguments) != 2 || concatField.Arguments[0] == nil || concatField.Arguments[1] == nil || concatField.Arguments[0].Type.String() != "String!" || concatField.Arguments[1].Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid args: %+v", "concat", concatField.Arguments)
				}
				return nil
			},
		},
		{
			name: "Extend type with interface implementation",
			sdl: `
			interface Named {
				name: String!
			}
			type Product {
				id: ID!
			}
			extend type Product implements Named {
				name: String!
			}`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Product"].(*types.ObjectTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Product")
				}
				idField := typ.Fields.Get("id")
				if idField == nil {
					return fmt.Errorf("field %q not found", "id")
				}
				if idField.Type.String() != "ID!" {
					return fmt.Errorf("field %q has an invalid type: %q", "id", idField.Type.String())
				}
				nameField := typ.Fields.Get("name")
				if nameField == nil {
					return fmt.Errorf("field %q not found", "name")
				}
				if nameField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "name", nameField.Type.String())
				}

				ifc, ok := s.Types["Named"].(*types.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Named")
				}
				nameField = ifc.Fields.Get("name")
				if nameField == nil {
					return fmt.Errorf("field %q not found", "name")
				}
				if nameField.Type.String() != "String!" {
					return fmt.Errorf("field %q has an invalid type: %q", "name", nameField.Type.String())
				}
				return nil
			},
		},
		{
			name: "Extend union type",
			sdl: `
			type Named {
				name: String!
			}
			type Numbered {
				num: Int!
			}
			union Item = Named | Numbered
			type Coloured {
				Colour: String!
			}
			extend union Item = Coloured
			`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Item"].(*types.Union)
				if !ok {
					return fmt.Errorf("type %q not found", "Item")
				}
				if len(typ.UnionMemberTypes) != 3 {
					return fmt.Errorf("Expected 3 possible types, but instead got %d types", len(typ.UnionMemberTypes))
				}
				posible := map[string]struct{}{
					"Coloured": struct{}{},
					"Named":    struct{}{},
					"Numbered": struct{}{},
				}
				for _, pt := range typ.UnionMemberTypes {
					if _, ok := posible[pt.Name]; !ok {
						return fmt.Errorf("Unexpected possible type %q", pt.Name)
					}
				}
				return nil
			},
		},
		{
			name: "Extend enum type",
			sdl: `
			enum Currencies{
				AUD
				USD
				EUR
			}
			extend enum Currencies {
				BGN
				GBP
			}
			`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Currencies"].(*types.EnumTypeDefinition)
				if !ok {
					return fmt.Errorf("enum %q not found", "Currencies")
				}
				if len(typ.EnumValuesDefinition) != 5 {
					return fmt.Errorf("Expected 5 enum values, but instead got %d types", len(typ.EnumValuesDefinition))
				}
				posible := map[string]struct{}{
					"AUD": struct{}{},
					"USD": struct{}{},
					"EUR": struct{}{},
					"BGN": struct{}{},
					"GBP": struct{}{},
				}
				for _, v := range typ.EnumValuesDefinition {
					if _, ok := posible[v.EnumValue]; !ok {
						return fmt.Errorf("Unexpected enum value %q", v.EnumValue)
					}
				}
				return nil
			},
		},
		{
			name: "Extend incompatible type",
			sdl: `
			type Query {
				hello: String!
			}

			extend interface Query {
				name: String!
			}`,
			validateError: func(err error) error {
				msg := `trying to extend type "OBJECT" with type "INTERFACE"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend type already implements an interface",
			sdl: `
			interface Named {
				name: String!
			}
			type Product implements Named {
				id: ID!
				name: String!
			}
			extend type Product implements Named {
			}`,
			validateError: func(err error) error {
				msg := `interface "Named" implemented in the extension is already implemented in "Product"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend union already contains type",
			sdl: `
			type Named {
				name: String!
			}
			type Numbered {
				num: Int!
			}
			union Item = Named | Numbered
			type Coloured {
				Colour: String!
			}
			extend union Item = Coloured | Named
			`,
			validateError: func(err error) error {
				msg := `union type "Named" already declared in "Item"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend union contains type",
			sdl: `
			type Named {
				name: String!
			}
			type Numbered {
				num: Int!
			}
			union Item = Named | Numbered

			type Coloured {
				Colour: String!
			}
			
			extend union Item = Coloured
			`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Item"].(*types.Union)
				if !ok {
					return fmt.Errorf("type %q not found", "Item")
				}
				if len(typ.UnionMemberTypes) != 3 {
					return fmt.Errorf("Expected 3 possible types, but instead got %d types", len(typ.UnionMemberTypes))
				}
				posible := map[string]struct{}{
					"Coloured": struct{}{},
					"Named":    struct{}{},
					"Numbered": struct{}{},
				}
				for _, pt := range typ.UnionMemberTypes {
					if _, ok := posible[pt.Name]; !ok {
						return fmt.Errorf("Unexpected possible type %q", pt.Name)
					}
				}
				return nil
			},
		},
		{
			name: "Extend input",
			sdl: `
			input Product {
				id: ID!
				name: String!
			}
			extend input Product {
				category: Category!
				tags: [String!]! = ["sale", "shoes"]
			}
			input Category {
				id: ID!
				name: String!
			}
			`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Product"].(*types.InputObject)
				if !ok {
					return fmt.Errorf("type %q not found", "Product")
				}
				if len(typ.Values) != 4 {
					return fmt.Errorf("Expected 4 fields, but instead got %d types", len(typ.Values))
				}
				posible := map[string]struct{}{
					"id":       struct{}{},
					"name":     struct{}{},
					"category": struct{}{},
					"tags":     struct{}{},
				}
				for _, pt := range typ.Values {
					if _, ok := posible[pt.Name.Name]; !ok {
						return fmt.Errorf("Unexpected possible type %q", pt.Name)
					}
				}
				categoryField := typ.Values.Get("category")
				if categoryField == nil {
					return fmt.Errorf("field %q not found", "category")
				}
				if categoryField.Type.String() != "Category!" {
					return fmt.Errorf("expected type %q, but got %q", "Category!", categoryField.Type.String())
				}
				if categoryField.Type.Kind() != "NON_NULL" {
					return fmt.Errorf("expected kind %q, but got %q", "NON_NULL", categoryField.Type.Kind())
				}
				return nil
			},
		},
		{
			name: "Extend enum value already exists",
			sdl: `
			enum Currencies{
				AUD
				USD
				EUR
			}
			extend enum Currencies {
				AUD
			}`,
			validateError: func(err error) error {
				msg := `enum value "AUD" already declared in "Currencies"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend input field already exists",
			sdl: `
			input Product{
				name: String!
			}
			extend input Product {
				name: String!
			}`,
			validateError: func(err error) error {
				msg := `extended field {"name" {'\x06' '\x05'}} already exists`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend field already exists",
			sdl: `
			interface Named {
				name: String!
			}
			type Product implements Named {
				id: ID!
				name: String!
			}
			extend type Product {
				name: String!
			}`,
			validateError: func(err error) error {
				msg := `extended field "name" already exists`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend interface type",
			sdl: `
			interface Product {
				id: ID!
				name: String!
			}
			extend interface Product {
				category: String!
			}
			`,
			validateSchema: func(s *types.Schema) error {
				typ, ok := s.Types["Product"].(*types.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Product")
				}
				if len(typ.Fields) != 3 {
					return fmt.Errorf("Expected 3 fields, but instead got %d types", len(typ.Fields))
				}
				fields := map[string]struct{}{
					"id":       struct{}{},
					"name":     struct{}{},
					"category": struct{}{},
				}
				for _, f := range typ.Fields {
					if _, ok := fields[f.Name]; !ok {
						return fmt.Errorf("Unexpected field %q", f.Name)
					}
				}
				return nil
			},
		},
		{
			name: "Extend unknown type",
			sdl: `
			extend type User {
				name: String!
			}
			`,
			validateError: func(err error) error {
				msg := `trying to extend unknown type "User"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Extend invalid syntax",
			sdl: `
			extend invalid Node {
				id: ID!
			}
			`,
			validateError: func(err error) error {
				msg := `graphql: syntax error: unexpected "invalid", expecting "schema", "type", "enum", "interface", "union" or "input" (line 2, column 19)`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Parses directives",
			sdl: `
			directive @objectdirective on OBJECT
			directive @fielddirective on FIELD_DEFINITION
			directive @enumdirective on ENUM
			directive @uniondirective on UNION
			directive @directive on SCALAR
				| OBJECT
				| FIELD_DEFINITION
				| ARGUMENT_DEFINITION
				| INTERFACE
				| UNION
				| ENUM
				| ENUM_VALUE
				| INPUT_OBJECT
				| INPUT_FIELD_DEFINITION

			interface NamedEntity @directive { name: String }

			scalar Time @directive

			type Photo @objectdirective {
				id: ID! @deprecated @fielddirective
			}

			type Person implements NamedEntity @objectdirective {
				name: String
			}

			enum Direction @enumdirective {
				NORTH @deprecated
				EAST
				SOUTH
				WEST
			}

			union Union @uniondirective = Photo | Person
			`,
			validateSchema: func(s *types.Schema) error {
				namedEntityDirectives := s.Types["NamedEntity"].(*types.InterfaceTypeDefinition).Directives
				if len(namedEntityDirectives) != 1 || namedEntityDirectives[0].Name.Name != "directive" {
					return fmt.Errorf("missing directive on NamedEntity interface, expected @directive but got %v", namedEntityDirectives)
				}

				timeDirectives := s.Types["Time"].(*types.ScalarTypeDefinition).Directives
				if len(timeDirectives) != 1 || timeDirectives[0].Name.Name != "directive" {
					return fmt.Errorf("missing directive on Time scalar, expected @directive but got %v", timeDirectives)
				}

				photo := s.Types["Photo"].(*types.ObjectTypeDefinition)
				photoDirectives := photo.Directives
				if len(photoDirectives) != 1 || photoDirectives[0].Name.Name != "objectdirective" {
					return fmt.Errorf("missing directive on Time scalar, expected @objectdirective but got %v", photoDirectives)
				}
				if len(photo.Fields.Get("id").Directives) != 2 {
					return fmt.Errorf("expected Photo.id to have 2 directives but got %v", photoDirectives)
				}

				directionDirectives := s.Types["Direction"].(*types.EnumTypeDefinition).Directives
				if len(directionDirectives) != 1 || directionDirectives[0].Name.Name != "enumdirective" {
					return fmt.Errorf("missing directive on Direction enum, expected @enumdirective but got %v", directionDirectives)
				}

				unionDirectives := s.Types["Union"].(*types.Union).Directives
				if len(unionDirectives) != 1 || unionDirectives[0].Name.Name != "uniondirective" {
					return fmt.Errorf("missing directive on Union union, expected @uniondirective but got %v", unionDirectives)
				}
				return nil
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, err := schema.ParseSchema(test.sdl, test.useStringDescriptions)
			if err != nil {
				if test.validateError == nil {
					t.Fatal(err)
				}
				if err := test.validateError(err); err != nil {
					t.Fatal(err)
				}
			}
			if test.validateSchema != nil {
				if err := test.validateSchema(s); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
