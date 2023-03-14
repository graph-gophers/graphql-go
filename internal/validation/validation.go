package validation

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/query"
)

type varSet map[*ast.InputValueDefinition]struct{}

type selectionPair struct{ a, b ast.Selection }

type nameSet map[string]errors.Location

type fieldInfo struct {
	sf     *ast.FieldDefinition
	parent ast.NamedType
}

type context struct {
	schema           *ast.Schema
	doc              *ast.ExecutableDefinition
	errs             []*errors.QueryError
	opErrs           map[*ast.OperationDefinition][]*errors.QueryError
	usedVars         map[*ast.OperationDefinition]varSet
	fieldMap         map[*ast.Field]fieldInfo
	overlapValidated map[selectionPair]struct{}
	maxDepth         int
}

func (c *context) addErr(loc errors.Location, rule string, format string, a ...interface{}) {
	c.addErrMultiLoc([]errors.Location{loc}, rule, format, a...)
}

func (c *context) addErrMultiLoc(locs []errors.Location, rule string, format string, a ...interface{}) {
	c.errs = append(c.errs, &errors.QueryError{
		Message:   fmt.Sprintf(format, a...),
		Locations: locs,
		Rule:      rule,
	})
}

type opContext struct {
	*context
	ops []*ast.OperationDefinition
}

func newContext(s *ast.Schema, doc *ast.ExecutableDefinition, maxDepth int) *context {
	return &context{
		schema:           s,
		doc:              doc,
		opErrs:           make(map[*ast.OperationDefinition][]*errors.QueryError),
		usedVars:         make(map[*ast.OperationDefinition]varSet),
		fieldMap:         make(map[*ast.Field]fieldInfo),
		overlapValidated: make(map[selectionPair]struct{}),
		maxDepth:         maxDepth,
	}
}

