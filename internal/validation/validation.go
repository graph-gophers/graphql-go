package validation

import (
	"fmt"
	"math"
	"reflect"
	"slices"
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

type nameSet map[string][]errors.Location

type fieldInfo struct {
	sf     *ast.FieldDefinition
	parent ast.NamedType
}

type valueTypeIssue struct {
	loc     errors.Location
	message string
}

type context struct {
	schema               *ast.Schema
	doc                  *ast.ExecutableDefinition
	errs                 []*errors.QueryError
	opErrs               map[*ast.OperationDefinition][]*errors.QueryError
	usedVars             map[*ast.OperationDefinition]varSet
	fieldMap             map[*ast.Field]fieldInfo
	overlapValidated     map[selectionPair]bool
	maxDepth             int
	overlapPairLimit     int
	overlapPairsObserved int
	overlapLimitHit      bool
}

func (c *context) addErr(loc errors.Location, rule string, format string, a ...any) {
	c.addErrMultiLoc([]errors.Location{loc}, rule, format, a...)
}

func (c *context) addErrMultiLoc(locs []errors.Location, rule string, format string, a ...any) {
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

func newContext(s *ast.Schema, doc *ast.ExecutableDefinition, maxDepth int, overlapPairLimit int) *context {
	return &context{
		schema:           s,
		doc:              doc,
		opErrs:           make(map[*ast.OperationDefinition][]*errors.QueryError),
		usedVars:         make(map[*ast.OperationDefinition]varSet),
		fieldMap:         make(map[*ast.Field]fieldInfo),
		overlapValidated: make(map[selectionPair]bool),
		maxDepth:         maxDepth,
		overlapPairLimit: overlapPairLimit,
	}
}

func Validate(s *ast.Schema, doc *ast.ExecutableDefinition, variables map[string]any, maxDepth int, overlapPairLimit int) []*errors.QueryError {
	c := newContext(s, doc, maxDepth, overlapPairLimit)

	opNames := make(nameSet, len(doc.Operations))
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
			c.addErr(op.Loc, "LoneAnonymousOperationRule", "This anonymous operation must be the only defined operation.")
		}

		if n := op.Name.Name; n != "" {
			opNames[n] = append(opNames[n], op.Name.Loc)
		}

		varNames := make(nameSet, len(op.Vars))
		for _, v := range op.Vars {
			varNames[v.Name.Name] = append(varNames[v.Name.Name], v.Name.Loc)

			validateDirectives(opc, "VARIABLE_DEFINITION", v.Directives)

			t := resolveType(c, v.Type)
			if !canBeInput(t) {
				c.addErr(v.TypeLoc, "VariablesAreInputTypesRule", "Variable %q cannot be non-input type %q.", "$"+v.Name.Name, t)
			}
			validateValue(opc, v, variables[v.Name.Name], t)

			if v.Default != nil {
				validateLiteral(opc, v.Default)

				if t != nil {
					if nn, ok := t.(*ast.NonNull); ok {
						c.addErr(v.Default.Location(), "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
					}

					if inputType := unwrapInputObjectType(t); inputType != nil {
						if obj, ok := v.Default.(*ast.ObjectValue); ok {
							issues := collectInputObjectValueIssues(opc, obj, inputType)
							if len(issues) > 0 {
								for _, issue := range issues {
									c.addErr(issue.loc, "ValuesOfCorrectTypeRule", "%s", issue.message)
								}
								continue
							}
						}
					}

					if ok, errLoc, reason := validateValueType(opc, v.Default, t); !ok {
						c.addErr(errLoc, "ValuesOfCorrectTypeRule", "%s", reason)
					}
				}
			}
		}

		validateDirectives(opc, string(op.Type), op.Directives)

		for n, locs := range varNames {
			validateName(c, locs, n, "UniqueVariableNamesRule", "variable")
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

	for n, locs := range opNames {
		validateName(c, locs, n, "UniqueOperationNamesRule", "operation")
	}

	fragNames := make(nameSet, len(doc.Fragments))
	fragVisited := make(map[*ast.FragmentDefinition]struct{})
	for _, frag := range doc.Fragments {
		opc := &opContext{c, fragUsedBy[frag]}

		fragNames[frag.Name.Name] = append(fragNames[frag.Name.Name], frag.Name.Loc)

		validateDirectives(opc, "FRAGMENT_DEFINITION", frag.Directives)

		t := unwrapType(resolveType(c, &frag.On))
		// continue even if t is nil
		if t != nil && !canBeFragment(t) {
			c.addErr(frag.On.Loc, "FragmentsOnCompositeTypesRule", "Fragment %q cannot condition on non composite type %q.", frag.Name.Name, t)
			continue
		}

		validateSelectionSet(opc, frag.Selections, t)

		if _, ok := fragVisited[frag]; !ok {
			detectFragmentCycle(c, frag.Selections, fragVisited, nil, map[string]int{frag.Name.Name: 0})
		}
	}

	for n, locs := range fragNames {
		validateName(c, locs, n, "UniqueFragmentNamesRule", "fragment")
	}

	for _, frag := range doc.Fragments {
		if len(fragUsedBy[frag]) == 0 {
			c.addErr(frag.Loc, "NoUnusedFragmentsRule", "Fragment %q is never used.", frag.Name.Name)
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
				c.addErr(v.Loc, "NoUnusedVariablesRule", "Variable %q is never used%s.", "$"+v.Name.Name, opSuffix)
			}
		}
	}

	return c.errs
}

func validateValue(c *opContext, v *ast.InputValueDefinition, val any, t ast.Type) {
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
		vv, ok := val.([]any)
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
		in, ok := val.(map[string]any)
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
	if len(sels) == 0 {
		return
	}

	// First pass: validate each selection and bucket fields by response name (alias or name).
	fieldGroups := make(map[string][]ast.Selection)
	var fragments []ast.Selection // fragment spreads & inline fragments
	for _, sel := range sels {
		if c.overlapLimitHit {
			return
		}
		validateSelection(c, sel, t)
		switch s := sel.(type) {
		case *ast.Field:
			name := fieldResponseName(s)
			fieldGroups[name] = append(fieldGroups[name], sel)
		default:
			fragments = append(fragments, sel)
		}
	}

	// Compare fields only within same response name group (was O(n^2) across all fields previously).
	for _, group := range fieldGroups {
		if c.overlapLimitHit {
			break
		}
		if len(group) < 2 {
			continue
		}
		for i, a := range group {
			if c.overlapLimitHit {
				break
			}
			for _, b := range group[i+1:] {
				if c.overlapLimitHit {
					break
				}
				c.validateOverlap(a, b, nil, nil, false)
			}
		}
	}

	// Fragments can introduce any field names, so we must compare them with all fields and each other.
	if len(fragments) > 0 && !c.overlapLimitHit {
		// Flatten fields for fragment comparison.
		var allFields []ast.Selection
		for _, group := range fieldGroups {
			allFields = append(allFields, group...)
		}
		for i, fa := range fragments {
			if c.overlapLimitHit {
				break
			}
			// Compare fragment with fields. If fragment has only direct field selections,
			// compare by matching response name groups; otherwise conservatively compare
			// against all fields.
			compareSelectionAgainstFields(c, fa, fieldGroups, allFields)
			// Compare fragment with following fragments
			for _, fb := range fragments[i+1:] {
				if c.overlapLimitHit {
					break
				}
				c.validateOverlap(fa, fb, nil, nil, false)
			}
		}
	}
}

func fieldResponseName(f *ast.Field) string {
	if f.Alias.Name != "" {
		return f.Alias.Name
	}
	return f.Name.Name
}

func selectionTopLevelFieldNames(c *context, sel ast.Selection) (map[string]struct{}, bool) {
	names := make(map[string]struct{})

	getNames := func(selections []ast.Selection) (map[string]struct{}, bool) {
		for _, child := range selections {
			field, ok := child.(*ast.Field)
			if !ok {
				return nil, true
			}
			names[fieldResponseName(field)] = struct{}{}
		}
		return names, false
	}

	switch s := sel.(type) {
	case *ast.InlineFragment:
		return getNames(s.Selections)
	case *ast.FragmentSpread:
		frag := c.doc.Fragments.Get(s.Name.Name)
		if frag == nil {
			return names, false
		}
		return getNames(frag.Selections)
	default:
		return nil, true
	}
}

func compareSelectionAgainstFields(c *opContext, sel ast.Selection, fieldGroups map[string][]ast.Selection, allFields []ast.Selection) {
	fieldNames, exhaustive := selectionTopLevelFieldNames(c.context, sel)
	if exhaustive {
		for _, fld := range allFields {
			if c.overlapLimitHit {
				return
			}
			c.validateOverlap(fld, sel, nil, nil, false)
		}
		return
	}

	for name := range fieldNames {
		for _, fld := range fieldGroups[name] {
			if c.overlapLimitHit {
				return
			}
			c.validateOverlap(fld, sel, nil, nil, false)
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
		default:
			f = fields(t).Get(fieldName)
			if f == nil && t != nil {
				suggestion := makeSuggestion("Did you mean", fields(t).Names(), fieldName)
				c.addErr(sel.Alias.Loc, "FieldsOnCorrectTypeRule", "Cannot query field %q on type %q.%s", fieldName, t, suggestion)
			}
		}
		c.fieldMap[sel] = fieldInfo{sf: f, parent: t}

		validateArgumentLiterals(c, sel.Arguments)
		if f != nil {
			if reason, ok := deprecatedReason(f.Directives); ok && t != nil {
				c.addErr(sel.Name.Loc, "NoDeprecatedCustomRule", "The field %s.%s is deprecated. %s", t.TypeName(), fieldName, reason)
			}

			validateArgumentTypes(c, sel.Arguments, f.Arguments, sel.Alias.Loc,
				func() string { return fmt.Sprintf(`field "%s.%s"`, t, fieldName) },
				func() string { return fmt.Sprintf("Field %q", fieldName) },
			)

			if t != nil {
				for _, selArg := range sel.Arguments {
					argDecl := f.Arguments.Get(selArg.Name.Name)
					if argDecl == nil {
						continue
					}
					if reason, ok := deprecatedReason(argDecl.Directives); ok {
						c.addErr(selArg.Name.Loc, "NoDeprecatedCustomRule", "Field %q argument %q is deprecated. %s", t.TypeName()+"."+fieldName, selArg.Name.Name, reason)
					}
				}

				for _, directive := range sel.Directives {
					directiveDef, ok := c.schema.Directives[directive.Name.Name]
					if !ok {
						continue
					}
					for _, selArg := range directive.Arguments {
						argDecl := directiveDef.Arguments.Get(selArg.Name.Name)
						if argDecl == nil {
							continue
						}
						if reason, ok := deprecatedReason(argDecl.Directives); ok {
							c.addErr(selArg.Name.Loc, "NoDeprecatedCustomRule", "Directive %q argument %q is deprecated. %s", "@"+directive.Name.Name, selArg.Name.Name, reason)
						}
					}
				}
			}
		}

		var ft ast.Type
		if f != nil {
			ft = f.Type
			sf := hasSubfields(ft)
			if sf && sel.SelectionSet == nil {
				c.addErr(sel.Alias.Loc, "ScalarLeafsRule", "Field %q of type %q must have a selection of subfields. Did you mean \"%s { ... }\"?", fieldName, ft, fieldName)
			}
			if !sf && sel.SelectionSet != nil {
				c.addErr(sel.SelectionSetLoc, "ScalarLeafsRule", "Field %q must not have a selection since type %q has no subfields.", fieldName, ft)
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
				c.addErr(sel.Loc, "PossibleFragmentSpreadsRule", "Fragment cannot be spread here as objects of type %q can never be of type %q.", t, fragTyp)
			}
			t = fragTyp
			// continue even if t is nil
		}
		if t != nil && !canBeFragment(t) {
			c.addErr(sel.On.Loc, "FragmentsOnCompositeTypesRule", "Fragment cannot condition on non composite type %q.", t)
			return
		}
		validateSelectionSet(c, sel.Selections, unwrapType(t))

	case *ast.FragmentSpread:
		validateDirectives(c, "FRAGMENT_SPREAD", sel.Directives)
		frag := c.doc.Fragments.Get(sel.Name.Name)
		if frag == nil {
			c.addErr(sel.Name.Loc, "KnownFragmentNamesRule", "Unknown fragment %q.", sel.Name.Name)
			return
		}
		fragTyp := c.schema.Types[frag.On.Name]
		if !compatible(t, fragTyp) {
			c.addErr(sel.Loc, "PossibleFragmentSpreadsRule", "Fragment %q cannot be spread here as objects of type %q can never be of type %q.", frag.Name.Name, t, fragTyp)
		}

	default:
		panic("unreachable")
	}
}

func compatible(a, b ast.Type) bool {
	for _, pta := range possibleTypes(a) {
		if slices.Contains(possibleTypes(b), pta) {
			return true
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
					names[i] = fmt.Sprintf("%q", frag.Name.Name)
				}
				via = " via " + strings.Join(names, ", ")
			}

			locs := make([]errors.Location, len(cyclePath))
			for i, frag := range cyclePath {
				locs[i] = frag.Loc
			}
			c.addErrMultiLoc(locs, "NoFragmentCyclesRule", "Cannot spread fragment %q within itself%s.", frag.Name.Name, via)
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

func (c *context) validateOverlap(a, b ast.Selection, reasons *[]string, locs *[]errors.Location, parentMutuallyExclusive bool) {
	if a == b {
		return
	}

	// Optimisation 1: store only one direction of the pair to halve memory and lookups.
	pa := reflect.ValueOf(a).Pointer()
	pb := reflect.ValueOf(b).Pointer()
	key := selectionPair{a: a, b: b}
	if pb < pa { // canonical ordering for key only
		key = selectionPair{a: b, b: a}
	}
	if existing, ok := c.overlapValidated[key]; ok {
		// Mutually exclusive comparisons are always safe to skip once this pair was seen.
		if parentMutuallyExclusive {
			return
		}
		if !existing {
			return
		}
	}
	c.overlapValidated[key] = parentMutuallyExclusive

	if c.overlapPairLimit > 0 && !c.overlapLimitHit {
		c.overlapPairsObserved++
		if c.overlapPairsObserved > c.overlapPairLimit {
			c.overlapLimitHit = true
			// determine a representative location for error reporting
			var loc errors.Location
			switch sel := a.(type) {
			case *ast.Field:
				loc = sel.Alias.Loc
			case *ast.InlineFragment:
				loc = sel.Loc
			case *ast.FragmentSpread:
				loc = sel.Loc
			default:
				// leave zero value
			}
			c.addErr(loc, "OverlapValidationLimitExceeded", "Overlapping field validation aborted after examining %d pairs (limit %d). Consider restructuring the query or increasing the limit.", c.overlapPairsObserved-1, c.overlapPairLimit)
			return
		}
	}

	switch a := a.(type) {
	case *ast.Field:
		switch b := b.(type) {
		case *ast.Field:
			if reasons2, locs2 := c.validateFieldOverlap(a, b, parentMutuallyExclusive); len(reasons2) != 0 {
				locs2 = append(locs2, a.Alias.Loc, b.Alias.Loc)
				if reasons == nil {
					c.addErrMultiLoc(locs2, "OverlappingFieldsCanBeMergedRule", "Fields %q conflict because %s. Use different aliases on the fields to fetch both if this was intentional.", a.Alias.Name, strings.Join(reasons2, " and "))
					return
				}
				for _, r := range reasons2 {
					*reasons = append(*reasons, fmt.Sprintf("subfields %q conflict because %s", a.Alias.Name, r))
				}
				*locs = append(*locs, locs2...)
			}

		case *ast.InlineFragment:
			for _, sel := range b.Selections {
				c.validateOverlap(a, sel, reasons, locs, parentMutuallyExclusive)
			}

		case *ast.FragmentSpread:
			if frag := c.doc.Fragments.Get(b.Name.Name); frag != nil {
				for _, sel := range frag.Selections {
					c.validateOverlap(a, sel, reasons, locs, parentMutuallyExclusive)
				}
			}

		default:
			panic("unreachable")
		}

	case *ast.InlineFragment:
		for _, sel := range a.Selections {
			c.validateOverlap(sel, b, reasons, locs, parentMutuallyExclusive)
		}

	case *ast.FragmentSpread:
		if frag := c.doc.Fragments.Get(a.Name.Name); frag != nil {
			for _, sel := range frag.Selections {
				c.validateOverlap(sel, b, reasons, locs, parentMutuallyExclusive)
			}
		}

	default:
		panic("unreachable")
	}
}

func (c *context) validateFieldOverlap(a, b *ast.Field, parentMutuallyExclusive bool) ([]string, []errors.Location) {
	if a.Alias.Name != b.Alias.Name {
		return nil, nil
	}

	if asf := c.fieldMap[a].sf; asf != nil {
		if bsf := c.fieldMap[b].sf; bsf != nil {
			if !typesCompatible(asf.Type, bsf.Type) {
				return []string{fmt.Sprintf("they return conflicting types %q and %q", asf.Type, bsf.Type)}, nil
			}
		}
	}

	at := c.fieldMap[a].parent
	bt := c.fieldMap[b].parent
	areMutuallyExclusive := parentMutuallyExclusive || mutuallyExclusiveParents(at, bt)
	if !areMutuallyExclusive {
		if a.Name.Name != b.Name.Name {
			return []string{fmt.Sprintf("%q and %q are different fields", a.Name.Name, b.Name.Name)}, nil
		}

		if argumentsConflict(a.Arguments, b.Arguments) {
			return []string{"they have differing arguments"}, nil
		}
	}

	var reasons []string
	var locs []errors.Location

	// Fast-path: if either side has no subselections we are done.
	if len(a.SelectionSet) == 0 || len(b.SelectionSet) == 0 {
		return nil, nil
	}

	// Optimisation 2: avoid O(m*n) cartesian product for large sibling lists with mostly
	// distinct response names (common & exploitable for DoS). Instead, index B's field
	// selections by response name (alias/name). For each field in A we only compare
	// against fields in B with the same response name plus all fragment spreads / inline
	// fragments (which can expand to any field names and must be compared exhaustively).
	bFieldIndex := make(map[string][]ast.Selection, len(b.SelectionSet))
	var bNonField []ast.Selection
	for _, bs := range b.SelectionSet {
		if f, ok := bs.(*ast.Field); ok {
			name := fieldResponseName(f)
			bFieldIndex[name] = append(bFieldIndex[name], bs)
			continue
		}
		bNonField = append(bNonField, bs)
	}

	for _, a2 := range a.SelectionSet {
		if af, ok := a2.(*ast.Field); ok {
			name := fieldResponseName(af)
			// Compare only against same-name fields + all non-field selections.
			if matches := bFieldIndex[name]; len(matches) != 0 {
				for _, bMatch := range matches {
					c.validateOverlap(a2, bMatch, &reasons, &locs, areMutuallyExclusive)
				}
			}
			for _, bnf := range bNonField {
				c.validateOverlap(a2, bnf, &reasons, &locs, areMutuallyExclusive)
			}
			continue
		}
		// For fragments / inline fragments we still need to compare against every selection in B.
		for _, b2 := range b.SelectionSet {
			c.validateOverlap(a2, b2, &reasons, &locs, areMutuallyExclusive)
		}
	}

	return reasons, locs
}

func mutuallyExclusiveParents(a, b ast.NamedType) bool {
	if a == nil || b == nil || a == b {
		return false
	}
	_, aIsObject := a.(*ast.ObjectTypeDefinition)
	_, bIsObject := b.(*ast.ObjectTypeDefinition)
	return aIsObject && bIsObject
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
	directiveNames := make(nameSet, len(directives))
	for _, d := range directives {
		dirName := d.Name.Name

		directiveNames[dirName] = append(directiveNames[dirName], d.Name.Loc)

		validateArgumentLiterals(c, d.Arguments)

		dd, ok := c.schema.Directives[dirName]
		if !ok {
			c.addErr(d.Name.Loc, "KnownDirectivesRule", "Unknown directive %q.", "@"+dirName)
			continue
		}

		locOK := slices.Contains(dd.Locations, loc)
		if !locOK {
			c.addErr(d.Name.Loc, "KnownDirectivesRule", "Directive %q may not be used on %s.", "@"+dirName, loc)
		}

		validateArgumentTypes(c, d.Arguments, dd.Arguments, d.Name.Loc,
			func() string { return fmt.Sprintf("directive %q", "@"+dirName) },
			func() string { return fmt.Sprintf("Directive %q", "@"+dirName) },
		)

		if loc != "FIELD" {
			for _, selArg := range d.Arguments {
				argDecl := dd.Arguments.Get(selArg.Name.Name)
				if argDecl == nil {
					continue
				}
				if reason, ok := deprecatedReason(argDecl.Directives); ok {
					c.addErr(selArg.Name.Loc, "NoDeprecatedCustomRule", "Directive %q argument %q is deprecated. %s", "@"+dirName, selArg.Name.Name, reason)
				}
			}
		}
	}

	// Iterating in the declared order, rather than using the directiveNames ordering which is random
	for _, d := range directives {
		n := d.Name.Name

		ds := directiveNames[n]
		if len(ds) <= 1 {
			continue
		}

		dd, ok := c.schema.Directives[n]
		if !ok {
			// Invalid directive will have been flagged already
			continue
		}

		if dd.Repeatable {
			continue
		}

		for _, loc := range ds[1:] {
			// Duplicate directive errors are inconsistent with the behaviour for other types in graphql-js
			// Instead of reporting a single error with all locations, errors are reported for each duplicate after the first declaration
			// with the original location, and the duplicate. Behaviour is replicated here, as we use those tests to validate the implementation
			validateNameCustomMsg(c.context, []errors.Location{ds[0], loc}, "UniqueDirectivesPerLocationRule", func() string {
				return fmt.Sprintf("The directive %q can only be used once at this location.", "@"+n)
			})
		}

		// drop the name from the set to prevent the same errors being re-added for duplicates
		delete(directiveNames, n)
	}
}

func validateName(c *context, locs []errors.Location, name string, rule string, kind string) {
	validateNameCustomMsg(c, locs, rule, func() string {
		if kind == "variable" {
			return fmt.Sprintf("There can be only one %s named %q.", kind, "$"+name)
		}

		return fmt.Sprintf("There can be only one %s named %q.", kind, name)
	})
}

func validateNameCustomMsg(c *context, locs []errors.Location, rule string, msg func() string) {
	if len(locs) > 1 {
		c.addErrMultiLoc(locs, rule, "%s", msg())
		return
	}
}

func validateArgumentTypes(c *opContext, args ast.ArgumentList, argDecls ast.ArgumentsDefinition, loc errors.Location, owner1, owner2 func() string) {
	for _, selArg := range args {
		arg := argDecls.Get(selArg.Name.Name)
		if arg == nil {
			suggestion := makeSuggestion("Did you mean", argDecls.Names(), selArg.Name.Name)
			c.addErr(selArg.Name.Loc, "KnownArgumentNamesRule", "Unknown argument %q on %s.%s", selArg.Name.Name, owner1(), suggestion)
			continue
		}
		value := selArg.Value
		if ok, errLoc, reason := validateValueType(c, value, arg.Type); !ok {
			c.addErr(errLoc, "ValuesOfCorrectTypeRule", "%s", reason)
		}
	}
	for _, decl := range argDecls {
		if _, ok := decl.Type.(*ast.NonNull); ok {
			if _, ok := args.Get(decl.Name.Name); !ok {
				if decl.Default != nil {
					continue
				}

				c.addErr(loc, "ProvidedRequiredArgumentsRule", "%s argument %q of type %q is required, but it was not provided.", owner2(), decl.Name.Name, decl.Type)
			}
		}
	}
}

func validateArgumentLiterals(c *opContext, args ast.ArgumentList) {
	argNames := make(nameSet, len(args))
	for _, arg := range args {
		validateLiteral(c, arg.Value)

		argNames[arg.Name.Name] = append(argNames[arg.Name.Name], arg.Name.Loc)
	}

	for n, locs := range argNames {
		validateName(c.context, locs, n, "UniqueArgumentNamesRule", "argument")
	}
}

func validateLiteral(c *opContext, l ast.Value) {
	switch l := l.(type) {
	case *ast.ObjectValue:
		fieldNames := make(nameSet, len(l.Fields))
		for _, f := range l.Fields {
			fieldNames[f.Name.Name] = append(fieldNames[f.Name.Name], f.Name.Loc)
			validateLiteral(c, f.Value)
		}

		for n, locs := range fieldNames {
			if len(locs) <= 1 {
				continue
			}

			// Similar to for directives, duplicates here aren't all reported together but using an error for each duplicate
			for _, loc := range locs[1:] {
				validateName(c.context, []errors.Location{locs[0], loc}, n, "UniqueInputFieldNamesRule", "input field")
			}
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
					Rule:      "NoUndefinedVariablesRule",
				})
				continue
			}
			_, _, _ = validateValueType(c, l, resolveType(c.context, v.Type))
			c.usedVars[op][v] = struct{}{}
		}
	}
}

