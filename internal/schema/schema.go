package schema

import (
	"fmt"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

// New initializes an instance of Schema.
func New() *ast.Schema {
	s := &ast.Schema{
		SchemaDefinition: ast.SchemaDefinition{
			EntryPointNames: make(map[string]string),
		},
		Types:      make(map[string]ast.NamedType),
		Directives: make(map[string]*ast.DirectiveDefinition),
	}
	m := newMeta()
	for n, t := range m.Types {
		s.Types[n] = t
	}
	for n, d := range m.Directives {
		s.Directives[n] = d
	}
	return s
}

func Parse(s *ast.Schema, schemaString string, useStringDescriptions bool) error {
	l := common.NewLexer(schemaString, useStringDescriptions)
	err := l.CatchSyntaxError(func() { parseSchema(s, l) })
	if err != nil {
		return err
	}

	if err := mergeExtensions(s); err != nil {
		return err
	}

	for _, t := range s.Types {
		if err := resolveNamedType(s, t); err != nil {
			return err
		}
	}
	for _, d := range s.Directives {
		for _, arg := range d.Arguments {
			t, err := common.ResolveType(arg.Type, s.Resolve)
			if err != nil {
				return err
			}
			arg.Type = t
		}
	}

	// https://graphql.github.io/graphql-spec/June2018/#sec-Root-Operation-Types
	// > While any type can be the root operation type for a GraphQL operation, the type system definition language can
	// > omit the schema definition when the query, mutation, and subscription root types are named Query, Mutation,
	// > and Subscription respectively.
	if len(s.EntryPointNames) == 0 {
		if _, ok := s.Types["Query"]; ok {
			s.EntryPointNames["query"] = "Query"
		}
		if _, ok := s.Types["Mutation"]; ok {
			s.EntryPointNames["mutation"] = "Mutation"
		}
		if _, ok := s.Types["Subscription"]; ok {
			s.EntryPointNames["subscription"] = "Subscription"
		}
	}
	s.RootOperationTypes = make(map[string]ast.NamedType)
	for key, name := range s.EntryPointNames {
		t, ok := s.Types[name]
		if !ok {
			return errors.Errorf("type %q not found", name)
		}
		s.RootOperationTypes[key] = t
	}

	// Interface types need validation: https://spec.graphql.org/draft/#sec-Interfaces.Interfaces-Implementing-Interfaces
	for _, typeDef := range s.Types {
		switch t := typeDef.(type) {
		case *ast.InterfaceTypeDefinition:
			for i, implements := range t.Interfaces {
				typ, ok := s.Types[implements.Name]
				if !ok {
					return errors.Errorf("interface %q not found", implements)
				}
				inteface, ok := typ.(*ast.InterfaceTypeDefinition)
				if !ok {
					return errors.Errorf("type %q is not an interface", inteface)
				}

				for _, f := range inteface.Fields.Names() {
					if t.Fields.Get(f) == nil {
						return errors.Errorf("interface %q expects field %q but %q does not provide it", inteface.Name, f, t.Name)
					}
				}

				t.Interfaces[i] = inteface
			}
		default:
			continue
		}
	}

	for _, obj := range s.Objects {
		obj.Interfaces = make([]*ast.InterfaceTypeDefinition, len(obj.InterfaceNames))
		if err := resolveDirectives(s, obj.Directives, "OBJECT"); err != nil {
			return err
		}
		for _, field := range obj.Fields {
			if err := resolveDirectives(s, field.Directives, "FIELD_DEFINITION"); err != nil {
				return err
			}
		}
		for i, intfName := range obj.InterfaceNames {
			t, ok := s.Types[intfName]
			if !ok {
				return errors.Errorf("interface %q not found", intfName)
			}
			intf, ok := t.(*ast.InterfaceTypeDefinition)
			if !ok {
				return errors.Errorf("type %q is not an interface", intfName)
			}
			for _, f := range intf.Fields.Names() {
				if obj.Fields.Get(f) == nil {
					return errors.Errorf("interface %q expects field %q but %q does not provide it", intfName, f, obj.Name)
				}
			}
			obj.Interfaces[i] = intf
			intf.PossibleTypes = append(intf.PossibleTypes, obj)
		}
	}

	for _, union := range s.Unions {
		if err := resolveDirectives(s, union.Directives, "UNION"); err != nil {
			return err
		}
		union.UnionMemberTypes = make([]*ast.ObjectTypeDefinition, len(union.TypeNames))
		for i, name := range union.TypeNames {
			t, ok := s.Types[name]
			if !ok {
				return errors.Errorf("object type %q not found", name)
			}
			obj, ok := t.(*ast.ObjectTypeDefinition)
			if !ok {
				return errors.Errorf("type %q is not an object", name)
			}
			union.UnionMemberTypes[i] = obj
		}
	}

	for _, enum := range s.Enums {
		if err := resolveDirectives(s, enum.Directives, "ENUM"); err != nil {
			return err
		}
		for _, value := range enum.EnumValuesDefinition {
			if err := resolveDirectives(s, value.Directives, "ENUM_VALUE"); err != nil {
				return err
			}
		}
	}

	s.SchemaString = schemaString

	return nil
}

func ParseSchema(schemaString string, useStringDescriptions bool) (*ast.Schema, error) {
	s := New()
	err := Parse(s, schemaString, useStringDescriptions)
	return s, err
}

func mergeExtensions(s *ast.Schema) error {
	for _, ext := range s.Extensions {
		typ := s.Types[ext.Type.TypeName()]
		if typ == nil {
			return fmt.Errorf("trying to extend unknown type %q", ext.Type.TypeName())
		}

		if typ.Kind() != ext.Type.Kind() {
			return fmt.Errorf("trying to extend type %q with type %q", typ.Kind(), ext.Type.Kind())
		}

		switch og := typ.(type) {
		case *ast.ObjectTypeDefinition:
			e := ext.Type.(*ast.ObjectTypeDefinition)

			for _, field := range e.Fields {
				if og.Fields.Get(field.Name) != nil {
					return fmt.Errorf("extended field %q already exists", field.Name)
				}
			}
			og.Fields = append(og.Fields, e.Fields...)

			for _, en := range e.InterfaceNames {
				for _, on := range og.InterfaceNames {
					if on == en {
						return fmt.Errorf("interface %q implemented in the extension is already implemented in %q", on, og.Name)
					}
				}
			}
			og.InterfaceNames = append(og.InterfaceNames, e.InterfaceNames...)

		case *ast.InputObject:
			e := ext.Type.(*ast.InputObject)

			for _, field := range e.Values {
				if og.Values.Get(field.Name.Name) != nil {
					return fmt.Errorf("extended field %q already exists", field.Name)
				}
			}
			og.Values = append(og.Values, e.Values...)

		case *ast.InterfaceTypeDefinition:
			e := ext.Type.(*ast.InterfaceTypeDefinition)

			for _, field := range e.Fields {
				if og.Fields.Get(field.Name) != nil {
					return fmt.Errorf("extended field %s already exists", field.Name)
				}
			}
			og.Fields = append(og.Fields, e.Fields...)

		case *ast.Union:
			e := ext.Type.(*ast.Union)

			for _, en := range e.TypeNames {
				for _, on := range og.TypeNames {
					if on == en {
						return fmt.Errorf("union type %q already declared in %q", on, og.Name)
					}
				}
			}
			og.TypeNames = append(og.TypeNames, e.TypeNames...)

		case *ast.EnumTypeDefinition:
			e := ext.Type.(*ast.EnumTypeDefinition)

			for _, en := range e.EnumValuesDefinition {
				for _, on := range og.EnumValuesDefinition {
					if on.EnumValue == en.EnumValue {
						return fmt.Errorf("enum value %q already declared in %q", on.EnumValue, og.Name)
					}
				}
			}
			og.EnumValuesDefinition = append(og.EnumValuesDefinition, e.EnumValuesDefinition...)
		default:
			return fmt.Errorf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union" or "input"`, og.TypeName())
		}
	}

	return nil
}

func resolveNamedType(s *ast.Schema, t ast.NamedType) error {
	switch t := t.(type) {
	case *ast.ObjectTypeDefinition:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *ast.InterfaceTypeDefinition:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
		if err := resolveDirectives(s, t.Directives, "INTERFACE"); err != nil {
			return err
		}
	case *ast.InputObject:
		if err := resolveInputObject(s, t.Values); err != nil {
			return err
		}
		if err := resolveDirectives(s, t.Directives, "INPUT_OBJECT"); err != nil {
			return err
		}
	case *ast.ScalarTypeDefinition:
		if err := resolveDirectives(s, t.Directives, "SCALAR"); err != nil {
			return err
		}
	}
	return nil
}

func resolveField(s *ast.Schema, f *ast.FieldDefinition) error {
	t, err := common.ResolveType(f.Type, s.Resolve)
	if err != nil {
		return err
	}
	f.Type = t
	if err := resolveDirectives(s, f.Directives, "FIELD_DEFINITION"); err != nil {
		return err
	}
	return resolveInputObject(s, f.Arguments)
}

func resolveDirectives(s *ast.Schema, directives ast.DirectiveList, loc string) error {
	alreadySeenNonRepeatable := make(map[string]struct{})
	for _, d := range directives {
		dirName := d.Name.Name
		dd, ok := s.Directives[dirName]
		if !ok {
			return errors.Errorf("directive %q not found", dirName)
		}
		validLoc := false
		for _, l := range dd.Locations {
			if l == loc {
				validLoc = true
				break
			}
		}
		if !validLoc {
			return errors.Errorf("invalid location %q for directive %q (must be one of %v)", loc, dirName, dd.Locations)
		}
		for _, arg := range d.Arguments {
			if dd.Arguments.Get(arg.Name.Name) == nil {
				return errors.Errorf("invalid argument %q for directive %q", arg.Name.Name, dirName)
			}
		}
		for _, arg := range dd.Arguments {
			if _, ok := d.Arguments.Get(arg.Name.Name); !ok {
				d.Arguments = append(d.Arguments, &ast.Argument{Name: arg.Name, Value: arg.Default})
			}
		}

		if dd.Repeatable {
			continue
		}
		if _, seen := alreadySeenNonRepeatable[dirName]; seen {
			return errors.Errorf(`non repeatable directive %q can not be repeated. Consider adding "repeatable".`, dirName)
		}
		alreadySeenNonRepeatable[dirName] = struct{}{}
	}
	return nil
}

func resolveInputObject(s *ast.Schema, values ast.ArgumentsDefinition) error {
	for _, v := range values {
		t, err := common.ResolveType(v.Type, s.Resolve)
		if err != nil {
			return err
		}
		v.Type = t

		if err := resolveDirectives(s, v.Directives, "ARGUMENT_DEFINITION"); err != nil {
			return err
		}

	}
	return nil
}

func parseSchema(s *ast.Schema, l *common.Lexer) {
	l.ConsumeWhitespace()

	for l.Peek() != scanner.EOF {
		desc := l.DescComment()
		switch x := l.ConsumeIdent(); x {

		case "schema":
			s.SchemaDefinition.Present = true
			s.SchemaDefinition.Loc = l.Location()
			s.SchemaDefinition.Desc = desc
			s.SchemaDefinition.Directives = common.ParseDirectives(l)
			l.ConsumeToken('{')
			for l.Peek() != '}' {

				name := l.ConsumeIdent()
				l.ConsumeToken(':')
				typ := l.ConsumeIdent()
				s.EntryPointNames[name] = typ
			}
			l.ConsumeToken('}')

		case "type":
			obj := parseObjectDef(l)
			obj.Desc = desc
			s.Types[obj.Name] = obj
			s.Objects = append(s.Objects, obj)

		case "interface":
			iface := parseInterfaceDef(l)
			iface.Desc = desc
			s.Types[iface.Name] = iface

		case "union":
			union := parseUnionDef(l)
			union.Desc = desc
			s.Types[union.Name] = union
			s.Unions = append(s.Unions, union)

		case "enum":
			enum := parseEnumDef(l)
			enum.Desc = desc
			s.Types[enum.Name] = enum
			s.Enums = append(s.Enums, enum)

		case "input":
			input := parseInputDef(l)
			input.Desc = desc
			s.Types[input.Name] = input

		case "scalar":
			loc := l.Location()
			name := l.ConsumeIdent()
			directives := common.ParseDirectives(l)
			s.Types[name] = &ast.ScalarTypeDefinition{Name: name, Desc: desc, Directives: directives, Loc: loc}

		case "directive":
			directive := parseDirectiveDef(l)
			directive.Desc = desc
			s.Directives[directive.Name] = directive

		case "extend":
			parseExtension(s, l)

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union", "input", "scalar" or "directive"`, x))
		}
	}
}

func parseObjectDef(l *common.Lexer) *ast.ObjectTypeDefinition {
	object := &ast.ObjectTypeDefinition{Loc: l.Location(), Name: l.ConsumeIdent()}

	for {
		if l.Peek() == '{' {
			break
		}

		if l.Peek() == '@' {
			object.Directives = common.ParseDirectives(l)
			continue
		}

		if l.Peek() != scanner.Ident {
			break
		}

		l.ConsumeKeyword("implements")

		for l.Peek() != '{' && l.Peek() != '@' {
			if l.Peek() == '&' {
				l.ConsumeToken('&')
			}

			object.InterfaceNames = append(object.InterfaceNames, l.ConsumeIdent())
		}
	}
	l.ConsumeToken('{')
	object.Fields = parseFieldsDef(l)
	l.ConsumeToken('}')

	return object

}

func parseInterfaceDef(l *common.Lexer) *ast.InterfaceTypeDefinition {
	i := &ast.InterfaceTypeDefinition{Loc: l.Location(), Name: l.ConsumeIdent()}

	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")
		i.Interfaces = append(i.Interfaces, &ast.InterfaceTypeDefinition{Name: l.ConsumeIdent()})

		for l.Peek() == '&' {
			l.ConsumeToken('&')
			i.Interfaces = append(i.Interfaces, &ast.InterfaceTypeDefinition{Name: l.ConsumeIdent()})
		}
	}

	i.Directives = common.ParseDirectives(l)

	l.ConsumeToken('{')
	i.Fields = parseFieldsDef(l)
	l.ConsumeToken('}')

	return i
}

