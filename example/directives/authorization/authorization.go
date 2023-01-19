// Package authorization contains a simple GraphQL schema using directives.
package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go/example/directives/authorization/user"
	"github.com/graph-gophers/graphql-go/types"
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

type HasRoleDirective struct{}

func (h *HasRoleDirective) Before(ctx context.Context, directive *types.Directive, input interface{}) (bool, error) {
	u, ok := user.FromContext(ctx)
	if !ok {
		return true, fmt.Errorf("user not provided in cotext")
	}
	role := strings.ToLower(directive.Arguments.MustGet("role").String())
	if !u.HasRole(role) {
		return true, fmt.Errorf("access denied, %q role required", role)
	}
	return false, nil
}

func (h *HasRoleDirective) After(ctx context.Context, directive *types.Directive, output interface{}) (interface{}, error) {
	return output, nil
}

type Resolver struct{}

func (r *Resolver) PublicGreet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hello from the public resolver, %s!", args.Name)
}

func (r *Resolver) PrivateGreet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hi from the protected resolver, %s!", args.Name)
}
