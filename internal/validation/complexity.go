package validation

import (
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
)

type ComplexityEstimator interface {
	DoEstimate(c *opContext, sels []query.Selection) bool
}

type SimpleEstimator struct {
	MaxComplexity int
}

func (e SimpleEstimator) DoEstimate(c *opContext, sels []query.Selection) bool {
	if e.MaxComplexity == 0 {
		return false
	}

	complexity := e.doSimpleEstimate(c, sels)
	if complexity > e.MaxComplexity {
		return true
	}

	return false
}

func (e SimpleEstimator) doSimpleEstimate(c *opContext, sels []query.Selection) int {
	complexity := 0

	for _, sel := range sels {
		var loc errors.Location
		switch sel := sel.(type) {
		case *query.Field:
			loc = sel.Alias.Loc
			complexity += e.doSimpleEstimate(c, sel.Selections) + 1
		case *query.InlineFragment:
			loc = sel.Loc
			complexity += e.doSimpleEstimate(c, sel.Selections)
		case *query.FragmentSpread:
			frag := c.doc.Fragments.Get(sel.Name.Name)
			if frag == nil {
				c.addErr(sel.Loc, "MaxComplexityEvaluationError", "Unknown fragment %q. Unable to evaluate complexity.", sel.Name.Name)
				continue
			}
			loc = frag.Loc
			complexity += e.doSimpleEstimate(c, frag.Selections)
		}

		if complexity > e.MaxComplexity {
			c.addErr(loc, "MaxComplexityExceeded",
				"The query exceeds the maximum complexity of %d. Actual complexity is %d.", e.MaxComplexity, complexity)

			return complexity
		}
	}

	return complexity
}

type RecursionEstimator struct {
	MaxDepth int
}

func (e RecursionEstimator) DoEstimate(c *opContext, sels []query.Selection) bool {
	if e.MaxDepth == 0 {
		return false
	}

	return e.doRecursivelyVisitSelections(c, sels, map[string]int{}, getEntryPoint(c.schema, c.ops[0]))
}

type visitedSels map[string]int

func (s visitedSels) copy() visitedSels {
	newSels := visitedSels{}
	for index, value := range s {
		newSels[index] = value
	}

	return newSels
}

func (e RecursionEstimator) doRecursivelyVisitSelections(
	c *opContext, sels []query.Selection, visited visitedSels, t schema.NamedType) bool {

	fields := fields(t)

	exceeded := false

	for _, sel := range sels {
		switch sel := sel.(type) {
		case *query.Field:
			fieldName := sel.Name.Name
			switch fieldName {
			case "__typename", "__schema", "__type":
				continue
			default:
				if sel.Selections == nil {
					continue
				}

				if f := fields.Get(fieldName); f != nil {
					v := visited.copy()

					if depth, ok := v[f.Type.String()]; ok {
						v[f.Type.String()] = depth + 1
					} else {
						v[f.Type.String()] = 1
					}

					currentDepth := v[f.Type.String()]
					if currentDepth > e.MaxDepth {
						c.addErr(sel.Alias.Loc, "MaxDepthRecursionExceeded",
							"The query exceeds the maximum depth recursion of %d. Actual is %d.",
							e.MaxDepth, currentDepth)

						return true
					}

					exceeded = e.doRecursivelyVisitSelections(c, sel.Selections, v, unwrapType(f.Type))
				}
			}
		case *query.InlineFragment:
			exceeded = e.doRecursivelyVisitSelections(c, sel.Selections, visited, unwrapType(t))
		case *query.FragmentSpread:
			if frag := c.doc.Fragments.Get(sel.Name.Name); frag != nil {
				exceeded = e.doRecursivelyVisitSelections(c, frag.Selections, visited, c.schema.Types[frag.On.Name])
			}
		}
	}

	return exceeded
}