func parseUnionDef(l *common.Lexer) *ast.Union {
	union := &ast.Union{Loc: l.Location(), Name: l.ConsumeIdent()}

	union.Directives = common.ParseDirectives(l)
	l.ConsumeToken('=')
	union.TypeNames = []string{l.ConsumeIdent()}
	for l.Peek() == '|' {
		l.ConsumeToken('|')
		union.TypeNames = append(union.TypeNames, l.ConsumeIdent())
	}

	return union
}

func parseInputDef(l *common.Lexer) *ast.InputObject {
	i := &ast.InputObject{}
	i.Loc = l.Location()
	i.Name = l.ConsumeIdent()
	i.Directives = common.ParseDirectives(l)
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		i.Values = append(i.Values, common.ParseInputValue(l))
	}
	l.ConsumeToken('}')
	return i
}

func parseEnumDef(l *common.Lexer) *ast.EnumTypeDefinition {
	enum := &ast.EnumTypeDefinition{Loc: l.Location(), Name: l.ConsumeIdent()}

	enum.Directives = common.ParseDirectives(l)
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		v := &ast.EnumValueDefinition{
			Desc:       l.DescComment(),
			Loc:        l.Location(),
			EnumValue:  l.ConsumeIdent(),
			Directives: common.ParseDirectives(l),
		}

		enum.EnumValuesDefinition = append(enum.EnumValuesDefinition, v)
	}
	l.ConsumeToken('}')
	return enum
}
func parseDirectiveDef(l *common.Lexer) *ast.DirectiveDefinition {
	l.ConsumeToken('@')
	loc := l.Location()
	d := &ast.DirectiveDefinition{Name: l.ConsumeIdent(), Loc: loc}

	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			v := common.ParseInputValue(l)
			d.Arguments = append(d.Arguments, v)
		}
		l.ConsumeToken(')')
	}

	switch x := l.ConsumeIdent(); x {
	case "on":
		// no-op; Go doesn't fallthrough by default
	case "repeatable":
		d.Repeatable = true
		l.ConsumeKeyword("on")
	default:
		l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "on" or "repeatable"`, x))
	}

	for {
		loc := l.ConsumeIdent()
		if _, ok := legalDirectiveLocationNames[loc]; !ok {
			l.SyntaxError(fmt.Sprintf("%q is not a legal directive location (options: %v)", loc, legalDirectiveLocationNames))
		}
		d.Locations = append(d.Locations, loc)
		if l.Peek() != '|' {
			break
		}
		l.ConsumeToken('|')
	}
	return d
}

func parseExtension(s *ast.Schema, l *common.Lexer) {
	loc := l.Location()
	switch x := l.ConsumeIdent(); x {
	case "schema":
		l.ConsumeToken('{')
		s.SchemaDefinition.Present = true
		s.SchemaDefinition.Directives = append(s.SchemaDefinition.Directives, common.ParseDirectives(l)...)
		for l.Peek() != '}' {
			name := l.ConsumeIdent()
			l.ConsumeToken(':')
			typ := l.ConsumeIdent()
			s.EntryPointNames[name] = typ
		}
		l.ConsumeToken('}')

	case "type":
		obj := parseObjectDef(l)
		s.Extensions = append(s.Extensions, &ast.Extension{Type: obj, Loc: loc})

	case "interface":
		iface := parseInterfaceDef(l)
		s.Extensions = append(s.Extensions, &ast.Extension{Type: iface, Loc: loc})

	case "union":
		union := parseUnionDef(l)
		s.Extensions = append(s.Extensions, &ast.Extension{Type: union, Loc: loc})

	case "enum":
		enum := parseEnumDef(l)
		s.Extensions = append(s.Extensions, &ast.Extension{Type: enum, Loc: loc})

	case "input":
		input := parseInputDef(l)
		s.Extensions = append(s.Extensions, &ast.Extension{Type: input, Loc: loc})

	default:
		// TODO: Add ScalarTypeDefinition when adding directives
		l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union" or "input"`, x))
	}
}

