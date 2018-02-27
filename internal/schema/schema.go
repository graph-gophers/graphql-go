package schema

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
)

type Schema struct {
	EntryPoints map[string]NamedType
	Types       map[string]NamedType
	Directives  map[string]*DirectiveDecl

	entryPointNames map[string]*EntryPoint
	objects         []*Object
	unions          []*Union
	enums           []*Enum
}

func (s *Schema) Resolve(name string) common.Type {
	return s.Types[name]
}

type EntryPoint struct {
	Name string
	Type string
	Loc  errors.Location
}

type NamedType interface {
	common.Type
	TypeName() string
	Description() string
	Location() errors.Location
}

type Scalar struct {
	Name string
	Desc string
	Loc  errors.Location
}

type Object struct {
	Name       string
	Interfaces []*Interface
	Fields     FieldList
	Desc       string
	Loc        errors.Location

	interfaceNames []string
}

type Interface struct {
	Name          string
	PossibleTypes []*Object
	Fields        FieldList
	Desc          string
	Loc           errors.Location
}

type Union struct {
	Name          string
	PossibleTypes []*Object
	Desc          string
	Loc           errors.Location

	typeNames []string
}

type Enum struct {
	Name   string
	Values []*EnumValue
	Desc   string
	Loc    errors.Location
}

type EnumValue struct {
	Name       string
	Directives common.DirectiveList
	Desc       string
}

type InputObject struct {
	Name   string
	Desc   string
	Values common.InputValueList
	Loc    errors.Location
}

type FieldList []*Field

