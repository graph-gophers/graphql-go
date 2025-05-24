package graphql_test

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
)

func FuzzSchemaExec(f *testing.F) {
	resolver := &starwars.Resolver{}
	opts := []graphql.SchemaOpt{graphql.MaxDepth(3)}
	schema, err := graphql.ParseSchema(starwars.Schema, resolver, opts...)
	if err != nil {
		f.Errorf("ParseSchema: %v", err)
		return
	}

	queries := []string{
		`{ hero { name } }`,
		`{ hero { name appearsIn } }`,
		`{ hero { name appearsIn friends { name } } }`,
		`{ hero(episode: EMPIRE) { name } }`,
		`{ episode(episode: EMPIRE) { title characters { name } } }`,
		`{ episode(episode: EMPIRE) { title characters { name friends { name } } } }`,
		`query { episode(episode: EMPIRE) { title characters { name friends { name } } } }`,
		`query HeroName { hero { name } }`,
		`query HeroNameAndFriends { hero { name friends { name } } }`,
		`query HeroNameAndFriendsWithAlias { hero { name friends { name alias: name } } }`,
	}
	for _, q := range queries {
		f.Add(q)
	}

	f.Fuzz(func(t *testing.T, query string) {
		// ignore invalid queries in order to test only the execution against the schema
		errs := schema.Validate(query)
		if len(errs) > 0 {
			t.Skip()
		}

		res := schema.Exec(context.Background(), query, "", nil)
		if res.Data != nil && len(res.Errors) > 0 {
			t.Errorf("Exec(%q) returned both data and errors: %v", query, res.Errors)
		}
		if res.Errors != nil {
			t.Logf("Exec(%q) returned errors: %v", query, res.Errors)
		}
		if res.Data == nil && len(res.Errors) == 0 {
			t.Errorf("Exec(%q) returned nil data and no errors", query)
		}
	})
}
