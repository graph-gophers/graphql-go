package graphql_test

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

type (
	user         struct{ id, name, email string }
	userResolver struct{ u user }
)

func (r *userResolver) ID() graphql.ID { return graphql.ID(r.u.id) }
func (r *userResolver) Name() *string  { return &r.u.name }
func (r *userResolver) Email() *string { return &r.u.email }
func (r *userResolver) Friends(ctx context.Context) []*userResolver {
	// Return a couple of dummy friends (data itself not important for field selection example)
	return []*userResolver{
		{u: user{id: "F1", name: "Bob"}},
		{u: user{id: "F2", name: "Carol"}},
	}
}

type root struct{}

func (r *root) User(ctx context.Context, args struct{ ID string }) *userResolver {
	fields := graphql.SelectedFieldNames(ctx)
	fmt.Println(fields)
	return &userResolver{u: user{id: args.ID, name: "Alice", email: "a@example.com"}}
}

// Example_selectedFieldNames demonstrates SelectedFieldNames usage in a resolver for
// conditional data fetching (e.g. building a DB projection list).
func Example_selectedFieldNames() {
	const s = `
        schema { query: Query }
        type Query { user(id: ID!): User }
        type User { id: ID! name: String email: String friends: [User!]! }
    `
	schema := graphql.MustParseSchema(s, &root{})
	query := `query { user(id: "U1") { id name friends { id name } } }`
	_ = schema.Exec(context.Background(), query, "", nil)
	// Output:
	// [id name friends friends.id friends.name]
}
