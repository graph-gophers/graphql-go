// Package authorization contains a simple GraphQL schema using directives.
package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go/directives"
	"github.com/graph-gophers/graphql-go/example/directives/authorization/user"
)

const Schema = `
	schema {
		query: Query
	}

	directive @hasRole(role: Role!) on FIELD_DEFINITION
	
	type Query {
		publicGreet(name: String!): String!
		privateGreet(name: String!): String! @hasRole(role: ADMIN)
	}
	
	enum Role {
		ADMIN
		USER
	}
`

type HasRoleDirective struct {
	Role string
}

func (h *HasRoleDirective) ImplementsDirective() string {
	return "hasRole"
}

func (h *HasRoleDirective) Resolve(ctx context.Context, args interface{}, next directives.Resolver) (output interface{}, err error) {
	u, ok := user.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("user not provided in cotext")
	}
	role := strings.ToLower(h.Role)
	if !u.HasRole(role) {
		return nil, fmt.Errorf("access denied, %q role required", role)
	}

	return next.Resolve(ctx, args)
}

type Resolver struct{}

func (r *Resolver) PublicGreet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hello from the public resolver, %s!", args.Name)
}

func (r *Resolver) PrivateGreet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hi from the protected resolver, %s!", args.Name)
}