func (l FieldList) Get(name string) *Field {
	for _, f := range l {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (l FieldList) Names() []string {
	names := make([]string, len(l))
	for i, f := range l {
		names[i] = f.Name
	}
	return names
}

type DirectiveDecl struct {
	Name string
	Desc string
	Loc  errors.Location
	Locs []string
	Args common.InputValueList
}

func (*Scalar) Kind() string      { return "SCALAR" }
func (*Object) Kind() string      { return "OBJECT" }
func (*Interface) Kind() string   { return "INTERFACE" }
func (*Union) Kind() string       { return "UNION" }
func (*Enum) Kind() string        { return "ENUM" }
func (*InputObject) Kind() string { return "INPUT_OBJECT" }

func (t *Scalar) String() string      { return t.Name }
func (t *Object) String() string      { return t.Name }
func (t *Interface) String() string   { return t.Name }
func (t *Union) String() string       { return t.Name }
func (t *Enum) String() string        { return t.Name }
func (t *InputObject) String() string { return t.Name }

func (t *Scalar) TypeName() string      { return t.Name }
func (t *Object) TypeName() string      { return t.Name }
func (t *Interface) TypeName() string   { return t.Name }
func (t *Union) TypeName() string       { return t.Name }
func (t *Enum) TypeName() string        { return t.Name }
func (t *InputObject) TypeName() string { return t.Name }

func (t *Scalar) Description() string      { return t.Desc }
func (t *Object) Description() string      { return t.Desc }
func (t *Interface) Description() string   { return t.Desc }
func (t *Union) Description() string       { return t.Desc }
func (t *Enum) Description() string        { return t.Desc }
func (t *InputObject) Description() string { return t.Desc }

func (t *Scalar) Location() errors.Location      { return t.Loc }
func (t *Object) Location() errors.Location      { return t.Loc }
func (t *Interface) Location() errors.Location   { return t.Loc }
func (t *Union) Location() errors.Location       { return t.Loc }
func (t *Enum) Location() errors.Location        { return t.Loc }
func (t *InputObject) Location() errors.Location { return t.Loc }

type Field struct {
	Name       string
	Args       common.InputValueList
	Type       common.Type
	Directives common.DirectiveList
	Desc       string
}

func New() *Schema {
	s := &Schema{
		entryPointNames: make(map[string]*EntryPoint),
		Types:           make(map[string]NamedType),
		Directives:      make(map[string]*DirectiveDecl),
	}
	for n, t := range Meta.Types {
		s.Types[n] = t
	}
	for n, d := range Meta.Directives {
		s.Directives[n] = d
	}
	return s
}

func (s *Schema) Parse(schemaString string) error {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(schemaString))

	l := common.New(sc)
	var err error
	syntaxErr := l.CatchSyntaxError(func() {
		err = parseSchema(s, l)
	})
	if syntaxErr != nil {
		return syntaxErr
	}
	if err != nil {
		return err
	}

	for _, t := range s.Types {
		if err := resolveNamedType(s, t); err != nil {
			return err
		}
	}
	for _, d := range s.Directives {
		for _, arg := range d.Args {
			t, err := common.ResolveType(arg.Type, s.Resolve)
			if err != nil {
				return err
			}
			arg.Type = t
		}
	}

	s.EntryPoints = make(map[string]NamedType)
	for key, e := range s.entryPointNames {
		t, ok := s.Types[e.Type]
		if !ok {
			return errors.Errorf("type %q not found", e.Type)
		}
		s.EntryPoints[key] = t
	}

	for _, obj := range s.objects {
		obj.Interfaces = make([]*Interface, len(obj.interfaceNames))
		for i, intfName := range obj.interfaceNames {
			t, ok := s.Types[intfName]
			if !ok {
				return errors.Errorf("interface %q not found %s", intfName)
			}
			intf, ok := t.(*Interface)
			if !ok {
				return errors.Errorf("type %q is not an interface", intfName)
			}
			obj.Interfaces[i] = intf
			intf.PossibleTypes = append(intf.PossibleTypes, obj)
		}
	}

	for _, union := range s.unions {
		union.PossibleTypes = make([]*Object, len(union.typeNames))
		for i, name := range union.typeNames {
			t, ok := s.Types[name]
			if !ok {
				return errors.Errorf("object type %q not found", name)
			}
			obj, ok := t.(*Object)
			if !ok {
				return errors.Errorf("type %q is not an object", name)
			}
			union.PossibleTypes[i] = obj
		}
	}

	for _, enum := range s.enums {
		for _, value := range enum.Values {
			if err := resolveDirectives(s, value.Directives); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveNamedType(s *Schema, t NamedType) error {
	switch t := t.(type) {
	case *Object:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *Interface:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *InputObject:
		if err := resolveInputObject(s, t.Values); err != nil {
			return err
		}
	}
	return nil
}

func resolveField(s *Schema, f *Field) error {
	t, err := common.ResolveType(f.Type, s.Resolve)
	if err != nil {
		return err
	}
	f.Type = t
	if err := resolveDirectives(s, f.Directives); err != nil {
		return err
	}
	return resolveInputObject(s, f.Args)
}

func resolveDirectives(s *Schema, directives common.DirectiveList) error {
	for _, d := range directives {
		dirName := d.Name.Name
		dd, ok := s.Directives[dirName]
		if !ok {
			return errors.Errorf("directive %q not found", dirName)
		}
		for _, arg := range d.Args {
			if dd.Args.Get(arg.Name.Name) == nil {
				return errors.Errorf("invalid argument %q for directive %q", arg.Name.Name, dirName)
			}
		}
		for _, arg := range dd.Args {
			if _, ok := d.Args.Get(arg.Name.Name); !ok {
				d.Args = append(d.Args, common.Argument{Name: arg.Name, Value: arg.Default})
			}
		}
	}
	return nil
}

func resolveInputObject(s *Schema, values common.InputValueList) error {
	for _, v := range values {
		t, err := common.ResolveType(v.Type, s.Resolve)
		if err != nil {
			return err
		}
		v.Type = t
	}
	return nil
}

func parseSchema(s *Schema, l *common.Lexer) error {
	for l.Peek() != scanner.EOF {
		desc := l.DescComment()
		switch x := l.ConsumeIdent(); x {
		case "schema":
			l.ConsumeToken('{')
			for l.Peek() != '}' {
				ident := l.ConsumeIdentWithLoc()
				name := ident.Name
				l.ConsumeToken(':')
				typeIdent := l.ConsumeIdentWithLoc()
				entryPoint := &EntryPoint{Name: name, Type: typeIdent.Name, Loc: ident.Loc}
				if err := validateEntryPointName(s, entryPoint); err != nil {
					return err
				}
				s.entryPointNames[name] = entryPoint
			}
			l.ConsumeToken('}')
		case "type":
			obj := parseObjectDecl(l)
			obj.Desc = desc
			if err := validateTypeName(s, obj); err != nil {
				return err
			}
			s.Types[obj.Name] = obj
			s.objects = append(s.objects, obj)
		case "interface":
			intf := parseInterfaceDecl(l)
			intf.Desc = desc
			if err := validateTypeName(s, intf); err != nil {
				return err
			}
			s.Types[intf.Name] = intf
		case "union":
			union := parseUnionDecl(l)
			union.Desc = desc
			if err := validateTypeName(s, union); err != nil {
				return err
			}
			s.Types[union.Name] = union
			s.unions = append(s.unions, union)
		case "enum":
			enum := parseEnumDecl(l)
			enum.Desc = desc
			if err := validateTypeName(s, enum); err != nil {
				return err
			}
			s.Types[enum.Name] = enum
			s.enums = append(s.enums, enum)
		case "input":
			input := parseInputDecl(l)
			input.Desc = desc
			if err := validateTypeName(s, input); err != nil {
				return err
			}
			s.Types[input.Name] = input
		case "scalar":
			ident := l.ConsumeIdentWithLoc()
			name := ident.Name
			scalar := &Scalar{Name: name, Desc: desc, Loc: ident.Loc}
			if err := validateTypeName(s, scalar); err != nil {
				return err
			}
			s.Types[name] = scalar
		case "directive":
			directive := parseDirectiveDecl(l)
			directive.Desc = desc
			if err := validateDirectiveName(s, directive); err != nil {
				return err
			}
			s.Directives[directive.Name] = directive
		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union", "input", "scalar" or "directive"`, x))
		}
	}
	return nil
}

func validateEntryPointName(s *Schema, entryPoint *EntryPoint) error {
	if s == Meta {
		return nil
	}
	switch name := entryPoint.Name; name {
	case "query", "mutation", "subscription":
		if prev, ok := s.entryPointNames[name]; ok {
			return &errors.QueryError{
				Message:   fmt.Sprintf(`%q type provided more than once`, name),
				Locations: []errors.Location{prev.Loc, entryPoint.Loc},
			}
		}
	default:
		return &errors.QueryError{
			Message:   fmt.Sprintf(`unexpected %q, expected "query", "mutation" or "subscription"`, name),
			Locations: []errors.Location{entryPoint.Loc},
		}
	}
	return nil
}

func validateTypeName(s *Schema, t NamedType) error {
	if s == Meta {
		return nil
	}
	name := t.TypeName()
	if strings.HasPrefix(name, "__") {
		return &errors.QueryError{
			Message:   fmt.Sprintf(`%q must not begin with "__", reserved for introspection types`, name),
			Locations: []errors.Location{t.Location()},
		}
	}
	if _, ok := Meta.Types[name]; ok {
		return &errors.QueryError{
			Message:   fmt.Sprintf(`built-in type %q redefined`, name),
			Locations: []errors.Location{t.Location()},
		}
	}
	if prev, ok := s.Types[name]; ok {
		return &errors.QueryError{
			Message:   fmt.Sprintf(`%q defined more than once`, name),
			Locations: []errors.Location{prev.Location(), t.Location()},
		}
	}
	return nil
}

func validateDirectiveName(s *Schema, directive *DirectiveDecl) error {
	if s == Meta {
		return nil
	}
	name := directive.Name
	if _, ok := Meta.Directives[name]; ok {
		return &errors.QueryError{
			Message:   fmt.Sprintf(`built-in directive %q redefined`, name),
			Locations: []errors.Location{directive.Loc},
		}
	}
	if prev, ok := s.Directives[name]; ok {
		return &errors.QueryError{
			Message:   fmt.Sprintf("%q defined more than once", name),
			Locations: []errors.Location{prev.Loc, directive.Loc},
		}
	}
	return nil
}

func parseObjectDecl(l *common.Lexer) *Object {
	o := &Object{}
	ident := l.ConsumeIdentWithLoc()
	o.Name = ident.Name
	o.Loc = ident.Loc
	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")
		for {
			o.interfaceNames = append(o.interfaceNames, l.ConsumeIdent())
			if l.Peek() == '{' {
				break
			}
		}
	}
	l.ConsumeToken('{')
	o.Fields = parseFields(l)
	l.ConsumeToken('}')
	return o
}

func parseInterfaceDecl(l *common.Lexer) *Interface {
	i := &Interface{}
	ident := l.ConsumeIdentWithLoc()
	i.Name = ident.Name
	i.Loc = ident.Loc
	l.ConsumeToken('{')
	i.Fields = parseFields(l)
	l.ConsumeToken('}')
	return i
}

func parseUnionDecl(l *common.Lexer) *Union {
	union := &Union{}
	ident := l.ConsumeIdentWithLoc()
	union.Name = ident.Name
	union.Loc = ident.Loc
	l.ConsumeToken('=')
	union.typeNames = []string{l.ConsumeIdent()}
	for l.Peek() == '|' {
		l.ConsumeToken('|')
		union.typeNames = append(union.typeNames, l.ConsumeIdent())
	}
	return union
}

func parseInputDecl(l *common.Lexer) *InputObject {
	i := &InputObject{}
	ident := l.ConsumeIdentWithLoc()
	i.Name = ident.Name
	i.Loc = ident.Loc
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		i.Values = append(i.Values, common.ParseInputValue(l))
	}
	l.ConsumeToken('}')
	return i
}

func parseEnumDecl(l *common.Lexer) *Enum {
	enum := &Enum{}
	ident := l.ConsumeIdentWithLoc()
	enum.Name = ident.Name
	enum.Loc = ident.Loc
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		v := &EnumValue{}
		v.Desc = l.DescComment()
		v.Name = l.ConsumeIdent()
		v.Directives = common.ParseDirectives(l)
		enum.Values = append(enum.Values, v)
	}
	l.ConsumeToken('}')
	return enum
}

func parseDirectiveDecl(l *common.Lexer) *DirectiveDecl {
	d := &DirectiveDecl{}
	l.ConsumeToken('@')
	ident := l.ConsumeIdentWithLoc()
	d.Name = ident.Name
	d.Loc = ident.Loc
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			v := common.ParseInputValue(l)
			d.Args = append(d.Args, v)
		}
		l.ConsumeToken(')')
	}
	l.ConsumeKeyword("on")
	for {
		loc := l.ConsumeIdent()
		d.Locs = append(d.Locs, loc)
		if l.Peek() != '|' {
			break
		}
		l.ConsumeToken('|')
	}
	return d
}

func parseFields(l *common.Lexer) FieldList {
	var fields FieldList
	for l.Peek() != '}' {
		f := &Field{}
		f.Desc = l.DescComment()
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				f.Args = append(f.Args, common.ParseInputValue(l))
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
