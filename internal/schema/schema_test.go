package schema_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/ast"
	qerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/introspection"
)

func TestDescriptionBeforeExtension(t *testing.T) {
	extensions := []struct {
		name string
		base string
		ext  string
	}{
		{"schema", `type Query { x: String }`, `extend schema { query: Query }`},
		{"type", `type Query { x: String }`, `extend type Query { y: String }`},
		{"interface", `interface Node { x: String }`, `extend interface Node { y: String }`},
		{"union", `type A { x: String } type B { x: String } union U = A`, `extend union U = B`},
		{"enum", `enum E { A }`, `extend enum E { B }`},
		{"input", `input I { x: String }`, `extend input I { y: String }`},
		{"scalar", `scalar S`, `extend scalar S @specifiedBy(url: "https://example.com")`},
	}

	descriptions := []struct {
		name string
		text string
	}{
		{"string", `"description"`},
		{"empty string", `""`},
		{"block", "\"\"\"\nDescription\n\"\"\""},
		{"empty block", `""""""`},
		{"embedded syntax", `"secret \" extend type Query { admin: String }"`},
		{"untrusted text", `"` + strings.Repeat("x", 4096) + `"`},
	}

	for _, extension := range extensions {
		for _, description := range descriptions {
			for _, useStrings := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/stringDescriptions=%t", extension.name, description.name, useStrings), func(t *testing.T) {
					sdl := extension.base + "\n   " + description.text + "\n# comment\n" + extension.ext
					_, err := schema.ParseSchema(sdl, useStrings)
					var queryErr *qerrors.QueryError
					if !errors.As(err, &queryErr) || !errors.Is(err, qerrors.ErrSyntax) {
						t.Fatalf("expected syntax error, got %v", err)
					}
					if queryErr.Message != "syntax error: descriptions are not allowed on type system extensions" {
						t.Fatalf("unexpected error: %v", queryErr)
					}
					if len(queryErr.Locations) != 1 || queryErr.Locations[0] != (qerrors.Location{Line: 2, Column: 4}) {
						t.Fatalf("expected description location, got %v", queryErr.Locations)
					}
				})
			}
		}
		if extension.name == "scalar" {
			continue
		}
		for _, useStrings := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/comment/stringDescriptions=%t", extension.name, useStrings), func(t *testing.T) {
				if _, err := schema.ParseSchema(extension.base+"\n# description-like comment\n"+extension.ext, useStrings); err != nil {
					t.Fatalf("comment before extension should be accepted: %v", err)
				}
			})
		}
	}
}