func Validate(s *ast.Schema, doc *ast.ExecutableDefinition, variables map[string]interface{}, maxDepth int) []*errors.QueryError {
	c := newContext(s, doc, maxDepth)

	opNames := make(nameSet)
	fragUsedBy := make(map[*ast.FragmentDefinition][]*ast.OperationDefinition)
	for _, op := range doc.Operations {
		c.usedVars[op] = make(varSet)
		opc := &opContext{c, []*ast.OperationDefinition{op}}

		// Check if max depth is exceeded, if it's set. If max depth is exceeded,
		// don't continue to validate the document and exit early.
		if validateMaxDepth(opc, op.Selections, nil, 1) {
			return c.errs
		}

		if op.Name.Name == "" && len(doc.Operations) != 1 {
			c.addErr(op.Loc, "LoneAnonymousOperation", "This anonymous operation must be the only defined operation.")
		}
		if op.Name.Name != "" {
			validateName(c, opNames, op.Name, "UniqueOperationNames", "operation")
		}

		validateDirectives(opc, string(op.Type), op.Directives)

		varNames := make(nameSet)
		for _, v := range op.Vars {
			validateName(c, varNames, v.Name, "UniqueVariableNames", "variable")

			t := resolveType(c, v.Type)
			if !canBeInput(t) {
				c.addErr(v.TypeLoc, "VariablesAreInputTypes", "Variable %q cannot be non-input type %q.", "$"+v.Name.Name, t)
			}
			validateValue(opc, v, variables[v.Name.Name], t)

			if v.Default != nil {
				validateLiteral(opc, v.Default)

				if t != nil {
					if nn, ok := t.(*ast.NonNull); ok {
						c.addErr(v.Default.Location(), "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
					}

					if ok, reason := validateValueType(opc, v.Default, t); !ok {
						c.addErr(v.Default.Location(), "DefaultValuesOfCorrectType", "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name.Name, t, v.Default, reason)
					}
				}
			}
		}

		var entryPoint ast.NamedType
		switch op.Type {
		case query.Query:
			entryPoint = s.RootOperationTypes["query"]
		case query.Mutation:
			entryPoint = s.RootOperationTypes["mutation"]
		case query.Subscription:
			entryPoint = s.RootOperationTypes["subscription"]
		default:
			panic("unreachable")
		}

		validateSelectionSet(opc, op.Selections, entryPoint)

		fragUsed := make(map[*ast.FragmentDefinition]struct{})
		markUsedFragments(c, op.Selections, fragUsed)
		for frag := range fragUsed {
			fragUsedBy[frag] = append(fragUsedBy[frag], op)
		}
	}

	fragNames := make(nameSet)
	fragVisited := make(map[*ast.FragmentDefinition]struct{})
	for _, frag := range doc.Fragments {
		opc := &opContext{c, fragUsedBy[frag]}

		validateName(c, fragNames, frag.Name, "UniqueFragmentNames", "fragment")
		validateDirectives(opc, "FRAGMENT_DEFINITION", frag.Directives)

		t := unwrapType(resolveType(c, &frag.On))
		// continue even if t is nil
		if t != nil && !canBeFragment(t) {
			c.addErr(frag.On.Loc, "FragmentsOnCompositeTypes", "Fragment %q cannot condition on non composite type %q.", frag.Name.Name, t)
			continue
		}

		validateSelectionSet(opc, frag.Selections, t)

		if _, ok := fragVisited[frag]; !ok {
			detectFragmentCycle(c, frag.Selections, fragVisited, nil, map[string]int{frag.Name.Name: 0})
		}
	}

	for _, frag := range doc.Fragments {
		if len(fragUsedBy[frag]) == 0 {
			c.addErr(frag.Loc, "NoUnusedFragments", "Fragment %q is never used.", frag.Name.Name)
		}
	}

	for _, op := range doc.Operations {
		c.errs = append(c.errs, c.opErrs[op]...)

		opUsedVars := c.usedVars[op]
		for _, v := range op.Vars {
			if _, ok := opUsedVars[v]; !ok {
				opSuffix := ""
				if op.Name.Name != "" {
					opSuffix = fmt.Sprintf(" in operation %q", op.Name.Name)
				}
				c.addErr(v.Loc, "NoUnusedVariables", "Variable %q is never used%s.", "$"+v.Name.Name, opSuffix)
			}
		}
	}

	return c.errs
}

func validateValue(c *opContext, v *ast.InputValueDefinition, val interface{}, t ast.Type) {
	switch t := t.(type) {
	case *ast.NonNull:
		if val == nil {
			c.addErr(v.Loc, "VariablesOfCorrectType", "Variable \"%s\" has invalid value null.\nExpected type \"%s\", found null.", v.Name.Name, t)
			return
		}
		validateValue(c, v, val, t.OfType)
	case *ast.List:
		if val == nil {
			return
		}
		vv, ok := val.([]interface{})
		if !ok {
			// Input coercion rules allow single items without wrapping array
			validateValue(c, v, val, t.OfType)
			return
		}
		for _, elem := range vv {
			validateValue(c, v, elem, t.OfType)
		}
	case *ast.EnumTypeDefinition:
		if val == nil {
			return
		}
		e, ok := val.(string)
		if !ok {
			c.addErr(v.Loc, "VariablesOfCorrectType", "Variable \"%s\" has invalid type %T.\nExpected type \"%s\", found %v.", v.Name.Name, val, t, val)
			return
		}
		for _, option := range t.EnumValuesDefinition {
			if option.EnumValue == e {
				return
			}
		}
		c.addErr(v.Loc, "VariablesOfCorrectType", "Variable \"%s\" has invalid value %s.\nExpected type \"%s\", found %s.", v.Name.Name, e, t, e)
	case *ast.InputObject:
		if val == nil {
			return
		}
		in, ok := val.(map[string]interface{})
		if !ok {
			c.addErr(v.Loc, "VariablesOfCorrectType", "Variable \"%s\" has invalid type %T.\nExpected type \"%s\", found %s.", v.Name.Name, val, t, val)
			return
		}
		for _, f := range t.Values {
			fieldVal := in[f.Name.Name]
			validateValue(c, f, fieldVal, f.Type)
		}
	}
}

// validates the query doesn't go deeper than maxDepth (if set). Returns whether
// or not query validated max depth to avoid excessive recursion.
//
// The visited map is necessary to ensure that max depth validation does not get stuck in cyclical
// fragment spreads.
func validateMaxDepth(c *opContext, sels []ast.Selection, visited map[*ast.FragmentDefinition]struct{}, depth int) bool {
	// maxDepth checking is turned off when maxDepth is 0
	if c.maxDepth == 0 {
		return false
	}

	exceededMaxDepth := false
	if visited == nil {
		visited = map[*ast.FragmentDefinition]struct{}{}
	}

	for _, sel := range sels {
		switch sel := sel.(type) {
		case *ast.Field:
			if depth > c.maxDepth {
				exceededMaxDepth = true
				c.addErr(sel.Alias.Loc, "MaxDepthExceeded", "Field %q has depth %d that exceeds max depth %d", sel.Name.Name, depth, c.maxDepth)
				continue
			}
			exceededMaxDepth = exceededMaxDepth || validateMaxDepth(c, sel.SelectionSet, visited, depth+1)

		case *ast.InlineFragment:
			// Depth is not checked because inline fragments resolve to other fields which are checked.
			// Depth is not incremented because inline fragments have the same depth as neighboring fields
			exceededMaxDepth = exceededMaxDepth || validateMaxDepth(c, sel.Selections, visited, depth)
		case *ast.FragmentSpread:
			// Depth is not checked because fragments resolve to other fields which are checked.
			frag := c.doc.Fragments.Get(sel.Name.Name)
			if frag == nil {
				// In case of unknown fragment (invalid request), ignore max depth evaluation
				c.addErr(sel.Loc, "MaxDepthEvaluationError", "Unknown fragment %q. Unable to evaluate depth.", sel.Name.Name)
				continue
			}

			if _, ok := visited[frag]; ok {
				// we've already seen this fragment, don't check depth again.
				continue
			}
			visited[frag] = struct{}{}

			// Depth is not incremented because fragments have the same depth as surrounding fields
			exceededMaxDepth = exceededMaxDepth || validateMaxDepth(c, frag.Selections, visited, depth)
		}
	}

	return exceededMaxDepth
}

func validateSelectionSet(c *opContext, sels []ast.Selection, t ast.NamedType) {
	for _, sel := range sels {
		validateSelection(c, sel, t)
	}

	for i, a := range sels {
		for _, b := range sels[i+1:] {
			c.validateOverlap(a, b, nil, nil)
		}
	}
}

func validateSelection(c *opContext, sel ast.Selection, t ast.NamedType) {
	switch sel := sel.(type) {
	case *ast.Field:
		validateDirectives(c, "FIELD", sel.Directives)

		fieldName := sel.Name.Name
		var f *ast.FieldDefinition
		switch fieldName {
		case "__typename":
			f = &ast.FieldDefinition{
				Name: "__typename",
				Type: c.schema.Types["String"],
			}
		case "__schema":
			f = &ast.FieldDefinition{
				Name: "__schema",
				Type: c.schema.Types["__Schema"],
			}
		case "__type":
			f = &ast.FieldDefinition{
				Name: "__type",
				Arguments: ast.ArgumentsDefinition{
					&ast.InputValueDefinition{
						Name: ast.Ident{Name: "name"},
						Type: &ast.NonNull{OfType: c.schema.Types["String"]},
					},
				},
				Type: c.schema.Types["__Type"],
			}
		case "_service":
			f = &ast.FieldDefinition{
				Name: "_service",
				Type: c.schema.Types["_Service"],
			}
		default:
			f = fields(t).Get(fieldName)
			if f == nil && t != nil {
				suggestion := makeSuggestion("Did you mean", fields(t).Names(), fieldName)
				c.addErr(sel.Alias.Loc, "FieldsOnCorrectType", "Cannot query field %q on type %q.%s", fieldName, t, suggestion)
			}
		}
		c.fieldMap[sel] = fieldInfo{sf: f, parent: t}

		validateArgumentLiterals(c, sel.Arguments)
		if f != nil {
			validateArgumentTypes(c, sel.Arguments, f.Arguments, sel.Alias.Loc,
				func() string { return fmt.Sprintf("field %q of type %q", fieldName, t) },
				func() string { return fmt.Sprintf("Field %q", fieldName) },
			)
		}

		var ft ast.Type
		if f != nil {
			ft = f.Type
			sf := hasSubfields(ft)
			if sf && sel.SelectionSet == nil {
				c.addErr(sel.Alias.Loc, "ScalarLeafs", "Field %q of type %q must have a selection of subfields. Did you mean \"%s { ... }\"?", fieldName, ft, fieldName)
			}
			if !sf && sel.SelectionSet != nil {
				c.addErr(sel.SelectionSetLoc, "ScalarLeafs", "Field %q must not have a selection since type %q has no subfields.", fieldName, ft)
			}
		}
		if sel.SelectionSet != nil {
			validateSelectionSet(c, sel.SelectionSet, unwrapType(ft))
		}

	case *ast.InlineFragment:
		validateDirectives(c, "INLINE_FRAGMENT", sel.Directives)
		if sel.On.Name != "" {
			fragTyp := unwrapType(resolveType(c.context, &sel.On))
			if fragTyp != nil && !compatible(t, fragTyp) {
				c.addErr(sel.Loc, "PossibleFragmentSpreads", "Fragment cannot be spread here as objects of type %q can never be of type %q.", t, fragTyp)
			}
			t = fragTyp
			// continue even if t is nil
		}
		if t != nil && !canBeFragment(t) {
			c.addErr(sel.On.Loc, "FragmentsOnCompositeTypes", "Fragment cannot condition on non composite type %q.", t)
			return
		}
		validateSelectionSet(c, sel.Selections, unwrapType(t))

	case *ast.FragmentSpread:
		validateDirectives(c, "FRAGMENT_SPREAD", sel.Directives)
		frag := c.doc.Fragments.Get(sel.Name.Name)
		if frag == nil {
			c.addErr(sel.Name.Loc, "KnownFragmentNames", "Unknown fragment %q.", sel.Name.Name)
			return
		}
		fragTyp := c.schema.Types[frag.On.Name]
		if !compatible(t, fragTyp) {
			c.addErr(sel.Loc, "PossibleFragmentSpreads", "Fragment %q cannot be spread here as objects of type %q can never be of type %q.", frag.Name.Name, t, fragTyp)
		}

	default:
		panic("unreachable")
	}
}

func compatible(a, b ast.Type) bool {
	for _, pta := range possibleTypes(a) {
		for _, ptb := range possibleTypes(b) {
			if pta == ptb {
				return true
			}
		}
	}
	return false
}

func possibleTypes(t ast.Type) []*ast.ObjectTypeDefinition {
	switch t := t.(type) {
	case *ast.ObjectTypeDefinition:
		return []*ast.ObjectTypeDefinition{t}
	case *ast.InterfaceTypeDefinition:
		return t.PossibleTypes
	case *ast.Union:
		return t.UnionMemberTypes
	default:
		return nil
	}
}

func markUsedFragments(c *context, sels []ast.Selection, fragUsed map[*ast.FragmentDefinition]struct{}) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *ast.Field:
			if sel.SelectionSet != nil {
				markUsedFragments(c, sel.SelectionSet, fragUsed)
			}

		case *ast.InlineFragment:
			markUsedFragments(c, sel.Selections, fragUsed)

		case *ast.FragmentSpread:
			frag := c.doc.Fragments.Get(sel.Name.Name)
			if frag == nil {
				return
			}

			if _, ok := fragUsed[frag]; ok {
				continue
			}

			fragUsed[frag] = struct{}{}
			markUsedFragments(c, frag.Selections, fragUsed)

		default:
			panic("unreachable")
		}
	}
}

