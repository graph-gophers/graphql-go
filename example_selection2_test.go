package graphql_test

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

type (
	user2         struct{ id, name, email string }
	userResolver2 struct{ u user2 }
)

func (r *userResolver2) ID() graphql.ID                               { return graphql.ID(r.u.id) }
func (r *userResolver2) Name() *string                                { return &r.u.name }
func (r *userResolver2) Email() *string                               { return &r.u.email }
func (r *userResolver2) Friends(ctx context.Context) []*userResolver2 { return nil }

type root2 struct{}

func (r *root2) User(ctx context.Context, args struct{ ID string }) *userResolver2 {
	if graphql.HasSelectedField(ctx, "email") {
		fmt.Println("email requested")
	}
	if graphql.HasSelectedField(ctx, "friends") {
		fmt.Println("friends requested")
	}
	return &userResolver2{u: user2{id: args.ID, name: "Alice", email: "a@example.com"}}
}

// Example_hasSelectedField demonstrates HasSelectedField helper for conditional
// logic without needing the full slice of field names. This can be handy when
// checking for a small number of specific fields (avoids allocating the names
// slice if it hasn't already been built).
func Example_hasSelectedField() {
	const s = `
		schema { query: Query }
		type Query { user(id: ID!): User }
		type User { id: ID! name: String email: String friends: [User!]! }
	`
	schema := graphql.MustParseSchema(s, &root2{})
	// Select a subset of fields including a nested composite field; friends requires its own selection set.
	query := `query { user(id: "U1") { id email friends { id } } }`
	_ = schema.Exec(context.Background(), query, "", nil)
	// Output:
	// email requested
	// friends requested
}