func validateValueType(c *opContext, v ast.Value, t ast.Type) (bool, errors.Location, string) {
	if v, ok := v.(*ast.Variable); ok {
		for _, op := range c.ops {
			if v2 := op.Vars.Get(v.Name); v2 != nil {
				t2, err := common.ResolveType(v2.Type, c.schema.Resolve)
				if _, ok := t2.(*ast.NonNull); !ok && v2.Default != nil {
					if _, ok := v2.Default.(*ast.NullValue); !ok {
						t2 = &ast.NonNull{OfType: t2}
					}
				}
				if err == nil && !typeCanBeUsedAs(t2, t) {
					c.addErrMultiLoc([]errors.Location{v2.Loc, v.Loc}, "VariablesInAllowedPositionRule", "Variable %q of type %q used in position expecting type %q.", "$"+v.Name, t2, t)
				}
			}
		}
		return true, errors.Location{}, ""
	}

	if nn, ok := t.(*ast.NonNull); ok {
		if isNull(v) {
			return false, v.Location(), fmt.Sprintf("Expected value of type %q, found null.", t)
		}
		t = nn.OfType
	}
	if isNull(v) {
		return true, errors.Location{}, ""
	}

	switch t := t.(type) {
	case *ast.ScalarTypeDefinition, *ast.EnumTypeDefinition:
		lit, ok := v.(*ast.PrimitiveValue)
		if !ok {
			return true, errors.Location{}, ""
		}

		isValid, reason := validateBasicLit(lit, t)
		if !isValid {
			return false, lit.Location(), reason
		}

		enumType, isEnum := t.(*ast.EnumTypeDefinition)
		if !isEnum || lit.Type != scanner.Ident {
			return true, errors.Location{}, ""
		}

		for _, option := range enumType.EnumValuesDefinition {
			if option.EnumValue != lit.Text {
				continue
			}
			if depReason, deprecated := deprecatedReason(option.Directives); deprecated {
				c.addErr(lit.Location(), "NoDeprecatedCustomRule", "The enum value %q is deprecated. %s", enumType.Name+"."+option.EnumValue, depReason)
			}
			break
		}
		return true, errors.Location{}, ""

	case *ast.List:
		list, ok := v.(*ast.ListValue)
		if !ok {
			return validateValueType(c, v, t.OfType) // single value instead of list
		}
		for _, entry := range list.Values {
			if ok, errLoc, reason := validateValueType(c, entry, t.OfType); !ok {
				return false, errLoc, reason
			}
		}
		return true, errors.Location{}, ""

	case *ast.InputObject:
		orig := v
		v, ok := v.(*ast.ObjectValue)
		if !ok {
			return false, orig.Location(), fmt.Sprintf("Expected value of type %q, found %s.", t, orig)
		}

		providedFields := make(map[string]struct{}, len(v.Fields))
		for _, f := range v.Fields {
			name := f.Name.Name
			providedFields[name] = struct{}{}
			iv := t.Values.Get(name)
			if iv == nil {
				suggestion := makeSuggestion("Did you mean", t.Values.Names(), name)
				return false, f.Name.Loc, fmt.Sprintf("Field %q is not defined by type %q.%s", name, t.Name, suggestion)
			}
			if depReason, deprecated := deprecatedReason(iv.Directives); deprecated {
				c.addErr(f.Name.Loc, "NoDeprecatedCustomRule", "The input field %s.%s is deprecated. %s", t.Name, iv.Name.Name, depReason)
			}
			if ok, errLoc, reason := validateValueType(c, f.Value, iv.Type); !ok {
				return false, errLoc, reason
			}
		}
		for _, iv := range t.Values {
			if _, found := providedFields[iv.Name.Name]; found {
				continue
			}
			if _, ok := iv.Type.(*ast.NonNull); ok && iv.Default == nil {
				return false, v.Location(), fmt.Sprintf("Field %q of required type %q was not provided.", t.Name+"."+iv.Name.Name, iv.Type)
			}
		}

		// Validate @oneOf constraint: exactly one non-null field must be provided
		if t.Directives.Get("oneOf") != nil {
			if len(v.Fields) != 1 {
				c.addErr(v.Location(), "ValuesOfCorrectTypeRule", "OneOf Input Object %q must specify exactly one key.", t.Name)
				return true, errors.Location{}, ""
			}

			f := v.Fields[0]

			// Check for explicit null values
			if _, isNull := f.Value.(*ast.NullValue); isNull {
				c.addErr(v.Location(), "ValuesOfCorrectTypeRule", "Field %q must be non-null.", t.Name+"."+f.Name.Name)
				return true, errors.Location{}, ""
			}

			// Check for nullable variables
			if varRef, isVar := f.Value.(*ast.Variable); isVar {
				for _, op := range c.ops {
					if varDef := op.Vars.Get(varRef.Name); varDef != nil {
						if _, ok := varDef.Type.(*ast.NonNull); !ok {
							varType := varDef.Type
							if resolved := resolveType(c.context, varDef.Type); resolved != nil {
								varType = resolved
							}
							c.addErrMultiLoc([]errors.Location{varDef.Loc, varRef.Loc}, "VariablesInAllowedPositionRule", "Variable %q is of type %q but must be non-nullable to be used for OneOf Input Object %q.", "$"+varRef.Name, varType, t.Name)
							return true, errors.Location{}, ""
						}
					}
				}
			}
		}

		return true, errors.Location{}, ""
	}

	return false, v.Location(), fmt.Sprintf("Expected type %q, found %s.", t, v)
}