func detectFragmentCycle(c *context, sels []ast.Selection, fragVisited map[*ast.FragmentDefinition]struct{}, spreadPath []*ast.FragmentSpread, spreadPathIndex map[string]int) {
	for _, sel := range sels {
		detectFragmentCycleSel(c, sel, fragVisited, spreadPath, spreadPathIndex)
	}
}

func detectFragmentCycleSel(c *context, sel ast.Selection, fragVisited map[*ast.FragmentDefinition]struct{}, spreadPath []*ast.FragmentSpread, spreadPathIndex map[string]int) {
	switch sel := sel.(type) {
	case *ast.Field:
		if sel.SelectionSet != nil {
			detectFragmentCycle(c, sel.SelectionSet, fragVisited, spreadPath, spreadPathIndex)
		}

	case *ast.InlineFragment:
		detectFragmentCycle(c, sel.Selections, fragVisited, spreadPath, spreadPathIndex)

	case *ast.FragmentSpread:
		frag := c.doc.Fragments.Get(sel.Name.Name)
		if frag == nil {
			return
		}

		spreadPath = append(spreadPath, sel)
		if i, ok := spreadPathIndex[frag.Name.Name]; ok {
			cyclePath := spreadPath[i:]
			via := ""
			if len(cyclePath) > 1 {
				names := make([]string, len(cyclePath)-1)
				for i, frag := range cyclePath[:len(cyclePath)-1] {
					names[i] = frag.Name.Name
				}
				via = " via " + strings.Join(names, ", ")
			}

			locs := make([]errors.Location, len(cyclePath))
			for i, frag := range cyclePath {
				locs[i] = frag.Loc
			}
			c.addErrMultiLoc(locs, "NoFragmentCycles", "Cannot spread fragment %q within itself%s.", frag.Name.Name, via)
			return
		}

		if _, ok := fragVisited[frag]; ok {
			return
		}
		fragVisited[frag] = struct{}{}

		spreadPathIndex[frag.Name.Name] = len(spreadPath)
		detectFragmentCycle(c, frag.Selections, fragVisited, spreadPath, spreadPathIndex)
		delete(spreadPathIndex, frag.Name.Name)

	default:
		panic("unreachable")
	}
}

