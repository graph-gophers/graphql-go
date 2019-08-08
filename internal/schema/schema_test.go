package schema_test

import (
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go/internal/schema"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name                  string
		sdl                   string
		useStringDescriptions bool
		validateError         func(err error) error
		validateSchema        func(s *schema.Schema) error
	}{
		{
			name: "Parses interface definition",
			sdl:  "interface Greeting { message: String! }",
			validateSchema: func(s *schema.Schema) error {
				const typeName = "Greeting"
				typ, ok := s.Types[typeName].(*schema.Interface)
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
			sdl:  `
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
			validateSchema: func(s *schema.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*schema.Object)
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
			name: "Parses type with multi-line description string",
			sdl: `
			"""
			Multi-line description.
			"""
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *schema.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*schema.Object)
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
			name: "Parses type with multi-line description and ignores comments",
			sdl: `
			"""
			Multi-line description with ignored comments.
			"""
			# This comment should be ignored.
			type Type {
				field: String
			}`,
			useStringDescriptions: true,
			validateSchema: func(s *schema.Schema) error {
				const typeName = "Type"
				typ, ok := s.Types[typeName].(*schema.Object)
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
			validateSchema: func(s *schema.Schema) error {
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
			validateSchema: func(s *schema.Schema) error {
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
	} {
		t.Run(test.name, func(t *testing.T) {
			s := schema.New()
			if err := s.Parse(test.sdl, test.useStringDescriptions); err != nil {
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