func deprecatedReason(directives ast.DirectiveList) (string, bool) {
	deprecated := directives.Get("deprecated")
	if deprecated == nil {
		return "", false
	}
	arg, ok := deprecated.Arguments.Get("reason")
	if !ok {
		return "No longer supported", true
	}
	reason, ok := arg.Deserialize(nil).(string)
	if !ok || reason == "" {
		return "No longer supported", true
	}
	return reason, true
}

func unwrapInputObjectType(t ast.Type) *ast.InputObject {
	for {
		switch tt := t.(type) {
		case *ast.NonNull:
			t = tt.OfType
		case *ast.InputObject:
			return tt
		default:
			return nil
		}
	}
}

func collectInputObjectValueIssues(c *opContext, obj *ast.ObjectValue, inputType *ast.InputObject) []valueTypeIssue {
	issues := make([]valueTypeIssue, 0)
	for _, field := range obj.Fields {
		decl := inputType.Values.Get(field.Name.Name)
		if decl == nil {
			continue
		}
		if ok, errLoc, reason := validateValueType(c, field.Value, decl.Type); !ok {
			issues = append(issues, valueTypeIssue{loc: errLoc, message: reason})
		}
	}
	return issues
}

func validateBasicLit(v *ast.PrimitiveValue, t ast.Type) (bool, string) {
	switch t := t.(type) {
	case *ast.ScalarTypeDefinition:
		switch t.Name {
		case "Int":
			if v.Type == scanner.Int {
				if validateBuiltInScalar(v.Text, "Int") {
					return true, ""
				}
				return false, fmt.Sprintf("Int cannot represent non 32-bit signed integer value: %s", v)
			}
			return false, fmt.Sprintf("Int cannot represent non-integer value: %s", v)
		case "Float":
			if v.Type == scanner.Int || v.Type == scanner.Float {
				if validateBuiltInScalar(v.Text, "Float") {
					return true, ""
				}
			}
			return false, fmt.Sprintf("Float cannot represent non numeric value: %s", v)
		case "String":
			if v.Type == scanner.String && validateBuiltInScalar(v.Text, "String") {
				return true, ""
			}
			return false, fmt.Sprintf("String cannot represent a non string value: %s", v)
		case "Boolean":
			if v.Type == scanner.Ident && validateBuiltInScalar(v.Text, "Boolean") {
				return true, ""
			}
			return false, fmt.Sprintf("Boolean cannot represent a non boolean value: %s", v)
		case "ID":
			if (v.Type == scanner.Int && validateBuiltInScalar(v.Text, "Int")) || (v.Type == scanner.String && validateBuiltInScalar(v.Text, "String")) {
				return true, ""
			}
			return false, fmt.Sprintf("ID cannot represent a non-string and non-integer value: %s", v)
		default:
			// TODO: Type-check against expected type by Unmarshalling
			return true, ""
		}

	case *ast.EnumTypeDefinition:
		values := make([]string, 0, len(t.EnumValuesDefinition))
		for _, option := range t.EnumValuesDefinition {
			values = append(values, option.EnumValue)
		}

		if v.Type == scanner.Ident {
			if v.Text == "true" || v.Text == "false" {
				return false, fmt.Sprintf("Enum %q cannot represent non-enum value: %s.", t.Name, v)
			}

			for _, option := range t.EnumValuesDefinition {
				if option.EnumValue == v.Text {
					return true, ""
				}
			}

			suggestion := makeSuggestion("Did you mean the enum value", values, v.Text)
			if suggestion == "" {
				for _, option := range values {
					if strings.EqualFold(option, v.Text) {
						suggestion = fmt.Sprintf(" Did you mean the enum value %q?", option)
						break
					}
				}
			}
			if suggestion != "" {
				return false, fmt.Sprintf("Value %q does not exist in %q enum.%s", v.Text, t.Name, suggestion)
			}
			return false, fmt.Sprintf("Value %q does not exist in %q enum.", v.Text, t.Name)
		}

		candidate := strings.Trim(v.Text, "\"")
		suggestion := makeSuggestion("Did you mean the enum value", values, candidate)
		if suggestion != "" {
			return false, fmt.Sprintf("Enum %q cannot represent non-enum value: %s.%s", t.Name, v, suggestion)
		}
		return false, fmt.Sprintf("Enum %q cannot represent non-enum value: %s.", t.Name, v)

	default:
		return false, fmt.Sprintf("Expected type %q, found %s.", t, v)
	}
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
	case nil:
		return true
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

func isNull(lit any) bool {
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
