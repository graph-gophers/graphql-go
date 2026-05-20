package graphql

import (
	"context"
	"fmt"
	"reflect"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/exec/selected"
)

type directiveArgsCacheKey struct {
	def *ast.DirectiveDefinition
	typ reflect.Type
}

type DirectiveContext struct {
	owner       *Schema
	schema      *ast.Schema
	directive   *ast.Directive
	args        map[string]any
	fieldArgs   map[string]any
	decodedArgs map[reflect.Type]reflect.Value
}

func (c DirectiveContext) DecodeArgs(dst any) error {
	if c.schema == nil || c.directive == nil {
		return fmt.Errorf("directive context is missing schema metadata")
	}
	def := c.schema.Directives[c.directive.Name.Name]
	if def == nil {
		return fmt.Errorf("directive %q is not defined in the schema", c.directive.Name.Name)
	}
	if dst == nil {
		return fmt.Errorf("destination must be a non-nil pointer")
	}
	typ := reflect.TypeOf(dst)
	if typ.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer, got %s", typ)
	}
	rv := reflect.ValueOf(dst)
	if rv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}
	var (
		sp  *packer.StructPacker
		err error
	)
	if c.owner != nil {
		sp, err = c.owner.directiveArgsPacker(def, typ)
	} else {
		b := packer.NewBuilder()
		sp, err = b.MakeStructPacker(def.Arguments, typ)
		if err != nil {
			return err
		}
		err = b.Finish()
	}
	if err != nil {
		return err
	}
	if c.decodedArgs != nil {
		if packed, ok := c.decodedArgs[typ]; ok {
			rv.Elem().Set(packed.Elem())
			return nil
		}
	}
	packed, err := sp.Pack(c.args)
	if err != nil {
		return err
	}
	if c.decodedArgs != nil {
		c.decodedArgs[typ] = packed
	}
	rv.Elem().Set(packed.Elem())
	return nil
}

func (c DirectiveContext) FieldArg(name string, dst any) error {
	if c.fieldArgs == nil {
		return fmt.Errorf("directive context is missing field arguments")
	}
	if dst == nil {
		return fmt.Errorf("destination must be a non-nil pointer")
	}
	typ := reflect.TypeOf(dst)
	if typ.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer, got %s", typ)
	}
	rv := reflect.ValueOf(dst)
	if rv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	value, ok := c.fieldArgs[name]
	if !ok {
		return fmt.Errorf("field argument %q not found", name)
	}

	packed, err := (&packer.ValuePacker{ValueType: typ.Elem()}).Pack(value)
	if err != nil {
		return err
	}
	rv.Elem().Set(packed)
	return nil
}

// DirectiveVisitor defines the interface for directive visitors.
type DirectiveVisitor interface {
	// Name returns the name of the directive this visitor handles.
	Name() string
	// Visit is called when the directive is encountered during field definition traversal.
	// Use [DirectiveContext.DecodeArgs] to parse the directive arguments.
	Visit(ctx context.Context, d DirectiveContext) error
}

// DirectiveVisitors registers one or more directive visitors with the schema.
// Visitor names must be unique within a schema.
func DirectiveVisitors(visitors ...DirectiveVisitor) SchemaOpt {
	return func(s *Schema) {
		for _, v := range visitors {
			visitor, err := newDirectiveVisitor(v)
			if err != nil {
				s.optErr = err
				return
			}
			s.directiveVisitors = append(s.directiveVisitors, visitor)
		}
	}
}

func newDirectiveVisitor(visitor DirectiveVisitor) (DirectiveVisitor, error) {
	if visitor == nil {
		return nil, fmt.Errorf("directive visitor is nil")
	}

	name := visitor.Name()
	if name == "" {
		return nil, fmt.Errorf("directive visitor must have a non-empty name")
	}

	return visitor, nil
}