func (c *context) validateOverlap(a, b ast.Selection, reasons *[]string, locs *[]errors.Location) {
	if a == b {
		return
	}

	if _, ok := c.overlapValidated[selectionPair{a, b}]; ok {
		return
	}
	c.overlapValidated[selectionPair{a, b}] = struct{}{}
	c.overlapValidated[selectionPair{b, a}] = struct{}{}

	switch a := a.(type) {
	case *ast.Field:
		switch b := b.(type) {
		case *ast.Field:
			if b.Alias.Loc.Before(a.Alias.Loc) {
				a, b = b, a
			}
			if reasons2, locs2 := c.validateFieldOverlap(a, b); len(reasons2) != 0 {
				locs2 = append(locs2, a.Alias.Loc, b.Alias.Loc)
				if reasons == nil {
					c.addErrMultiLoc(locs2, "OverlappingFieldsCanBeMerged", "Fields %q conflict because %s. Use different aliases on the fields to fetch both if this was intentional.", a.Alias.Name, strings.Join(reasons2, " and "))
					return
				}
				for _, r := range reasons2 {
					*reasons = append(*reasons, fmt.Sprintf("subfields %q conflict because %s", a.Alias.Name, r))
				}
				*locs = append(*locs, locs2...)
			}

		case *ast.InlineFragment:
			for _, sel := range b.Selections {
				c.validateOverlap(a, sel, reasons, locs)
			}

		case *ast.FragmentSpread:
			if frag := c.doc.Fragments.Get(b.Name.Name); frag != nil {
				for _, sel := range frag.Selections {
					c.validateOverlap(a, sel, reasons, locs)
				}
			}

		default:
			panic("unreachable")
		}

	case *ast.InlineFragment:
		for _, sel := range a.Selections {
			c.validateOverlap(sel, b, reasons, locs)
		}

	case *ast.FragmentSpread:
		if frag := c.doc.Fragments.Get(a.Name.Name); frag != nil {
			for _, sel := range frag.Selections {
				c.validateOverlap(sel, b, reasons, locs)
			}
		}

	default:
		panic("unreachable")
	}
}

