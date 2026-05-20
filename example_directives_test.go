package graphql_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/ast"
)

type authRoleKey struct{}

type authResolver struct {
	calls int
}

func (r *authResolver) Secret() string {
	r.calls++
	return "classified"
}

type authDirective struct{}

func (a *authDirective) Name() string {
	return "auth"
}

type authDirectiveArgs struct {
	Role string
}

func (a *authDirective) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args authDirectiveArgs
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}

	role, _ := ctx.Value(authRoleKey{}).(string)
	requiredRole := args.Role
	if role != requiredRole {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func ExampleDirectiveVisitors_auth() {
	resolver := &authResolver{}
	sdl := `
		enum Role {
			ADMIN
			MEMBER
		}

		directive @auth(role: Role!) on FIELD_DEFINITION

		type Query {
			secret: String! @auth(role: ADMIN)
		}
	`

	dirs := []graphql.SchemaOpt{
		graphql.DirectiveVisitors(&authDirective{}),
	}

	schema := graphql.MustParseSchema(sdl, resolver, dirs...)

	adminCtx := context.WithValue(context.Background(), authRoleKey{}, "ADMIN")
	admin := schema.Exec(adminCtx, `{ secret }`, "", nil)
	fmt.Println("admin:", string(admin.Data))
	fmt.Println("admin calls:", resolver.calls)

	resolver.calls = 0
	memberCtx := context.WithValue(context.Background(), authRoleKey{}, "MEMBER")
	member := schema.Exec(memberCtx, `{ secret }`, "", nil)
	fmt.Printf("member errors: %+v\n", member.Errors)
	fmt.Println("member calls:", resolver.calls)

	// Output:
	// admin: {"secret":"classified"}
	// admin calls: 1
	// member errors: [graphql: forbidden]
	// member calls: 0
}

type costContextKey struct{}

type costResolver struct {
	calls atomic.Int32
}

func (r *costResolver) Greet() string {
	r.calls.Add(1)
	return "hello"
}

func (r *costResolver) Items(ctx context.Context, args struct{ PageSize int32 }) ([]string, error) {
	r.calls.Add(1)
	if args.PageSize < 1 || args.PageSize > 100 {
		return nil, fmt.Errorf("pageSize must be between 1 and 100")
	}
	var res []string
	for range args.PageSize {
		res = append(res, "gopher")
	}
	return res, nil
}

type costDirective struct{}

func (c *costDirective) Name() string {
	return "cost"
}

func (c *costDirective) CheckThreshold(threshold uint) graphql.PreExecHookFunc {
	return func(ctx context.Context, _ *ast.ExecutableDefinition, _ *ast.OperationDefinition, _ map[string]any) error {
		if c.TotalCost(ctx) > threshold {
			return fmt.Errorf("query cost %d exceeds threshold %d", c.TotalCost(ctx), threshold)
		}
		return nil
	}
}

func (c *costDirective) Add(ctx context.Context, cost uint) {
	total, _ := ctx.Value(costContextKey{}).(*uint)
	if total != nil {
		*total += cost
	}
}

func (c *costDirective) TotalCost(ctx context.Context) uint {
	total, _ := ctx.Value(costContextKey{}).(*uint)
	if total == nil {
		return 0
	}
	return *total
}

type costDirectiveArgs struct {
	Weight     int32
	Multiplier graphql.NullString
}

func (c *costDirective) Visit(ctx context.Context, d graphql.DirectiveContext) error {
	var args costDirectiveArgs
	if err := d.DecodeArgs(&args); err != nil {
		return err
	}

	var multiplier int32 = 1
	if args.Multiplier.Value != nil { // multiplier is optional
		arg := *args.Multiplier.Value
		if err := d.FieldArg(arg, &multiplier); err != nil {
			return fmt.Errorf("decode multiplier %q: %s", arg, err)
		}
	}

	c.Add(ctx, uint(args.Weight)*uint(multiplier))
	return nil
}

func ExampleDirectiveVisitors_costMultiplier() {
	resolver := &costResolver{}
	complexity := &costDirective{}
	sdl := `
		directive @cost(weight: Int!, multiplier: String) on FIELD_DEFINITION

		type Query {
			greet: String! @cost(weight: 1)
			items(pageSize: Int!): [String!]! @cost(weight: 1, multiplier: "pageSize")
		}
	`

	opts := []graphql.SchemaOpt{
		graphql.DirectiveVisitors(complexity),
		graphql.PreExecHook(complexity.CheckThreshold(10)),
	}

	schema := graphql.MustParseSchema(sdl, resolver, opts...)

	var allowedTotal uint
	allowedCtx := context.WithValue(context.Background(), costContextKey{}, &allowedTotal)
	allowed := schema.Exec(allowedCtx, `{ greet items(pageSize: 5) }`, "", nil)
	fmt.Println("allowed:", string(allowed.Data))
	fmt.Printf("allowed errors: %+v\n", allowed.Errors)
	fmt.Println("allowed calls:", resolver.calls.Load())

	resolver.calls.Store(0)
	var blockedTotal uint
	blockedCtx := context.WithValue(context.Background(), costContextKey{}, &blockedTotal)
	blocked := schema.Exec(blockedCtx, `{ greet items(pageSize: 50) }`, "", nil)
	fmt.Printf("blocked:%s\n", blocked.Data)
	fmt.Printf("blocked errors: %+v\n", blocked.Errors)
	fmt.Println("blocked calls:", resolver.calls.Load())

	// Output:
	// allowed: {"greet":"hello","items":["gopher","gopher","gopher","gopher","gopher"]}
	// allowed errors: []
	// allowed calls: 2
	// blocked:
	// blocked errors: [graphql: query cost 51 exceeds threshold 10]
	// blocked calls: 0
}