func parseFieldsDef(l *common.Lexer) ast.FieldsDefinition {
	var fields ast.FieldsDefinition
	for l.Peek() != '}' {
		f := &ast.FieldDefinition{}
		f.Desc = l.DescComment()
		f.Loc = l.Location()
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				f.Arguments = append(f.Arguments, common.ParseInputValue(l))
			}
			l.ConsumeToken(')')
		}
		l.ConsumeToken(':')
		f.Type = common.ParseType(l)
		f.Directives = common.ParseDirectives(l)
		fields = append(fields, f)
	}
	return fields
}

var legalDirectiveLocationNames = map[string]struct{}{
	"SCHEMA":                 {},
	"SCALAR":                 {},
	"OBJECT":                 {},
	"FIELD_DEFINITION":       {},
	"ARGUMENT_DEFINITION":    {},
	"INTERFACE":              {},
	"UNION":                  {},
	"ENUM":                   {},
	"ENUM_VALUE":             {},
	"INPUT_OBJECT":           {},
	"INPUT_FIELD_DEFINITION": {},
	"QUERY":                  {},
	"MUTATION":               {},
	"SUBSCRIPTION":           {},
	"FIELD":                  {},
	"FRAGMENT_DEFINITION":    {},
	"FRAGMENT_SPREAD":        {},
	"INLINE_FRAGMENT":        {},
	"VARIABLE_DEFINITION":    {},
}