func (c *context) validateFieldOverlap(a, b *ast.Field) ([]string, []errors.Location) {
	if a.Alias.Name != b.Alias.Name {
		return nil, nil
	}

	if asf := c.fieldMap[a].sf; asf != nil {
		if bsf := c.fieldMap[b].sf; bsf != nil {
			if !typesCompatible(asf.Type, bsf.Type) {
				return []string{fmt.Sprintf("they return conflicting types %s and %s", asf.Type, bsf.Type)}, nil
			}
		}
	}

	at := c.fieldMap[a].parent
	bt := c.fieldMap[b].parent
	if at == nil || bt == nil || at == bt {
		if a.Name.Name != b.Name.Name {
			return []string{fmt.Sprintf("%s and %s are different fields", a.Name.Name, b.Name.Name)}, nil
		}

		if argumentsConflict(a.Arguments, b.Arguments) {
			return []string{"they have differing arguments"}, nil
		}
	}

	var reasons []string
	var locs []errors.Location
	for _, a2 := range a.SelectionSet {
		for _, b2 := range b.SelectionSet {
			c.validateOverlap(a2, b2, &reasons, &locs)
		}
	}
	return reasons, locs
}

func argumentsConflict(a, b ast.ArgumentList) bool {
	if len(a) != len(b) {
		return true
	}
	for _, argA := range a {
		valB, ok := b.Get(argA.Name.Name)
		if !ok || !reflect.DeepEqual(argA.Value.Deserialize(nil), valB.Deserialize(nil)) {
			return true
		}
	}
	return false
}

