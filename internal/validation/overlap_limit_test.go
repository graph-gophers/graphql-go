package validation

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/internal/query"
)

func TestOverlapLimitStopsNestedComparisons(t *testing.T) {
	var children strings.Builder
	for i := range 400 {
		fmt.Fprintf(&children, "f%d: id ", i)
	}

	doc, err := query.Parse(fmt.Sprintf("{ root { x: child { %s } x: child { %s } } }", children.String(), children.String()))
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Operations[0].Selections[0].(*ast.Field)
	c := newContext(nil, doc, 0, 1, false)
	c.validateOverlap(root.SelectionSet[0], root.SelectionSet[1], nil, nil, false)

	if !c.overlapLimitHit || c.overlapPairsObserved != 1 || len(c.overlapValidated) != 1 {
		t.Fatalf("limit hit=%v, comparisons=%d, cached pairs=%d", c.overlapLimitHit, c.overlapPairsObserved, len(c.overlapValidated))
	}
	if len(c.errs) != 1 || c.errs[0].Rule != "OverlapValidationLimitExceeded" {
		t.Fatalf("expected overlap limit error, got %v", c.errs)
	}
}

func TestOverlapLimitDoesNotOverflow(t *testing.T) {
	doc, err := query.Parse("{ id }")
	if err != nil {
		t.Fatal(err)
	}
	sel := doc.Operations[0].Selections[0]
	c := newContext(nil, doc, 0, math.MaxInt, false)
	c.overlapPairsObserved = math.MaxInt - 1
	c.validateOverlap(sel, sel, nil, nil, false)
	c.validateOverlap(sel, sel, nil, nil, false)
	if !c.overlapLimitHit || c.overlapPairsObserved != math.MaxInt {
		t.Fatalf("limit hit=%v, comparisons=%d", c.overlapLimitHit, c.overlapPairsObserved)
	}
}

func TestOverlapLimitCountsRepeatedComparisons(t *testing.T) {
	doc, err := query.Parse("{ id id }")
	if err != nil {
		t.Fatal(err)
	}
	a, b := doc.Operations[0].Selections[0], doc.Operations[0].Selections[1]
	for _, tc := range []struct {
		name string
		same bool
	}{
		{name: "cached_pair"},
		{name: "same_selection", same: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newContext(nil, doc, 0, 1, false)
			second := b
			if tc.same {
				second = a
			}
			c.validateOverlap(a, second, nil, nil, false)
			c.validateOverlap(a, second, nil, nil, false)
			if !c.overlapLimitHit || c.overlapPairsObserved != 1 {
				t.Fatalf("limit hit=%v, comparisons=%d", c.overlapLimitHit, c.overlapPairsObserved)
			}
		})
	}
}
