package graphql_test

import (
	"context"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
)

// Benchmarks to measure memory impact of buffer pooling optimizations

const memBenchSchema = `
	schema { query: Query }
	type Query {
		user(id: ID!): User
		users: [User!]!
	}
	type User {
		id: ID!
		name: String!
		email: String!
		posts: [Post!]!
	}
	type Post {
		id: ID!
		title: String!
		content: String!
		author: User!
	}
`

type memBenchResolver struct{}

type memBenchUser struct {
	id, name, email string
	postCount       int
}

type memBenchPost struct {
	id, title, content string
	user               *memBenchUser
}

func (r *memBenchResolver) User(args struct{ ID string }) *memBenchUser {
	return &memBenchUser{
		id:        args.ID,
		name:      "Test User",
		email:     "test@example.com",
		postCount: 10,
	}
}

func (r *memBenchResolver) Users() []*memBenchUser {
	users := make([]*memBenchUser, 50)
	for i := 0; i < 50; i++ {
		users[i] = &memBenchUser{
			id:        string(rune('A' + i)),
			name:      "User Name",
			email:     "user@example.com",
			postCount: 5,
		}
	}
	return users
}

func (u *memBenchUser) ID() graphql.ID {
	return graphql.ID(u.id)
}

func (u *memBenchUser) Name() string {
	return u.name
}

func (u *memBenchUser) Email() string {
	return u.email
}

func (u *memBenchUser) Posts() []*memBenchPost {
	posts := make([]*memBenchPost, u.postCount)
	for i := 0; i < u.postCount; i++ {
		posts[i] = &memBenchPost{
			id:      string(rune('0' + i)),
			title:   "Post Title",
			content: "This is the post content with some reasonable length to simulate real data.",
			user:    u,
		}
	}
	return posts
}

func (p *memBenchPost) ID() graphql.ID {
	return graphql.ID(p.id)
}

func (p *memBenchPost) Title() string {
	return p.title
}

func (p *memBenchPost) Content() string {
	return p.content
}

func (p *memBenchPost) Author() *memBenchUser {
	return p.user
}

// Simple query - single object with nested fields
func BenchmarkMemory_SimpleQuery(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	ctx := context.Background()
	query := `query { user(id: "1") { id name email } }`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := schema.Exec(ctx, query, "", nil)
		if len(result.Errors) > 0 {
			b.Fatal(result.Errors)
		}
	}
}

// List query - tests array buffer allocation
func BenchmarkMemory_ListQuery(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	ctx := context.Background()
	query := `query { users { id name email } }`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := schema.Exec(ctx, query, "", nil)
		if len(result.Errors) > 0 {
			b.Fatal(result.Errors)
		}
	}
}

// Deeply nested query - tests recursive buffer allocation
func BenchmarkMemory_NestedQuery(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	ctx := context.Background()
	query := `query {
		user(id: "1") {
			id
			name
			email
			posts {
				id
				title
				content
				author {
					id
					name
					email
				}
			}
		}
	}`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := schema.Exec(ctx, query, "", nil)
		if len(result.Errors) > 0 {
			b.Fatal(result.Errors)
		}
	}
}

// List with nested lists - maximum buffer churn
func BenchmarkMemory_ListWithNestedLists(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	ctx := context.Background()
	query := `query {
		users {
			id
			name
			posts {
				id
				title
			}
		}
	}`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := schema.Exec(ctx, query, "", nil)
		if len(result.Errors) > 0 {
			b.Fatal(result.Errors)
		}
	}
}

// Concurrent execution - tests pool contention
func BenchmarkMemory_Concurrent(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	query := `query { users { id name email posts { id title } } }`

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			result := schema.Exec(ctx, query, "", nil)
			if len(result.Errors) > 0 {
				b.Fatal(result.Errors)
			}
		}
	})
}

// Memory allocation test - run with -benchmem to see allocations
func BenchmarkMemory_AllocationsPerOp(b *testing.B) {
	schema := graphql.MustParseSchema(memBenchSchema, &memBenchResolver{})
	ctx := context.Background()

	queries := []struct {
		name  string
		query string
	}{
		{"Single", `query { user(id: "1") { id name } }`},
		{"List_10", `query { users { id } }`},
		{"Nested_Depth3", `query { user(id: "1") { posts { author { id } } } }`},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := schema.Exec(ctx, q.query, "", nil)
				if len(result.Errors) > 0 {
					b.Fatal(result.Errors)
				}
			}
		})
	}
}