func fields(t ast.Type) ast.FieldsDefinition {
	switch t := t.(type) {
	case *ast.ObjectTypeDefinition:
		return t.Fields
	case *ast.InterfaceTypeDefinition:
		return t.Fields
	default:
		return nil
	}
}

func unwrapType(t ast.Type) ast.NamedType {
	if t == nil {
		return nil
	}
	for {
		switch t2 := t.(type) {
		case ast.NamedType:
			return t2
		case *ast.List:
			t = t2.OfType
		case *ast.NonNull:
			t = t2.OfType
		default:
			panic("unreachable")
		}
	}
}

func resolveType(c *context, t ast.Type) ast.Type {
	t2, err := common.ResolveType(t, c.schema.Resolve)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return t2
}

func validateDirectives(c *opContext, loc string, directives ast.DirectiveList) {
	directiveNames := make(nameSet)
	for _, d := range directives {
		dirName := d.Name.Name
		validateNameCustomMsg(c.context, directiveNames, d.Name, "UniqueDirectivesPerLocation", func() string {
			return fmt.Sprintf("The directive %q can only be used once at this location.", dirName)
		})

		validateArgumentLiterals(c, d.Arguments)

		dd, ok := c.schema.Directives[dirName]
		if !ok {
			c.addErr(d.Name.Loc, "KnownDirectives", "Unknown directive %q.", dirName)
			continue
		}

		locOK := false
		for _, allowedLoc := range dd.Locations {
			if loc == allowedLoc {
				locOK = true
				break
			}
		}
		if !locOK {
			c.addErr(d.Name.Loc, "KnownDirectives", "Directive %q may not be used on %s.", dirName, loc)
		}

		validateArgumentTypes(c, d.Arguments, dd.Arguments, d.Name.Loc,
			func() string { return fmt.Sprintf("directive %q", "@"+dirName) },
			func() string { return fmt.Sprintf("Directive %q", "@"+dirName) },
		)
	}
}

func validateName(c *context, set nameSet, name ast.Ident, rule string, kind string) {
	validateNameCustomMsg(c, set, name, rule, func() string {
		return fmt.Sprintf("There can be only one %s named %q.", kind, name.Name)
	})
}

func validateNameCustomMsg(c *context, set nameSet, name ast.Ident, rule string, msg func() string) {
	if loc, ok := set[name.Name]; ok {
		c.addErrMultiLoc([]errors.Location{loc, name.Loc}, rule, msg())
		return
	}
	set[name.Name] = name.Loc
}

