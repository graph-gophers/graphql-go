package graphql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/ast"
	graphqlerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

type directiveVisitorResolver struct{}

func (r *directiveVisitorResolver) Hello() string {
	return "visitor"
}

func (r *directiveVisitorResolver) A() string {
	return "visitor-a"
}

func (r *directiveVisitorResolver) B() string {
	return "visitor-b"
}

func (r *directiveVisitorResolver) Items(args struct{ PageSize int32 }) string {
	return "visitor"
}

// validateDirective is an implementation of the DirectiveVisitor interface
type validateDirective struct {
	handler func(ctx context.Context, args struct{ Pattern string }, d graphql.DirectiveContext) error
}

func (v *validateDirective) Name() string {
	return "validate"
}

func (v *validateDirective) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args struct{ Pattern string }
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}
	return v.handler(ctx, args, d)
}

// TestDirectiveVisitors tests the new generic directive visitor API
func TestDirectiveVisitors(t *testing.T) {
	t.Parallel()

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				directive @validate(pattern: String!) on FIELD_DEFINITION

				type Query {
					hello: String! @validate(pattern: ".*")
				}
			`, &directiveVisitorResolver{}, graphql.DirectiveVisitors(&validateDirective{
				handler: func(ctx context.Context, args struct{ Pattern string }, d graphql.DirectiveContext) error {
					return nil
				},
			})),
			Query:          `{ hello }`,
			ExpectedResult: `{"hello":"visitor"}`,
		},
		{
			Schema: graphql.MustParseSchema(`
				directive @validate(pattern: String!) on FIELD_DEFINITION

				type Query {
					hello: String! @validate(pattern: "xyz")
				}
			`, &directiveVisitorResolver{},
				graphql.DirectiveVisitors(&validateDirective{
					handler: func(ctx context.Context, args struct{ Pattern string }, d graphql.DirectiveContext) error {
						return fmt.Errorf("validation failed")
					},
				})),
			Query: `{ hello }`,
			ExpectedErrors: []*graphqlerrors.QueryError{
				{
					Message: "validation failed", Path: []any{"hello"},
				},
			},
		},
	})
}

type costDirArgs struct {
	Weight     int32
	Multiplier graphql.NullString
}

// costDirectiveWithContext is an implementation of the DirectiveVisitor interface
type costDirectiveWithContext struct {
	handler func(ctx context.Context, args costDirArgs, d graphql.DirectiveContext) error
}

func (c *costDirectiveWithContext) Name() string {
	return "cost"
}

func (c *costDirectiveWithContext) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args costDirArgs
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}
	return c.handler(ctx, args, d)
}

// TestDirectiveVisitorsWithContext tests context-aware visitors with nullable args.
func TestDirectiveVisitorsWithContext(t *testing.T) {
	t.Parallel()

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: graphql.MustParseSchema(`
				directive @cost(weight: Int!, multiplier: String) on FIELD_DEFINITION

				type Query {
					items(pageSize: Int!): String! @cost(weight: 2, multiplier: "pageSize")
				}
			`, &directiveVisitorResolver{},
				graphql.DirectiveVisitors(&costDirectiveWithContext{
					handler: func(ctx context.Context, args costDirArgs, d graphql.DirectiveContext) error {
						multiplier := int32(1)
						if args.Multiplier.Value != nil {
							if err := d.FieldArg(*args.Multiplier.Value, &multiplier); err != nil {
								return err
							}
						}
						if args.Weight*multiplier > 5 {
							return fmt.Errorf("total cost %d exceeds budget", args.Weight*multiplier)
						}
						return nil
					},
				})),
			Query:          `{ items(pageSize: 3) }`,
			ExpectedErrors: []*graphqlerrors.QueryError{{Message: "total cost 6 exceeds budget", Path: []any{"items"}}},
		},
	})
}

type preExecHookResolver struct {
	calls int
}

func (r *preExecHookResolver) A() string {
	r.calls++
	return "a"
}

func (r *preExecHookResolver) B() string {
	r.calls++
	return "b"
}

type complexityTotalKey struct{}

// costDirectiveForPreExec is an implementation of the DirectiveVisitor interface
type costDirectiveForPreExec struct {
	handler func(ctx context.Context, args struct{ Weight int32 }, d graphql.DirectiveContext) error
}

func (c *costDirectiveForPreExec) Name() string {
	return "cost"
}

func (c *costDirectiveForPreExec) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args struct{ Weight int32 }
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}
	return c.handler(ctx, args, d)
}

func TestPreExecHookWithDirectiveVisitors(t *testing.T) {
	t.Parallel()

	resolver := &preExecHookResolver{}
	schema := graphql.MustParseSchema(`
		directive @cost(weight: Int!) on FIELD_DEFINITION

		type Query {
			a: String! @cost(weight: 3)
			b: String! @cost(weight: 4)
		}
	`, resolver,
		graphql.DirectiveVisitors(&costDirectiveForPreExec{
			handler: func(ctx context.Context, args struct{ Weight int32 }, d graphql.DirectiveContext) error {
				total, _ := ctx.Value(complexityTotalKey{}).(*int32)
				if total != nil {
					*total += args.Weight
				}
				return nil
			},
		}),
		graphql.PreExecHook(func(ctx context.Context, _ *ast.ExecutableDefinition, _ *ast.OperationDefinition, _ map[string]any) error {
			total, _ := ctx.Value(complexityTotalKey{}).(*int32)
			if total != nil && *total > 5 {
				return fmt.Errorf("total cost %d exceeds budget", *total)
			}
			return nil
		}),
	)

	var total int32
	ctx := context.WithValue(context.Background(), complexityTotalKey{}, &total)
	resp := schema.Exec(ctx, `{ a b }`, "", nil)

	if len(resp.Errors) != 1 || resp.Errors[0].Message != "total cost 7 exceeds budget" {
		t.Fatalf("unexpected errors: %+v", resp.Errors)
	}
	if resolver.calls != 0 {
		t.Fatalf("pre-exec hook should stop execution, got %d resolver calls", resolver.calls)
	}
}

type directiveVisitorCallListKey struct{}

type recordingDirective struct {
	name  string
	label string
}

func (v *recordingDirective) Name() string {
	return v.name
}

func (v *recordingDirective) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args struct{ Pattern string }
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}

	calls, _ := ctx.Value(directiveVisitorCallListKey{}).(*[]string)
	if calls != nil {
		*calls = append(*calls, v.label+":"+args.Pattern)
	}
	return nil
}

func TestDirectiveVisitors_DuplicateNamesRejected(t *testing.T) {
	t.Parallel()

	_, err := graphql.ParseSchema(`
		directive @validate(pattern: String!) on FIELD_DEFINITION

		type Query {
			hello: String! @validate(pattern: "ok")
		}
	`, &directiveVisitorResolver{},
		graphql.DirectiveVisitors(
			&recordingDirective{name: "validate", label: "first"},
			&recordingDirective{name: "validate", label: "second"},
		),
	)
	if err == nil {
		t.Fatal("expected duplicate directive visitor registration to fail")
	}
	if got, want := err.Error(), `directive visitor "validate" is already registered`; got != want {
		t.Fatalf("unexpected error: got %q want %q", got, want)
	}
}

func TestDirectiveVisitors_CloneDuplicateNamesRejected(t *testing.T) {
	t.Parallel()

	base := graphql.MustParseSchema(`
		directive @validate(pattern: String!) on FIELD_DEFINITION

		type Query {
			hello: String! @validate(pattern: "clone")
		}
	`, &directiveVisitorResolver{},
		graphql.DirectiveVisitors(&recordingDirective{name: "validate", label: "base"}),
	)

	_, err := base.Clone(
		&directiveVisitorResolver{},
		graphql.DirectiveVisitors(&recordingDirective{name: "validate", label: "clone"}),
	)
	if err == nil {
		t.Fatal("expected clone with duplicate directive visitor to fail")
	}
	if got, want := err.Error(), `directive visitor "validate" is already registered`; got != want {
		t.Fatalf("unexpected error: got %q want %q", got, want)
	}
}

func TestDirectiveVisitors_DistinctNamesDispatch(t *testing.T) {
	t.Parallel()

	schema := graphql.MustParseSchema(`
		directive @validateA(pattern: String!) on FIELD_DEFINITION
		directive @validateB(pattern: String!) on FIELD_DEFINITION

		type Query {
			hello: String! @validateA(pattern: "first") @validateB(pattern: "second")
		}
	`, &directiveVisitorResolver{},
		graphql.DirectiveVisitors(
			&recordingDirective{name: "validateA", label: "a"},
			&recordingDirective{name: "validateB", label: "b"},
		),
	)

	var calls []string
	ctx := context.WithValue(context.Background(), directiveVisitorCallListKey{}, &calls)
	resp := schema.Exec(ctx, `{ hello }`, "", nil)
	if len(resp.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", resp.Errors)
	}
	if len(calls) != 2 || calls[0] != "a:first" || calls[1] != "b:second" {
		t.Fatalf("unexpected visitor calls: %+v", calls)
	}
}
