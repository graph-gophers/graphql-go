package exec

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
	"github.com/graph-gophers/graphql-go/internal/exec/selected"
)

func collectFieldsToValidate(sels []selected.Selection, s *resolvable.Schema, fields *[]*fieldToValidate, fieldByAlias map[string]*fieldToValidate) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *selected.SchemaField:
			field, ok := fieldByAlias[sel.Alias]
			if !ok { // validation already checked for conflict (TODO)
				field = &fieldToValidate{field: sel}
				fieldByAlias[sel.Alias] = field
				*fields = append(*fields, field)
			}
			field.sels = append(field.sels, sel.Sels...)
		case *selected.TypenameField:
			// Ignore __typename, which has no directives
		case *selected.TypeAssertion:
			collectFieldsToValidate(sel.Sels, s, fields, fieldByAlias)
		default:
			panic(fmt.Sprintf("unexpected selection type %T", sel))
		}
	}
}

func validateFieldSelection(ctx context.Context, s *resolvable.Schema, f *fieldToValidate, path *pathSegment) []*errors.QueryError {
	if f.field.FixedResult.IsValid() {
		// Skip fixed result meta fields like __TypeName
		return nil
	}

	var args interface{}

	if f.field.ArgsPacker != nil {
		args = f.field.PackedArgs.Interface()
	}

	vErrs := f.field.Validate(ctx, args)

	if l := len(vErrs); l > 0 {
		errs := make([]*errors.QueryError, l)

		for i, err := range vErrs {
			var ext map[string]interface{}
			if e, ok := err.(extensionser); ok {
				ext = e.Extensions()
			}
			errs[i] = &errors.QueryError{
				Err:        err,
				Message:    err.Error(),
				Locations:  []errors.Location{f.field.Loc},
				Path:       path.toSlice(),
				Extensions: ext,
			}
		}

		return errs
	}

	return validateSelectionSet(ctx, f.sels, f.field.Type, path, s)
}

func validateSelectionSet(ctx context.Context, sels []selected.Selection, typ ast.Type, path *pathSegment, s *resolvable.Schema) []*errors.QueryError {
	t, _ := unwrapNonNull(typ)

	switch t.(type) {
	case *ast.ObjectTypeDefinition, *ast.InterfaceTypeDefinition, *ast.Union:
		return validateSelections(ctx, sels, path, s)
	}

	switch t := t.(type) {
	case *ast.List:
		return validateList(ctx, sels, t, path, s)
	case *ast.ScalarTypeDefinition, *ast.EnumTypeDefinition:
		// Field resolution already validated, don't need to check the value
	default:
		panic(fmt.Sprintf("unexpected type %T", t))
	}

	return nil
}

func validateSelections(ctx context.Context, sels []selected.Selection, path *pathSegment, s *resolvable.Schema) (errs []*errors.QueryError) {
	var fields []*fieldToValidate
	collectFieldsToValidate(sels, s, &fields, make(map[string]*fieldToValidate))

	for _, f := range fields {
		errs = append(errs, validateFieldSelection(ctx, s, f, &pathSegment{path, f.field.Alias})...)
	}

	return errs
}

func validateList(ctx context.Context, sels []selected.Selection, typ *ast.List, path *pathSegment, s *resolvable.Schema) []*errors.QueryError {
	// For lists, we only need to apply validation once. Nothing has been evaluated, so we have no list, and need to use '0' as the path index
	return validateSelectionSet(ctx, sels, typ.OfType, &pathSegment{path, 0}, s)
}