func (s *Schema) validateDirectiveVisitors() error {
	seen := make(map[string]struct{}, len(s.directiveVisitors))
	for _, v := range s.directiveVisitors {
		name := v.Name()
		def := s.schema.Directives[name]
		if def == nil {
			return fmt.Errorf("directive %q is not defined in the schema", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("directive visitor %q is already registered", name)
		}
		seen[name] = struct{}{}
	}

	return nil
}

func directiveArgs(schema *ast.Schema, d *ast.Directive, vars map[string]any) (map[string]any, error) {
	if d == nil {
		return nil, fmt.Errorf("directive is nil")
	}
	def := schema.Directives[d.Name.Name]
	if def == nil {
		return nil, fmt.Errorf("directive %q is not defined in the schema", d.Name.Name)
	}
	args := make(map[string]any, len(def.Arguments))
	for _, arg := range def.Arguments {
		if v, ok := d.Arguments.Get(arg.Name.Name); ok {
			if isNilValue(v) {
				args[arg.Name.Name] = nil
				continue
			}
			args[arg.Name.Name] = v.Deserialize(vars)
			continue
		}
		if arg.Default != nil {
			args[arg.Name.Name] = arg.Default.Deserialize(nil)
			continue
		}
		args[arg.Name.Name] = nil
	}
	return args, nil
}

func isNilValue(v ast.Value) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func (s *Schema) directiveArgsPacker(def *ast.DirectiveDefinition, typ reflect.Type) (*packer.StructPacker, error) {
	if def == nil {
		return nil, fmt.Errorf("directive definition is nil")
	}
	key := directiveArgsCacheKey{def: def, typ: typ}

	s.directiveArgsMu.RLock()
	sp := s.directiveArgsPackers[key]
	s.directiveArgsMu.RUnlock()
	if sp != nil {
		return sp, nil
	}

	b := packer.NewBuilder()
	sp, err := b.MakeStructPacker(def.Arguments, typ)
	if err != nil {
		return nil, err
	}
	if err := b.Finish(); err != nil {
		return nil, err
	}

	s.directiveArgsMu.Lock()
	defer s.directiveArgsMu.Unlock()
	if s.directiveArgsPackers == nil {
		s.directiveArgsPackers = make(map[directiveArgsCacheKey]*packer.StructPacker)
	}
	if cached := s.directiveArgsPackers[key]; cached != nil {
		return cached, nil
	}
	s.directiveArgsPackers[key] = sp
	return sp, nil
}

func (s *Schema) buildDirectiveCaches() {
	if s.directiveArgsPackers == nil {
		s.directiveArgsPackers = make(map[directiveArgsCacheKey]*packer.StructPacker)
	}
	s.directiveVisitorsByName = make(map[string]DirectiveVisitor, len(s.directiveVisitors))
	for _, v := range s.directiveVisitors {
		name := v.Name()
		s.directiveVisitorsByName[name] = v
	}
}

func (s *Schema) runDirectiveVisitors(ctx context.Context, vars map[string]any, sels []selected.Selection) []*errors.QueryError {
	if len(s.directiveVisitorsByName) == 0 {
		return nil
	}

	var errs []*errors.QueryError

	path := make([]any, 0, 8)
	var walk func([]selected.Selection)
	walk = func(selections []selected.Selection) {
		for _, sel := range selections {
			switch sel := sel.(type) {
			case *selected.SchemaField:
				path = append(path, sel.Alias)
				for _, d := range sel.Directives {
					hook, ok := s.directiveVisitorsByName[d.Name.Name]
					if !ok {
						continue
					}
					args, err := directiveArgs(s.schema, d, vars)
					if err != nil {
						errs = append(errs, &errors.QueryError{Message: err.Error(), Path: append([]any(nil), path...)})
						continue
					}
					dctx := DirectiveContext{
						owner:     s,
						schema:    s.schema,
						fieldArgs: sel.Args,
						directive: d,
						args:      args,
					}
					err = hook.Visit(ctx, dctx)
					if err != nil {
						errs = append(errs, &errors.QueryError{Message: err.Error(), Path: append([]any(nil), path...)})
					}
				}
				walk(sel.Sels)
				path = path[:len(path)-1]
			case *selected.TypeAssertion:
				walk(sel.Sels)
			}
		}
	}

	walk(sels)
	if len(errs) != 0 {
		return errs
	}
	return nil
}
