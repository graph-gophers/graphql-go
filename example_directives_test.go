package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

func (h *HasRoleDirective) Resolve(ctx context.Context, in interface{}, next directives.Resolver) (interface{}, error) {
	if ctx.Value(RoleKey) != h.Role {
		return nil, fmt.Errorf("access deinied, role %q required", h.Role)
	}
	return next.Resolve(ctx, in)
}

type authResolver struct{}

func (*authResolver) Greet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hello, %s!", args.Name)
}

// ExampleDirectives demonstrates the use of the Directives schema option.
func ExampleDirectives() {
	s := `
		schema {
			query: Query
		}

		directive @hasRole(role: String!) on FIELD_DEFINITION

		type Query {
			greet(name: String!): String! @hasRole(role: "admin")
		}
	`
	opts := []graphql.SchemaOpt{
		graphql.Directives(&HasRoleDirective{}),
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
	//       "message": "access deinied, role \"admin\" required",
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
	//     "greet": "Hello, GraphQL!"
	//   }
	// }
}