func TestSchemaDefaultValues(t *testing.T) {
	tests := []struct {
		name string
		sdl  string
		want string
	}{
		{"interface argument", `interface Node { f(x: Int = "bad"): Int }`, `argument of Node.f "x"`},
		{"extended field argument", `type Query { f: Int } extend type Query { g(x: Boolean = 1): Int }`, `argument of Query.g "x"`},
		{"input field", `input I { x: [Int!] = [1, null] }`, `field of I "x"`},
		{"directive argument", `directive @d(x: E = B) on FIELD enum E { A }`, `argument of @d "x"`},
		{"int out of range", `type Query { f(x: Int = 2147483648): Int }`, `Int cannot represent`},
		{"wrong scalar shape", `type Query { f(x: String = [1]): Int }`, `expected value of type`},
		{"unknown input field", `input I { x: Int } type Query { f(x: I = {y: 1}): Int }`, `field "y" is not defined`},
		{"duplicate input field", `input I { x: Int } type Query { f(x: I = {x: 1, x: 2}): Int }`, `field "x" is provided more than once`},
		{"missing required input field", `input I { x: Int! } type Query { f(x: I = {}): Int }`, `field "I.x" of required type`},
		{"nested input field", `input I { x: Int! } type Query { f(x: [I] = [{x: "bad"}]): Int }`, `Int cannot represent`},
		{"oneof empty", `input I @oneOf { x: Int y: Int } type Query { f(x: I = {}): Int }`, `must specify exactly one field`},
		{"oneof multiple", `input I @oneOf { x: Int y: Int } type Query { f(x: I = {x: 1, y: 2}): Int }`, `must specify exactly one field`},
		{"oneof null", `input I @oneOf { x: Int } type Query { f(x: I = {x: null}): Int }`, `field "I.x" must be non-null`},
		{"valid coercions", `input I { x: Int! = 1 } type Query { f(a: [Int!] = 1, b: I = {}, c: String = null): Int }`, ""},
		{"valid oneof", `input I @oneOf { x: Int } type Query { f(x: I = {x: 1}): Int }`, ""},
		{"custom scalar object", `scalar JSON type Query { f(x: JSON = {a: [1, true]}): Int }`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestDescriptionsOnDefinitionsRemainSupported(t *testing.T) {
	sdl := `"type description"
		type Query {
			"field description"
			field("argument description" arg: String): String
		}`
	s, err := schema.ParseSchema(sdl, true)
	if err != nil {
		t.Fatal(err)
	}
	query := s.Types["Query"].(*ast.ObjectTypeDefinition)
	field := query.Fields.Get("field")
	arg := field.Arguments.Get("arg")
	if query.Desc != "type description" || field.Desc != "field description" || arg.Desc != "argument description" {
		t.Fatalf("unexpected descriptions: type=%q field=%q argument=%q", query.Desc, field.Desc, arg.Desc)
	}
}

func TestDirectiveArgumentValuesAreValidated(t *testing.T) {
	tests := []struct {
		name    string
		sdl     string
		wantErr string
	}{
		{
			name: "accepts valid enum value",
			sdl: `enum Size { SMALL MEDIUM }
				directive @tshirt(size: Size!) on OBJECT
				type Shirt @tshirt(size: MEDIUM) { color: String }
				type Query { shirt: Shirt }`,
		},
		{
			name: "rejects invalid enum value",
			sdl: `enum Size { SMALL MEDIUM }
				directive @tshirt(size: Size) on OBJECT
				type Shirt @tshirt(size: MEDUIM) { color: String }
				type Query { shirt: Shirt }`,
			wantErr: `invalid value MEDUIM for argument "size" of directive "tshirt"`,
		},
		{
			name: "rejects null for non-null argument",
			sdl: `directive @limit(value: Int!) on OBJECT
				type Item @limit(value: null) { name: String }
				type Query { item: Item }`,
			wantErr: `invalid value null for argument "value" of directive "limit"`,
		},
		{
			name: "rejects invalid scalar value",
			sdl: `directive @limit(value: Int) on OBJECT
				type Item @limit(value: "many") { name: String }
				type Query { item: Item }`,
			wantErr: `invalid value "many" for argument "value" of directive "limit"`,
		},
		{
			name: "validates list elements",
			sdl: `enum Size { SMALL MEDIUM }
				directive @sizes(values: [Size!]) on OBJECT
				type Item @sizes(values: [SMALL, MEDUIM]) { name: String }
				type Query { item: Item }`,
			wantErr: `invalid value [SMALL, MEDUIM] for argument "values" of directive "sizes"`,
		},
		{
			name: "validates input object required fields",
			sdl: `input Options { count: Int! }
				directive @options(value: Options) on OBJECT
				type Item @options(value: {}) { name: String }
				type Query { item: Item }`,
			wantErr: `invalid value {} for argument "value" of directive "options"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestIntrospectionDescriptionPresence(t *testing.T) {
	sdl := `
		"" type Query {
			"" field("" arg: String, missing: String, "text" text: String): String
			plain: String
		}
		extend type Query { "" extended: String }
		"" interface Node { "" name: String }
		"" union Search = Query
		"" enum Choice { "" EMPTY PLAIN "text" TEXT }
		"" input Input { "" empty: String plain: String "text" text: String }
		"" scalar Custom
		type Undescribed { value: String }
		"" directive @empty("" arg: String, missing: String) on FIELD
		directive @plain on FIELD
		# comment description
		type Commented { value: String }
	`
	for _, useStrings := range []bool{false, true} {
		t.Run(fmt.Sprintf("stringDescriptions=%t", useStrings), func(t *testing.T) {
			s, err := schema.ParseSchema(sdl, useStrings)
			if err != nil {
				t.Fatal(err)
			}
			check := func(label string, got *string, want *string) {
				t.Helper()
				if got == nil && want == nil {
					return
				}
				if got == nil || want == nil || *got != *want {
					format := func(value *string) string {
						if value == nil {
							return "null"
						}
						return fmt.Sprintf("%q", *value)
					}
					t.Errorf("%s: got %s, want %s", label, format(got), format(want))
				}
			}
			empty, text, comment := "", "text", "comment description"
			var wantEmpty, wantText, wantComment *string
			if useStrings {
				wantEmpty, wantText = &empty, &text
			} else {
				wantComment = &comment
			}
			for _, name := range []string{"Query", "Node", "Search", "Choice", "Input", "Custom"} {
				check(name, introspection.WrapType(s.Types[name]).Description(), wantEmpty)
			}
			check("Undescribed", introspection.WrapType(s.Types["Undescribed"]).Description(), nil)
			check("Commented", introspection.WrapType(s.Types["Commented"]).Description(), wantComment)

			fields := *introspection.WrapType(s.Types["Query"]).Fields(&struct{ IncludeDeprecated bool }{})
			check("field", fields[0].Description(), wantEmpty)
			check("plain field", fields[1].Description(), nil)
			check("extension field", fields[2].Description(), wantEmpty)
			args := fields[0].Args(&struct{ IncludeDeprecated bool }{})
			check("argument", args[0].Description(), wantEmpty)
			check("plain argument", args[1].Description(), nil)
			check("text argument", args[2].Description(), wantText)

			values := *introspection.WrapType(s.Types["Choice"]).EnumValues(&struct{ IncludeDeprecated bool }{})
			check("enum value", values[0].Description(), wantEmpty)
			check("plain enum value", values[1].Description(), nil)
			check("text enum value", values[2].Description(), wantText)
			inputs := *introspection.WrapType(s.Types["Input"]).InputFields(&struct{ IncludeDeprecated bool }{})
			check("input field", inputs[0].Description(), wantEmpty)
			check("plain input field", inputs[1].Description(), nil)
			check("text input field", inputs[2].Description(), wantText)

			for _, directive := range introspection.WrapSchema(s).Directives() {
				switch directive.Name() {
				case "empty":
					check("directive", directive.Description(), wantEmpty)
					args := directive.Args(&struct{ IncludeDeprecated bool }{})
					check("directive argument", args[0].Description(), wantEmpty)
					check("plain directive argument", args[1].Description(), nil)
				case "plain":
					check("plain directive", directive.Description(), nil)
				}
			}
			check("list", introspection.WrapType(&ast.List{OfType: s.Types["Query"]}).Description(), nil)
		})
	}
}

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name                  string
		sdl                   string
		useStringDescriptions bool
		validateError         func(err error) error
		validateSchema        func(s *ast.Schema) error
	}{
		{
			name: "Parses interface definition",
			sdl:  "interface Greeting { message: String! }",
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Greeting"
				typ, ok := s.Types[typeName].(*ast.InterfaceTypeDefinition)
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
				id: ID!
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*ast.ObjectTypeDefinition)
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
			name: "Parses type invalid syntax",
			sdl: `
			type U = T
			`,
			validateError: func(err error) error {
				msg := `graphql: syntax error: unexpected "=", expecting "{" (line 2, column 11)`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
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
			validateSchema: func(s *ast.Schema) error {
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
			validateSchema: func(s *ast.Schema) error {
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
			validateSchema: func(s *ast.Schema) error {
				typq, ok := s.Types["Query"].(*ast.ObjectTypeDefinition)
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

				typm, ok := s.Types["Mutation"].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Query"].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				typq, ok := s.Types["Query"].(*ast.ObjectTypeDefinition)
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

				typm, ok := s.Types["Mutation"].(*ast.ObjectTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Product"].(*ast.ObjectTypeDefinition)
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

				ifc, ok := s.Types["Named"].(*ast.InterfaceTypeDefinition)
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Item"].(*ast.Union)
				if !ok {
					return fmt.Errorf("type %q not found", "Item")
				}
				if len(typ.UnionMemberTypes) != 3 {
					return fmt.Errorf("Expected 3 possible types, but instead got %d types", len(typ.UnionMemberTypes))
				}
				posible := map[string]struct{}{
					"Coloured": {},
					"Named":    {},
					"Numbered": {},
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Currencies"].(*ast.EnumTypeDefinition)
				if !ok {
					return fmt.Errorf("enum %q not found", "Currencies")
				}
				if len(typ.EnumValuesDefinition) != 5 {
					return fmt.Errorf("Expected 5 enum values, but instead got %d types", len(typ.EnumValuesDefinition))
				}
				posible := map[string]struct{}{
					"AUD": {},
					"USD": {},
					"EUR": {},
					"BGN": {},
					"GBP": {},
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Item"].(*ast.Union)
				if !ok {
					return fmt.Errorf("type %q not found", "Item")
				}
				if len(typ.UnionMemberTypes) != 3 {
					return fmt.Errorf("Expected 3 possible types, but instead got %d types", len(typ.UnionMemberTypes))
				}
				posible := map[string]struct{}{
					"Coloured": {},
					"Named":    {},
					"Numbered": {},
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Product"].(*ast.InputObject)
				if !ok {
					return fmt.Errorf("type %q not found", "Product")
				}
				if len(typ.Values) != 4 {
					return fmt.Errorf("Expected 4 fields, but instead got %d types", len(typ.Values))
				}
				posible := map[string]struct{}{
					"id":       {},
					"name":     {},
					"category": {},
					"tags":     {},
				}
				for _, pt := range typ.Values {
					if _, ok := posible[pt.Name.Name]; !ok {
						return fmt.Errorf("Unexpected possible type %q", pt.Name.Name)
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
				msg := `extended field "name" already exists`
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
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["Product"].(*ast.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("type %q not found", "Product")
				}
				if len(typ.Fields) != 3 {
					return fmt.Errorf("Expected 3 fields, but instead got %d types", len(typ.Fields))
				}
				fields := map[string]struct{}{
					"id":       {},
					"name":     {},
					"category": {},
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
			directive @repeatabledirective repeatable on SCALAR

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

			scalar Mass @repeatabledirective @repeatabledirective
			`,
			validateSchema: func(s *ast.Schema) error {
				namedEntityDirectives := s.Types["NamedEntity"].(*ast.InterfaceTypeDefinition).Directives
				if len(namedEntityDirectives) != 1 || namedEntityDirectives[0].Name.Name != "directive" {
					return fmt.Errorf("missing directive on NamedEntity interface, expected @directive but got %v", namedEntityDirectives)
				}

				timeDirectives := s.Types["Time"].(*ast.ScalarTypeDefinition).Directives
				if len(timeDirectives) != 1 || timeDirectives[0].Name.Name != "directive" {
					return fmt.Errorf("missing directive on Time scalar, expected @directive but got %v", timeDirectives)
				}

				photo := s.Types["Photo"].(*ast.ObjectTypeDefinition)
				photoDirectives := photo.Directives
				if len(photoDirectives) != 1 || photoDirectives[0].Name.Name != "objectdirective" {
					return fmt.Errorf("missing directive on Time scalar, expected @objectdirective but got %v", photoDirectives)
				}
				if len(photo.Fields.Get("id").Directives) != 2 {
					return fmt.Errorf("expected Photo.id to have 2 directives but got %v", photoDirectives)
				}

				directionDirectives := s.Types["Direction"].(*ast.EnumTypeDefinition).Directives
				if len(directionDirectives) != 1 || directionDirectives[0].Name.Name != "enumdirective" {
					return fmt.Errorf("missing directive on Direction enum, expected @enumdirective but got %v", directionDirectives)
				}

				unionDirectives := s.Types["Union"].(*ast.Union).Directives
				if len(unionDirectives) != 1 || unionDirectives[0].Name.Name != "uniondirective" {
					return fmt.Errorf("missing directive on Union union, expected @uniondirective but got %v", unionDirectives)
				}

				massDirectives := s.Types["Mass"].(*ast.ScalarTypeDefinition).Directives
				if len(massDirectives) != 2 || massDirectives[0].Name.Name != "repeatabledirective" || massDirectives[1].Name.Name != "repeatabledirective" {
					return fmt.Errorf("missing directive on Repeatable scalar, expected @repeatabledirective @repeatabledirective but got %v", massDirectives)
				}
				return nil
			},
		},
		{
			name: "Sets Directive.Repeatable if `repeatable` keyword is given",
			sdl: `
			directive @nonrepeatabledirective on SCALAR
			directive @repeatabledirective repeatable on SCALAR
			`,
			validateSchema: func(s *ast.Schema) error {
				if dir := s.Directives["nonrepeatabledirective"]; dir.Repeatable {
					return fmt.Errorf("did not expect directive to be repeatable: %v", dir)
				}
				if dir := s.Directives["repeatabledirective"]; !dir.Repeatable {
					return fmt.Errorf("expected directive to be repeatable: %v", dir)
				}
				return nil
			},
		},
		{
			name: "Directive definition does not allow double-`repeatable`",
			sdl: `
			directive @mydirective repeatable repeatable SCALAR
			scalar MyScalar @mydirective
			`,
			validateError: func(err error) error {
				msg := `graphql: syntax error: unexpected "repeatable", expecting "on" (line 2, column 38)`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Directive definition does not allow double-`on` instead of `repeatable on`",
			sdl: `
			directive @mydirective on on SCALAR
			scalar MyScalar @mydirective
			`,
			validateError: func(err error) error {
				prefix := `graphql: syntax error: "on" is not a legal directive location`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
				}
				return nil
			},
		},
		{
			name: "Disallow repeat of a directive if it is not `repeatable`",
			sdl: `
			directive @nonrepeatabledirective on FIELD_DEFINITION
			type Foo {
				bar: String @nonrepeatabledirective @nonrepeatabledirective
			}
			`,
			validateError: func(err error) error {
				prefix := `graphql: non repeatable directive "nonrepeatabledirective" can not be repeated. Consider adding "repeatable"`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
				}
				return nil
			},
		},
		{
			name: "Decorating scalar with an undeclared directive should return an error",
			sdl: `
			scalar S @undeclareddirective
			`,
			validateError: func(err error) error {
				prefix := `graphql: directive "undeclareddirective" not found`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
				}
				return nil
			},
		},
		{
			name: "Decorating argument with an undeclared directive should return an error",
			sdl: `
			type Query {
				hello(name: String! @undeclareddirective): String!
			}
			`,
			validateError: func(err error) error {
				prefix := `graphql: directive "undeclareddirective" not found`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
				}
				return nil
			},
		},
		{
			name: "Decorating input object with an undeclared directive should return an error",
			sdl: `
			input InputObject @undeclareddirective{field: String!}
			`,
			validateError: func(err error) error {
				prefix := `graphql: directive "undeclareddirective" not found`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
				}
				return nil
			},
		},
		{
			name: "Decorating interface with an undeclared directive should return an error",
			sdl: `
			interface I @undeclareddirective {field: String!}
			`,
			validateError: func(err error) error {
				prefix := `graphql: directive "undeclareddirective" not found`
				if err == nil || !strings.HasPrefix(err.Error(), prefix) {
					return fmt.Errorf("expected error starting with %q, but got %q", prefix, err)
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
				if err2 := test.validateError(err); err2 != nil {
					t.Fatal(err2)
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

func TestInterfaceImplementsInterface(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		sdl                   string
		useStringDescriptions bool
		validateError         func(err error) error
		validateSchema        func(s *ast.Schema) error
	}{
		{
			name: "Parses interface implementing other interface",
			sdl: `
			interface Foo {
				field: String!
			}
			interface Bar implements Foo {
				field: String!
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				const implementedInterfaceName = "Bar"
				typ, ok := s.Types[implementedInterfaceName].(*ast.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("interface %q not found", implementedInterfaceName)
				}
				if len(typ.Fields) != 1 {
					return fmt.Errorf("invalid number of fields: want %d, have %d", 1, len(typ.Fields))
				}
				const fieldName = "field"

				if typ.Fields[0].Name != fieldName {
					return fmt.Errorf("field %q not found", fieldName)
				}

				if len(typ.Interfaces) != 1 {
					return fmt.Errorf("invalid number of implementing interfaces found on %q: want %d, have %d", implementedInterfaceName, 1, len(typ.Interfaces))
				}

				const implementingInterfaceName = "Foo"
				if typ.Interfaces[0].Name != implementingInterfaceName {
					return fmt.Errorf("interface %q not found", implementingInterfaceName)
				}

				return nil
			},
		},
		{
			name: "Parses interface transitively implementing an interface that implements an interface",
			sdl: `
			interface Foo {
				field: String!
			}
			interface Bar implements Foo {
				field: String!
			}
			interface Baz implements Bar & Foo {
				field: String!
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				const implementedInterfaceName = "Baz"
				typ, ok := s.Types[implementedInterfaceName].(*ast.InterfaceTypeDefinition)
				if !ok {
					return fmt.Errorf("interface %q not found", implementedInterfaceName)
				}
				if len(typ.Fields) != 1 {
					return fmt.Errorf("invalid number of fields: want %d, have %d", 1, len(typ.Fields))
				}
				const fieldName = "field"

				if typ.Fields[0].Name != fieldName {
					return fmt.Errorf("field %q not found", fieldName)
				}

				if len(typ.Interfaces) != 2 {
					return fmt.Errorf("invalid number of implementing interfaces found on %q: want %d, have %d", implementedInterfaceName, 2, len(typ.Interfaces))
				}

				const firstImplementingInterfaceName = "Bar"
				if typ.Interfaces[0].Name != firstImplementingInterfaceName {
					return fmt.Errorf("first interface %q not found", firstImplementingInterfaceName)
				}

				const secondImplementingInterfaceName = "Foo"
				if typ.Interfaces[1].Name != secondImplementingInterfaceName {
					return fmt.Errorf("second interface %q not found", secondImplementingInterfaceName)
				}

				return nil
			},
		},
		{
			name: "Transitively implemented interfaces must also be defined on an implementing type or interface",
			sdl: `
			interface A {
				message: String!
			}
			interface B implements A {
				message: String!
				name: String!
			}
			interface C implements B {
				message: String!
				name: String!
				hug: Boolean!
			}
			`,
			validateError: func(err error) error {
				msg := `graphql: interface "C" must explicitly implement transitive interface "A"`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Unions can be defined with a leading pipe",
			sdl: `
			type Named {
				name: String!
			}
			type Numbered {
				num: Int!
			}
			union Item1 =
				| Named
				| Numbered
			union Item2 = | Named | Numbered
			`,
			validateSchema: func(s *ast.Schema) error {
				for _, itemName := range []string{"Item1", "Item2"} {
					typ, ok := s.Types[itemName].(*ast.Union)
					if !ok {
						return fmt.Errorf("type %q not found", "Item")
					}
					if len(typ.UnionMemberTypes) != 2 {
						return fmt.Errorf("Expected 2 possible types, but instead got %d types", len(typ.UnionMemberTypes))
					}
					posible := map[string]struct{}{
						"Named":    {},
						"Numbered": {},
					}
					for _, pt := range typ.UnionMemberTypes {
						if _, ok := posible[pt.Name]; !ok {
							return fmt.Errorf("Unexpected possible type %q", pt.Name)
						}
					}
				}
				return nil
			},
		},
		{
			name: "Accepts object implementing interface with additional optional argument",
			sdl: `
			interface Node {
				name(format: String!): String!
			}
			type User implements Node {
				name(format: String!, locale: String): String!
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				return nil
			},
		},
		{
			name: "Rejects object implementing interface missing required interface argument",
			sdl: `
			interface Node {
				name(format: String!): String!
			}
			type User implements Node {
				name: String!
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `expects argument "format"`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Rejects object implementing interface with incompatible argument type",
			sdl: `
			interface Node {
				name(format: String!): String!
			}
			type User implements Node {
				name(format: Int!): String!
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `argument "format" has type "String!"`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				if want, have := `defines type "Int!"`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Rejects object implementing interface with additional required argument",
			sdl: `
			interface Node {
				name(format: String!): String!
			}
			type User implements Node {
				name(format: String!, locale: String!): String!
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `additional argument "locale"`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				if want, have := `must not be required`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Rejects interface implementing interface with additional required argument",
			sdl: `
			interface Node {
				name(format: String!): String!
			}
			interface Entity implements Node {
				name(format: String!, locale: String!): String!
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `implementing interface "Entity"`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				if want, have := `must not be required`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Parses valid OneOf input type with nullable fields",
			sdl: `
			input FindUserInput @oneOf {
				id: ID
				email: String
			}
			type Query {
				findUser(by: FindUserInput): String
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["FindUserInput"].(*ast.InputObject)
				if !ok {
					return fmt.Errorf("type %q not found", "FindUserInput")
				}
				if len(typ.Values) != 2 {
					return fmt.Errorf("expected 2 fields, got %d", len(typ.Values))
				}
				if typ.Directives.Get("oneOf") == nil {
					return fmt.Errorf("expected @oneOf directive on FindUserInput")
				}
				return nil
			},
		},
		{
			name: "Rejects OneOf input type with non-nullable field",
			sdl: `
			input FindUserInput @oneOf {
				id: ID!
				email: String
			}
			type Query {
				findUser(by: FindUserInput): String
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `OneOf input field FindUserInput.id must be nullable`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Rejects OneOf input type with field default value",
			sdl: `
			input FindUserInput @oneOf {
				id: ID
				priority: Int = 1
			}
			type Query {
				findUser(by: FindUserInput): String
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if want, have := `OneOf input field FindUserInput.priority cannot have a default value`, err.Error(); !strings.Contains(have, want) {
					return fmt.Errorf("unexpected error: want %q to contain %q", have, want)
				}
				return nil
			},
		},
		{
			name: "Rejects @oneOf directive on non-input type",
			sdl: `
			type Query @oneOf {
				field: String
			}
			`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("want error, have <nil>")
				}
				if strings.Contains(err.Error(), "@oneOf") && strings.Contains(err.Error(), "INPUT_OBJECT") {
					return nil
				}
				return fmt.Errorf("unexpected error: want error about @oneOf only on INPUT_OBJECT, have %q", err.Error())
			},
		},
		{
			name: "Accepts object implementing interface where both field and interface field are deprecated",
			sdl: `
			interface Node {
				id: ID! @deprecated(reason: "use newId")
			}
			type User implements Node {
				id: ID! @deprecated(reason: "use newId")
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				return nil
			},
		},
		{
			name: "Accepts object implementing interface where interface field is deprecated and implementing field is not",
			sdl: `
			interface Node {
				id: ID! @deprecated(reason: "old")
			}
			type User implements Node {
				id: ID!
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				return nil
			},
		},
		{
			name: "Rejects object implementing interface where interface field is not deprecated but implementing field is",
			sdl: `
			interface Node {
				id: ID!
			}
			type User implements Node {
				id: ID! @deprecated(reason: "use newId")
			}
			`,
			validateError: func(err error) error {
				msg := `graphql: interface "Node" field "id" is not deprecated but implementing type "User" marks it as deprecated`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
		{
			name: "Rejects interface implementing interface where interface field is not deprecated but implementing interface field is",
			sdl: `
			interface Node {
				id: ID!
			}
			interface Entity implements Node {
				id: ID! @deprecated(reason: "use newId")
			}
			`,
			validateError: func(err error) error {
				msg := `graphql: interface "Node" field "id" is not deprecated but implementing interface "Entity" marks it as deprecated`
				if err == nil || err.Error() != msg {
					return fmt.Errorf("expected error %q, but got %q", msg, err)
				}
				return nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.ParseSchema(tt.sdl, tt.useStringDescriptions)
			if err != nil {
				if tt.validateError == nil {
					t.Fatal(err)
				}
				if err2 := tt.validateError(err); err2 != nil {
					t.Fatal(err2)
				}
			}
			if tt.validateSchema != nil {
				if err2 := tt.validateSchema(s); err2 != nil {
					t.Fatal(err2)
				}
			}
		})
	}
}

func TestSchemaValidatesInputOutputTypes(t *testing.T) {
	tests := []struct {
		name string
		sdl  string
	}{
		{
			name: "Object type used as input object field",
			sdl: `
				type Query {
					field(arg: SomeInputObject): String
				}

				type SomeObject {
					field: String
				}

				input SomeInputObject {
					badField: SomeObject
				}
			`,
		},
		{
			name: "Object type used as field argument",
			sdl: `
				type Query {
					hello(input: HelloInput): String!
				}
				type HelloInput {
					name: String
				}
			`,
		},
		{
			name: "Interface type used as input object field",
			sdl: `
				type Query {
					field(arg: SomeInputObject): String
				}

				interface SomeInterface {
					field: String
				}

				input SomeInputObject {
					badField: SomeInterface
				}
			`,
		},
		{
			name: "Union type used as input object field",
			sdl: `
				type Query {
					field(arg: SomeInputObject): String
				}

				type TypeA {
					field: String
				}
				type TypeB {
					field: String
				}

				union SomeUnion = TypeA | TypeB

				input SomeInputObject {
					badField: SomeUnion
				}
			`,
		},
		{
			name: "Input object used as output field type",
			sdl: `
				type Query {
					field: SomeInputObject
				}

				input SomeInputObject {
					name: String
				}
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if err == nil {
				t.Fatal("expected error for invalid type usage, but schema was accepted without error")
			}
		})
	}
}

func TestSchemaValidatesWrappedAndExtendedTypes(t *testing.T) {
	tests := []struct {
		name, sdl, want string
	}{
		{
			name: "wrapped input field",
			sdl:  `type Query { field: String } type Object { field: String } input Input { field: [[Object!]!]! }`,
			want: `input value "field": type "[[Object!]!]!" is not an input type`,
		},
		{
			name: "interface output field",
			sdl:  `input Input { field: String } interface Node { field: [Input!]! }`,
			want: `field "field": type "[Input!]!" is not an output type`,
		},
		{
			name: "extended field argument",
			sdl:  `type Query { field: String } type Object { field: String } extend type Query { added(arg: [Object!]): String }`,
			want: `input value "arg": type "[Object!]" is not an input type`,
		},
		{
			name: "directive argument",
			sdl:  `type Object { field: String } directive @test(arg: [Object!]!) on FIELD_DEFINITION`,
			want: `directive "test" argument "arg": type "[Object!]!" is not an input type`,
		},
		{
			name: "valid input and output types",
			sdl: `
				scalar Custom
				enum Choice { A B }
				input Input { scalar: Custom, choice: [Choice!], nested: Input }
				interface Node { field(arg: [Input!]): [Choice!]! }
				type Query { field(arg: [Input!]): [Choice!]!, object: Object, node: Node, result: Result }
				type Object { field: Custom }
				union Result = Object
				directive @test(arg: [Input!]) on FIELD_DEFINITION
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}

			var qerr *qerrors.QueryError
			if !errors.As(err, &qerr) || qerr.Message != tt.want {
				t.Fatalf("expected GraphQL error message %q, got %v", tt.want, err)
			}
		})
	}
}

func TestRejectsDuplicateInterfaceImplementation(t *testing.T) {
	tests := []struct {
		name          string
		sdl           string
		validateError func(err error) error
	}{
		{
			name: "Object declares same interface twice",
			sdl: `
			interface NonUniqInterface {
				f: String
			}
			interface AnotherInterface {
				f: String
			}
			type SomeObject implements NonUniqInterface & AnotherInterface & NonUniqInterface {
				f: String
			}`,
			validateError: func(err error) error {
				if err == nil {
					return fmt.Errorf("expected error for duplicate interface implementation, but got <nil>")
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if err2 := tt.validateError(err); err2 != nil {
				t.Fatal(err2)
			}
		})
	}
}

func TestNoDuplicatePossibleTypesWhenInterfaceRepeated(t *testing.T) {
	sdl := `
		interface Node {
			id: ID!
		}
		type User implements Node & Node {
			id: ID!
			name: String!
		}
	`
	s, err := schema.ParseSchema(sdl, false)
	if err != nil {
		// If the parser rejects duplicate interface names, that is acceptable
		// correct behavior — no duplicate PossibleTypes can result.
		return
	}

	nodeType, ok := s.Types["Node"].(*ast.InterfaceTypeDefinition)
	if !ok {
		t.Fatal("interface Node not found")
	}

	if len(nodeType.PossibleTypes) != 1 {
		t.Fatalf("expected 1 possible type for Node, got %d", len(nodeType.PossibleTypes))
	}
	if nodeType.PossibleTypes[0].Name != "User" {
		t.Errorf("expected possible type to be User, got %s", nodeType.PossibleTypes[0].Name)
	}
}

func TestExtendNonRepeatableDirective(t *testing.T) {
	tests := []struct {
		name string
		sdl  string
	}{
		{
			name: "extend type repeats non-repeatable directive",
			sdl: `
			directive @foo on OBJECT
			type Query {
				someField: String
			}
			type TestObject @foo {
				field: String
			}
			extend type TestObject @foo {
				newField: String
			}
			`,
		},
		{
			name: "extend interface repeats non-repeatable directive",
			sdl: `
			directive @foo on INTERFACE
			type Query {
				someField: String
			}
			interface TestInterface @foo {
				field: String
			}
			extend interface TestInterface @foo {
				newField: String
			}
			`,
		},
		{
			name: "extend union repeats non-repeatable directive",
			sdl: `
			directive @foo on UNION
			type Query {
				someField: String
			}
			type A { field: String }
			type B { field: String }
			union TestUnion @foo = A
			extend union TestUnion @foo = B
			`,
		},
		{
			name: "extend enum repeats non-repeatable directive",
			sdl: `
			directive @foo on ENUM
			type Query {
				someField: String
			}
			enum TestEnum @foo {
				A
			}
			extend enum TestEnum @foo {
				B
			}
			`,
		},
		{
			name: "extend input repeats non-repeatable directive",
			sdl: `
			directive @foo on INPUT_OBJECT
			type Query {
				someField: String
			}
			input TestInput @foo {
				field: String
			}
			extend input TestInput @foo {
				newField: String
			}
			`,
		},
		{
			name: "extend schema repeats non-repeatable directive",
			sdl: `
			directive @foo on SCHEMA
			type Query {
				someField: String
			}
			schema @foo {
				query: Query
			}
			extend schema @foo
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schema.ParseSchema(tt.sdl, false)
			if err == nil {
				t.Fatal("expected error for non-repeatable directive repeated via extend, but got <nil>")
			}
			if !strings.Contains(err.Error(), "non repeatable") && !strings.Contains(err.Error(), "can only be used once") {
				t.Fatalf("expected non-repeatable directive error, but got: %q", err.Error())
			}
		})
	}
}

func TestExtendInputObjectDirectivesAreMerged(t *testing.T) {
	tests := []struct {
		name           string
		sdl            string
		validateError  func(err error) error
		validateSchema func(s *ast.Schema) error
	}{
		{
			name: "Extend input object type with directive",
			sdl: `
			directive @oneOf on INPUT_OBJECT

			input SomeInput {
				first: String
			}

			extend input SomeInput @oneOf {
				second: Int
			}
			`,
			validateSchema: func(s *ast.Schema) error {
				typ, ok := s.Types["SomeInput"].(*ast.InputObject)
				if !ok {
					return fmt.Errorf("type %q not found", "SomeInput")
				}

				// Fields from the extension should be merged
				if len(typ.Values) != 2 {
					return fmt.Errorf("expected 2 fields, but got %d", len(typ.Values))
				}
				if typ.Values.Get("first") == nil {
					return fmt.Errorf("field %q not found", "first")
				}
				if typ.Values.Get("second") == nil {
					return fmt.Errorf("field %q not found", "second")
				}

				// Directives from the extension should also be merged
				d := typ.Directives.Get("oneOf")
				if d == nil {
					return fmt.Errorf("expected @oneOf directive on SomeInput after extension, but Directives.Get(\"oneOf\") returned nil")
				}

				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.ParseSchema(tt.sdl, false)
			if err != nil {
				if tt.validateError == nil {
					t.Fatal(err)
				}
				if err2 := tt.validateError(err); err2 != nil {
					t.Fatal(err2)
				}
			}
			if tt.validateSchema != nil {
				if err := tt.validateSchema(s); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestDuplicateSchemaNames(t *testing.T) {
	syntaxError := func(message string) error {
		return &qerrors.QueryError{Err: qerrors.ErrSyntax, Message: "syntax error: " + message}
	}
	tests := []struct {
		name string
		sdl  string
		want error
	}{
		{"objects", `type Foo { id: Int } type Foo { id: String }`, syntaxError(`type "Foo" already declared`)},
		{"different type kinds", `type Foo { id: Int } scalar Foo`, syntaxError(`type "Foo" already declared`)},
		{"directives", `directive @foo on FIELD directive @foo on FIELD`, syntaxError(`directive "@foo" already declared`)},
		{"interfaces", `interface Node { id: ID } interface Node { id: ID }`, syntaxError(`type "Node" already declared`)},
		{"unions", `type A { x: Int } union U = A union U = A`, syntaxError(`type "U" already declared`)},
		{"enums", `enum E { A } enum E { B }`, syntaxError(`type "E" already declared`)},
		{"inputs", `input I { x: Int } input I { y: Int }`, syntaxError(`type "I" already declared`)},
		{"scalars", `scalar S scalar S`, syntaxError(`type "S" already declared`)},
		{"built-in scalar twice", `scalar String scalar String`, syntaxError(`type "String" already declared`)},
		{"built-in directive twice", `directive @skip on FIELD directive @skip on FIELD`, syntaxError(`directive "@skip" already declared`)},
		{"interface implements twice", `interface A { x: Int } interface B implements A & A { x: Int }`, qerrors.Errorf(`interface "B" implements interface "A" more than once`)},
		{"interface extension repeats implementation", `interface A { x: Int } interface B implements A { x: Int } extend interface B implements A { y: Int }`, qerrors.Errorf(`interface "B" implements interface "A" more than once`)},
		{"interface extensions repeat implementation", `interface A { x: Int } interface B { x: Int } extend interface B implements A { y: Int } extend interface B implements A { z: Int }`, qerrors.Errorf(`interface "B" implements interface "A" more than once`)},
		{"interface extension validates implementation", `interface A { x: Int } interface B { y: Int } extend interface B implements A { z: Int }`, qerrors.Errorf(`interface "A" expects field "x" but "B" does not provide it`)},
		{"object extension repeats implementation", `interface A { x: Int } type B { x: Int } extend type B implements A & A { y: Int }`, errors.New(`interface "A" implemented in the extension is already implemented in "B"`)},
		{"object extensions repeat implementation", `interface A { x: Int } type B { x: Int } extend type B implements A { y: Int } extend type B implements A { z: Int }`, errors.New(`interface "A" implemented in the extension is already implemented in "B"`)},
		{"union member twice", `type A { x: Int } union U = A | A`, qerrors.Errorf(`union type "A" already declared in "U"`)},
		{"union extension repeats member", `type A { x: Int } type B { x: Int } union U = A extend union U = B | B`, errors.New(`union type "B" already declared in "U"`)},
		{"union extensions repeat member", `type A { x: Int } type B { x: Int } union U = A extend union U = B extend union U = B`, errors.New(`union type "B" already declared in "U"`)},
		{"built-ins declared once", `scalar String directive @skip(if: Boolean!) on FIELD type Query { x: String }`, nil},
		{"separate namespaces", `scalar A directive @A on FIELD`, nil},
		{"valid type extension", `type A { x: Int } extend type A { y: Int }`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseSchema(t, tt.sdl, tt.want)
		})
	}
	t.Run("valid interface extension", func(t *testing.T) {
		s := parseSchema(t, `interface A { x: Int } interface B { x: Int } extend interface B implements A { y: Int }`, nil)
		iface := s.Types["B"].(*ast.InterfaceTypeDefinition)
		if len(iface.Interfaces) != 1 || iface.Interfaces[0].Name != "A" {
			t.Fatalf("expected interface B to implement A, got %v", iface.Interfaces)
		}
	})
}

func TestInputObjectCircularReferences(t *testing.T) {
	tests := []struct {
		name string
		sdl  string
		want error
	}{
		{"breakable references", `type Query { field(arg: A): String } input A { self: A array: [A] nonNullArray: [A]! nonNullElements: [A!]! other: B } input B { parent: A }`, nil},
		{"self reference", `type Query { field(arg: A): String } input A { self: A! }`, qerrors.Errorf(`Cannot reference Input Object "A" within itself through a series of non-null fields: "self"`)},
		{"three object cycle", `type Query { field(arg: A): String } input A { toB: B! } input B { toC: C! } input C { toA: A! }`, qerrors.Errorf(`Cannot reference Input Object "A" within itself through a series of non-null fields: "toB.toC.toA"`)},
		{"multiple cycles", `type Query { field(arg: A): String } input A { toB: B! } input B { toA: A! toC: C! } input C { toB: B! self: C! }`, qerrors.Errorf(`Cannot reference Input Object "A" within itself through a series of non-null fields: "toB.toA"`)},
		{"cycle excludes leading path", `input A { toB: B! } input B { toC: C! } input C { toB: B! }`, qerrors.Errorf(`Cannot reference Input Object "B" within itself through a series of non-null fields: "toC.toB"`)},
		{"extension creates unused cycle", `type Query { field: String } input A { value: String } extend input A { toB: B! } input B { toA: A! }`, qerrors.Errorf(`Cannot reference Input Object "A" within itself through a series of non-null fields: "toB.toA"`)},
		{"nullable edge breaks cycle", `input A { toB: B! } input B { toA: A }`, nil},
		{"list edge breaks cycle", `input A { toB: B! } input B { toA: [A!]! }`, nil},
		{"shared dependency is not a cycle", `input A { first: B! second: C! } input B { toC: C! } input C { value: String! }`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseSchema(t, tt.sdl, tt.want)
		})
	}
}

func parseSchema(t *testing.T, sdl string, want error) *ast.Schema {
	t.Helper()
	s, err := schema.ParseSchema(sdl, false)
	if reflect.TypeOf(err) != reflect.TypeOf(want) {
		t.Fatalf("expected error type %T, got %T (%v)", want, err, err)
	}
	if want == nil {
		return s
	}
	if werr, ok := errors.AsType[*qerrors.QueryError](want); ok {
		got := err.(*qerrors.QueryError)
		if got.Message != werr.Message || errors.Is(err, qerrors.ErrSyntax) != errors.Is(want, qerrors.ErrSyntax) {
			t.Fatalf("expected error %q (syntax: %t), got %q (syntax: %t)", werr.Message, errors.Is(want, qerrors.ErrSyntax), got.Message, errors.Is(err, qerrors.ErrSyntax))
		}
	} else if err.Error() != want.Error() {
		t.Fatalf("expected error %q, got %q", want.Error(), err.Error())
	}
	return nil
}

func TestVariableNotAllowedInSchemaDirectiveArgument(t *testing.T) {
	varErr := &qerrors.QueryError{Err: qerrors.ErrSyntax, Message: "syntax error: variable not allowed"}
	tests := []struct {
		name string
		sdl  string
		want error
	}{
		{"variable in directive argument on type definition", `directive @test(arg: String) on OBJECT type Query @test(arg: $var) { hello: String! }`, varErr},
		{"variable in directive argument on field definition", `directive @test(arg: String) on FIELD_DEFINITION type Query { hello: String! @test(arg: $var) }`, varErr},
		{"variable nested in list in directive argument on type definition", `directive @test(arg: [String]) on OBJECT type Query @test(arg: [$var]) { hello: String! }`, varErr},
		{"variable nested in input object", `input Options { value: String } directive @test(arg: Options) on OBJECT type Query @test(arg: {value: $var}) { hello: String }`, varErr},
		{"variable on argument definition", `directive @test(arg: String) on ARGUMENT_DEFINITION type Query { hello(name: String @test(arg: $var)): String }`, varErr},
		{"variable on schema extension", `directive @test(arg: String) on SCHEMA type Query { hello: String } extend schema @test(arg: $var)`, varErr},
		{"constant nested values are allowed", `input Options { values: [String] } directive @test(arg: Options) on OBJECT type Query @test(arg: {values: ["literal"]}) { hello: String }`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseSchema(t, tt.sdl, tt.want)
		})
	}
}
