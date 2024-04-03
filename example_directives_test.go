package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/directives"
)

type roleKey string

const RoleKey = roleKey("role")

type HasRoleDirective struct {
	Role string
}

func (h *HasRoleDirective) ImplementsDirective() string {
	return "hasRole"
}

func (h *HasRoleDirective) Validate(ctx context.Context, _ interface{}) error {
	if ctx.Value(RoleKey) != h.Role {
		return fmt.Errorf("access denied, role %q required", h.Role)
	}
	return nil
}

// NullableDirective
type WithNullableArgumentDirective struct {
	ANullableArgument *string
}

func (_ *WithNullableArgumentDirective) Validate(_ context.Context, _ interface{}) error {
	return nil
}

func (_ *WithNullableArgumentDirective) ImplementsDirective() string {
	return "withNullableArgument"
}

type UpperDirective struct{}

func (d *UpperDirective) ImplementsDirective() string {
	return "upper"
}

func (d *UpperDirective) Resolve(ctx context.Context, args interface{}, next directives.Resolver) (interface{}, error) {
	out, err := next.Resolve(ctx, args)
	if err != nil {
		return out, err
	}

	s, ok := out.(string)
	if !ok {
		return out, nil
	}

	return strings.ToUpper(s), nil
}

type authResolver struct{}

func (*authResolver) Greet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hello, %s!", args.Name)
}
func (*authResolver) NullableDirective() string {
	return "nullableDirective"
}

// ExampleDirectives demonstrates the use of the Directives schema option.
func ExampleDirectives() {
	s := `
		schema {
			query: Query
		}

		directive @hasRole(role: String!) on FIELD_DEFINITION
		directive @upper on FIELD_DEFINITION
		directive @withNullableArgument(aNullableArgument: String) on FIELD_DEFINITION

		type Query {
			greet(name: String!): String! @hasRole(role: "admin") @upper
			nullableDirective: String! @withNullableArgument()
		}
	`
	opts := []graphql.SchemaOpt{
		graphql.Directives(&HasRoleDirective{}, &UpperDirective{}, &WithNullableArgumentDirective{}),
		// other options go here
	}
	schema := graphql.MustParseSchema(s, &authResolver{}, opts...)
	query := `
		query {
			greet(name: "GraphQL")
		}
	`
	cases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "Unauthorized",
			ctx:  context.Background(),
		},
		{
			name: "Admin user",
			ctx:  context.WithValue(context.Background(), RoleKey, "admin"),
		},
	}
	for _, c := range cases {
		fmt.Println(c.name, "result:")
		res := schema.Exec(c.ctx, query, "", nil)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err := enc.Encode(res)
		if err != nil {
			panic(err)
		}
	}
	// output:
	// Unauthorized result:
	// {
	//   "errors": [
	//     {
	//       "message": "access denied, role \"admin\" required",
	//       "locations": [
	//         {
	//           "line": 11,
	//           "column": 4
	//         }
	//       ],
	//       "path": [
	//         "greet"
	//       ]
	//     }
	//   ],
	//   "data": null
	// }
	// Admin user result:
	// {
	//   "data": {
	//     "greet": "HELLO, GRAPHQL!"
	//   }
	// }
}