func validateArgumentTypes(c *opContext, args ast.ArgumentList, argDecls ast.ArgumentsDefinition, loc errors.Location, owner1, owner2 func() string) {
	for _, selArg := range args {
		arg := argDecls.Get(selArg.Name.Name)
		if arg == nil {
			c.addErr(selArg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on %s.", selArg.Name.Name, owner1())
			continue
		}
		value := selArg.Value
		if ok, reason := validateValueType(c, value, arg.Type); !ok {
			c.addErr(value.Location(), "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, value, reason)
		}
	}
	for _, decl := range argDecls {
		if _, ok := decl.Type.(*ast.NonNull); ok {
			if _, ok := args.Get(decl.Name.Name); !ok {
				c.addErr(loc, "ProvidedNonNullArguments", "%s argument %q of type %q is required but not provided.", owner2(), decl.Name.Name, decl.Type)
			}
		}
	}
}

func validateArgumentLiterals(c *opContext, args ast.ArgumentList) {
	argNames := make(nameSet)
	for _, arg := range args {
		validateName(c.context, argNames, arg.Name, "UniqueArgumentNames", "argument")
		validateLiteral(c, arg.Value)
	}
}

func validateLiteral(c *opContext, l ast.Value) {
	switch l := l.(type) {
	case *ast.ObjectValue:
		fieldNames := make(nameSet)
		for _, f := range l.Fields {
			validateName(c.context, fieldNames, f.Name, "UniqueInputFieldNames", "input field")
			validateLiteral(c, f.Value)
		}
	case *ast.ListValue:
		for _, entry := range l.Values {
			validateLiteral(c, entry)
		}
	case *ast.Variable:
		for _, op := range c.ops {
			v := op.Vars.Get(l.Name)
			if v == nil {
				byOp := ""
				if op.Name.Name != "" {
					byOp = fmt.Sprintf(" by operation %q", op.Name.Name)
				}
				c.opErrs[op] = append(c.opErrs[op], &errors.QueryError{
					Message:   fmt.Sprintf("Variable %q is not defined%s.", "$"+l.Name, byOp),
					Locations: []errors.Location{l.Loc, op.Loc},
					Rule:      "NoUndefinedVariables",
				})
				continue
			}
			validateValueType(c, l, resolveType(c.context, v.Type))
			c.usedVars[op][v] = struct{}{}
		}
	}
}

func validateValueType(c *opContext, v ast.Value, t ast.Type) (bool, string) {
	if v, ok := v.(*ast.Variable); ok {
		for _, op := range c.ops {
			if v2 := op.Vars.Get(v.Name); v2 != nil {
				t2, err := common.ResolveType(v2.Type, c.schema.Resolve)
				if _, ok := t2.(*ast.NonNull); !ok && v2.Default != nil {
					t2 = &ast.NonNull{OfType: t2}
				}
				if err == nil && !typeCanBeUsedAs(t2, t) {
					c.addErrMultiLoc([]errors.Location{v2.Loc, v.Loc}, "VariablesInAllowedPosition", "Variable %q of type %q used in position expecting type %q.", "$"+v.Name, t2, t)
				}
			}
		}
		return true, ""
	}

	if nn, ok := t.(*ast.NonNull); ok {
		if isNull(v) {
			return false, fmt.Sprintf("Expected %q, found null.", t)
		}
		t = nn.OfType
	}
	if isNull(v) {
		return true, ""
	}

	switch t := t.(type) {
	case *ast.ScalarTypeDefinition, *ast.EnumTypeDefinition:
		if lit, ok := v.(*ast.PrimitiveValue); ok {
			if validateBasicLit(lit, t) {
				return true, ""
			}
			return false, fmt.Sprintf("Expected type %q, found %s.", t, v)
		}
		return true, ""

	case *ast.List:
		list, ok := v.(*ast.ListValue)
		if !ok {
			return validateValueType(c, v, t.OfType) // single value instead of list
		}
		for i, entry := range list.Values {
			if ok, reason := validateValueType(c, entry, t.OfType); !ok {
				return false, fmt.Sprintf("In element #%d: %s", i, reason)
			}
		}
		return true, ""

	case *ast.InputObject:
		v, ok := v.(*ast.ObjectValue)
		if !ok {
			return false, fmt.Sprintf("Expected %q, found not an object.", t)
		}
		for _, f := range v.Fields {
			name := f.Name.Name
			iv := t.Values.Get(name)
			if iv == nil {
				return false, fmt.Sprintf("In field %q: Unknown field.", name)
			}
			if ok, reason := validateValueType(c, f.Value, iv.Type); !ok {
				return false, fmt.Sprintf("In field %q: %s", name, reason)
			}
		}
		for _, iv := range t.Values {
			found := false
			for _, f := range v.Fields {
				if f.Name.Name == iv.Name.Name {
					found = true
					break
				}
			}
			if !found {
				if _, ok := iv.Type.(*ast.NonNull); ok && iv.Default == nil {
					return false, fmt.Sprintf("In field %q: Expected %q, found null.", iv.Name.Name, iv.Type)
				}
			}
		}
		return true, ""
	}

	return false, fmt.Sprintf("Expected type %q, found %s.", t, v)
}

func validateBasicLit(v *ast.PrimitiveValue, t ast.Type) bool {
	switch t := t.(type) {
	case *ast.ScalarTypeDefinition:
		switch t.Name {
		case "Int":
			if v.Type != scanner.Int {
				return false
			}
			return validateBuiltInScalar(v.Text, "Int")
		case "Float":
			return (v.Type == scanner.Int || v.Type == scanner.Float) && validateBuiltInScalar(v.Text, "Float")
		case "String":
			return v.Type == scanner.String && validateBuiltInScalar(v.Text, "String")
		case "Boolean":
			return v.Type == scanner.Ident && validateBuiltInScalar(v.Text, "Boolean")
		case "ID":
			return (v.Type == scanner.Int && validateBuiltInScalar(v.Text, "Int")) || (v.Type == scanner.String && validateBuiltInScalar(v.Text, "String"))
		default:
			//TODO: Type-check against expected type by Unmarshalling
			return true
		}

	case *ast.EnumTypeDefinition:
		if v.Type != scanner.Ident {
			return false
		}
		for _, option := range t.EnumValuesDefinition {
			if option.EnumValue == v.Text {
				return true
			}
		}
		return false
	}

	return false
}

func validateBuiltInScalar(v string, n string) bool {
	switch n {
	case "Int":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false
		}
		return f >= math.MinInt32 && f <= math.MaxInt32
	case "Float":
		f, fe := strconv.ParseFloat(v, 64)
		return fe == nil && f <= math.MaxFloat64
	case "String":
		vl := len(v)
		return vl >= 2 && v[0] == '"' && v[vl-1] == '"'
	case "Boolean":
		return v == "true" || v == "false"
	default:
		return false
	}
}

func canBeFragment(t ast.Type) bool {
	switch t.(type) {
	case *ast.ObjectTypeDefinition, *ast.InterfaceTypeDefinition, *ast.Union:
		return true
	default:
		return false
	}
}

func canBeInput(t ast.Type) bool {
	switch t := t.(type) {
	case *ast.InputObject, *ast.ScalarTypeDefinition, *ast.EnumTypeDefinition:
		return true
	case *ast.List:
		return canBeInput(t.OfType)
	case *ast.NonNull:
		return canBeInput(t.OfType)
	default:
		return false
	}
}

func hasSubfields(t ast.Type) bool {
	switch t := t.(type) {
	case *ast.ObjectTypeDefinition, *ast.InterfaceTypeDefinition, *ast.Union:
		return true
	case *ast.List:
		return hasSubfields(t.OfType)
	case *ast.NonNull:
		return hasSubfields(t.OfType)
	default:
		return false
	}
}

func isLeaf(t ast.Type) bool {
	switch t.(type) {
	case *ast.ScalarTypeDefinition, *ast.EnumTypeDefinition:
		return true
	default:
		return false
	}
}

func isNull(lit interface{}) bool {
	_, ok := lit.(*ast.NullValue)
	return ok
}

func typesCompatible(a, b ast.Type) bool {
	al, aIsList := a.(*ast.List)
	bl, bIsList := b.(*ast.List)
	if aIsList || bIsList {
		return aIsList && bIsList && typesCompatible(al.OfType, bl.OfType)
	}

	ann, aIsNN := a.(*ast.NonNull)
	bnn, bIsNN := b.(*ast.NonNull)
	if aIsNN || bIsNN {
		return aIsNN && bIsNN && typesCompatible(ann.OfType, bnn.OfType)
	}

	if isLeaf(a) || isLeaf(b) {
		return a == b
	}

	return true
}

func typeCanBeUsedAs(t, as ast.Type) bool {
	nnT, okT := t.(*ast.NonNull)
	if okT {
		t = nnT.OfType
	}

	nnAs, okAs := as.(*ast.NonNull)
	if okAs {
		as = nnAs.OfType
		if !okT {
			return false // nullable can not be used as non-null
		}
	}

	if t == as {
		return true
	}

	if lT, ok := t.(*ast.List); ok {
		if lAs, ok := as.(*ast.List); ok {
			return typeCanBeUsedAs(lT.OfType, lAs.OfType)
		}
	}
	return false
}
