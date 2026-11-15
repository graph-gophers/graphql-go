package graphql_test

import (
	"context"
	"math"
	"strings"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
)

const overlapLimitSchemaSDL = `schema { query: Query } type Query { root: Thing } type Thing { id: ID! name: String self: Thing! }`

type overlapLimitRoot struct{}

func (r *overlapLimitRoot) Root() *overlapThing { return &overlapThing{} }

type overlapThing struct{}

func (t *overlapThing) ID() graphql.ID      { return graphql.ID("1") }
func (t *overlapThing) Name() *string       { s := "n"; return &s }
func (t *overlapThing) Self() *overlapThing { return t }

func TestOverlapValidationLimit(t *testing.T) {
	t.Parallel()

	hasLimitErr := func(errs []*gqlerrors.QueryError) bool {
		for _, e := range errs {
			if e.Rule == "OverlapValidationLimitExceeded" {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name           string
		opts           []graphql.SchemaOpt
		query          string
		expectLimitErr bool
		comment        string
	}{
		{
			name:           "exceeded",
			opts:           []graphql.SchemaOpt{graphql.OverlapValidationLimit(3)}, // 5 repeated id fields -> combinations C(5,2)=10 > 3 => early abort
			query:          `query { root { id id id id id } }`,
			expectLimitErr: true,
			comment:        "should trigger OverlapValidationLimitExceeded",
		},
		{
			name:           "default_not_reached",
			query:          `query { root { id id id id id } }`,
			expectLimitErr: false,
			comment:        "default limit not reached",
		},
		{
			name:           "at_limit",
			opts:           []graphql.SchemaOpt{graphql.OverlapValidationLimit(3)},
			query:          `query { root { id id id } }`,
			expectLimitErr: false,
			comment:        "exactly at configured limit",
		},
		{
			name:           "shared_across_operations",
			opts:           []graphql.SchemaOpt{graphql.OverlapValidationLimit(1)},
			query:          `query A { root { id id } } query B { root { id id } }`,
			expectLimitErr: true,
			comment:        "limit applies to the whole document",
		},
		{
			name:           "fragment_definition",
			opts:           []graphql.SchemaOpt{graphql.OverlapValidationLimit(1)},
			query:          `query { root { ...F } } fragment F on Thing { id id id }`,
			expectLimitErr: true,
			comment:        "fragment validation respects the limit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := graphql.MustParseSchema(overlapLimitSchemaSDL, &overlapLimitRoot{}, tc.opts...)
			errs := schema.Validate(tc.query)
			gotLimitErr := hasLimitErr(errs)
			if gotLimitErr != tc.expectLimitErr {
				t.Fatalf("%s: expected limitErr=%v, got %v (errs=%#v)", tc.comment, tc.expectLimitErr, gotLimitErr, errs)
			}
		})
	}
}

func TestOverlapValidationDefaultLimit(t *testing.T) {
	belowLimit := `query { root { ` + strings.Repeat("id ", 447) + `} }`
	aboveLimit := `query { root { ` + strings.Repeat("id ", 448) + `} }`

	for _, tc := range []struct {
		name        string
		opts        []graphql.SchemaOpt
		expectError bool
	}{
		{name: "default", expectError: true},
		{name: "zero", opts: []graphql.SchemaOpt{graphql.OverlapValidationLimit(0)}, expectError: true},
		{name: "negative", opts: []graphql.SchemaOpt{graphql.OverlapValidationLimit(-1)}, expectError: true},
		{name: "large_limit", opts: []graphql.SchemaOpt{graphql.OverlapValidationLimit(math.MaxInt)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := graphql.MustParseSchema(overlapLimitSchemaSDL, &overlapLimitRoot{}, tc.opts...)
			if errs := s.Validate(belowLimit); len(errs) != 0 {
				t.Fatalf("below limit: %v", errs)
			}
			errs := s.Validate(aboveLimit)
			if tc.expectError {
				if len(errs) != 1 || errs[0].Rule != "OverlapValidationLimitExceeded" {
					t.Fatalf("expected overlap limit error, got %v", errs)
				}
			} else if len(errs) != 0 {
				t.Fatalf("unexpected validation errors: %v", errs)
			}
		})
	}

	t.Run("exec_before_selection_limit", func(t *testing.T) {
		s := graphql.MustParseSchema(overlapLimitSchemaSDL, &overlapLimitRoot{}, graphql.MaxSelections(1))
		q := `query { root { ` + strings.Repeat("x: self { id } ", 4000) + `} }`
		res := s.Exec(context.Background(), q, "", nil)
		if len(res.Errors) != 1 || res.Errors[0].Rule != "OverlapValidationLimitExceeded" {
			t.Fatalf("expected overlap limit error, got %v", res.Errors)
		}
	})

	t.Run("clone_resets_to_default", func(t *testing.T) {
		s := graphql.MustParseSchema(overlapLimitSchemaSDL, &overlapLimitRoot{}, graphql.OverlapValidationLimit(1))
		clone := s.MustClone(&overlapLimitRoot{}, graphql.OverlapValidationLimit(0))
		if errs := clone.Validate(belowLimit); len(errs) != 0 {
			t.Fatalf("clone did not use default limit: %v", errs)
		}
		if errs := clone.Validate(aboveLimit); len(errs) != 1 || errs[0].Rule != "OverlapValidationLimitExceeded" {
			t.Fatalf("expected overlap limit error, got %v", errs)
		}
	})
}
