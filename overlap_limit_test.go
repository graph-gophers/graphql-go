package graphql_test

import (
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
)

const overlapLimitSchemaSDL = `schema { query: Query } type Query { root: Thing } type Thing { id: ID! name: String }`

type overlapLimitRoot struct{}

func (r *overlapLimitRoot) Root() *overlapThing { return &overlapThing{} }

type overlapThing struct{}

func (t *overlapThing) ID() graphql.ID { return graphql.ID("1") }
func (t *overlapThing) Name() *string  { s := "n"; return &s }

// TestOverlapValidationLimit exercises overlap pair limit behaviors (exceeded, unlimited, not reached)
// in a single table-driven test for clarity and concision.
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
			name:           "unlimited_no_option",
			opts:           []graphql.SchemaOpt{}, // no option => unlimited
			query:          `query { root { id id id id id } }`,
			expectLimitErr: false,
			comment:        "no limit option supplied, cap disabled",
		},
		{
			name:           "not_reached",
			opts:           []graphql.SchemaOpt{graphql.OverlapValidationLimit(100)}, // 3 id fields -> combinations C(3,2)=3 < 100 => no error
			query:          `query { root { id id id } }`,
			expectLimitErr: false,
			comment:        "below configured limit",
		},
	}

	for _, tc := range tests {
		tc := tc
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
