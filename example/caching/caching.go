package caching

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go/example/caching/cache"
)

const Schema = `
	schema {
		query: Query
	}

	type Query {
		hello(name: String!): String!
		me: UserProfile!
	}

	type UserProfile {
		name: String!
	}
`

type Resolver struct{}

func (r Resolver) Hello(ctx context.Context, args struct{ Name string }) string {
	cache.AddHint(ctx, cache.Hint{MaxAge: cache.TTL(1 * time.Hour), Scope: cache.ScopePublic})
	return "Hello " + args.Name + "!"
}

func (r Resolver) Me(ctx context.Context) *UserProfile {
	cache.AddHint(ctx, cache.Hint{MaxAge: cache.TTL(1 * time.Minute), Scope: cache.ScopePrivate})
	return &UserProfile{name: "World"}
}

type UserProfile struct {
	name string
}

func (p *UserProfile) Name() string {
	return p.name
}
