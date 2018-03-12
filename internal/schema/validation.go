package schema

import (
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go/errors"
)

func validateEntryPointName(s *Schema, entryPoint *EntryPoint) error {
	if s == Meta {
		return nil
	}
	switch name := entryPoint.Name; name {
	case "query", "mutation", "subscription":
		if prev, ok := s.entryPointNames[name]; ok {
			return &errors.QueryError{
				Message:   fmt.Sprintf(`%q operation provided more than once`, name),
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
	if err := validatePrefix(name, t.Location()); err != nil {
		return err
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
	if err := validatePrefix(name, directive.Loc); err != nil {
		return err
	}
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

func validatePrefix(name string, loc errors.Location) error {
	if strings.HasPrefix(name, "__") {
		return &errors.QueryError{
			Message:   fmt.Sprintf(`%q must not begin with "__", reserved for introspection types`, name),
			Locations: []errors.Location{loc},
		}
	}
	return nil
}
