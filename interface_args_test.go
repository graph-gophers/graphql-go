package graphql_test

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/gqltesting"
)

const interfaceArgsQuery = `{ named { name } }`

type interfaceArgsContextKey struct{}

type defaultArgsQueryResolver struct{}

func (*defaultArgsQueryResolver) Named() []*defaultArgsNamedResolver {
	return []*defaultArgsNamedResolver{{}}
}

type defaultArgsNamedResolver struct{}

func (*defaultArgsNamedResolver) ToUser() (*defaultArgsUserResolver, bool) {
	return &defaultArgsUserResolver{}, true
}

type defaultArgsUserResolver struct{}

func (*defaultArgsUserResolver) Name(args struct{ Type string }) string {
	return args.Type
}

func TestInterfaceImplementerOptionalArgDefault(t *testing.T) {
	t.Parallel()

	const schema = `
		interface Named {
			name: String!
		}

		type User implements Named {
			name(type: NameType = FULL): String!
		}

		enum NameType {
			FULL
			FIRST
			LAST
		}

		type Query {
			named: [Named!]!
		}
	`

	parsedSchema := graphql.MustParseSchema(schema, &defaultArgsQueryResolver{}, graphql.UseFieldResolvers())

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema:         parsedSchema,
			Query:          interfaceArgsQuery,
			ExpectedResult: `{"named":[{"name":"FULL"}]}`,
		},
		{
			Schema:         parsedSchema,
			Query:          `{ named { ... on User { name(type: FIRST) } } }`,
			ExpectedResult: `{"named":[{"name":"FIRST"}]}`,
		},
		{
			Schema:         parsedSchema,
			Query:          `{ named { ... on User { name(type: LAST) } } }`,
			ExpectedResult: `{"named":[{"name":"LAST"}]}`,
		},
	})
}

func (*mixedArgsQueryResolver) Named(ctx context.Context) []*mixedArgsNamedResolver {
	return ctx.Value(interfaceArgsContextKey{}).([]*mixedArgsNamedResolver)
}

type mixedArgsNamedResolver struct {
	user  *mixedArgsUserResolver
	droid *mixedArgsDroidResolver
}

type mixedArgsQueryResolver struct{}

func mixedUserNamed() *mixedArgsNamedResolver {
	return &mixedArgsNamedResolver{user: &mixedArgsUserResolver{}}
}

func mixedDroidNamed() *mixedArgsNamedResolver {
	return &mixedArgsNamedResolver{droid: &mixedArgsDroidResolver{}}
}

func (r *mixedArgsNamedResolver) ToUser() (*mixedArgsUserResolver, bool) {
	return r.user, r.user != nil
}

func (r *mixedArgsNamedResolver) ToDroid() (*mixedArgsDroidResolver, bool) {
	return r.droid, r.droid != nil
}

type mixedArgsUserResolver struct{}

func (*mixedArgsUserResolver) Name(args struct {
	Style string
	Upper bool
},
) string {
	if args.Upper {
		return "upper"
	}
	return args.Style
}

type mixedArgsDroidResolver struct{}

func (*mixedArgsDroidResolver) Name(args struct{ Style string }) string {
	return "droid:" + args.Style
}

func TestInterfaceFallbackWithMixedImplementerArgs(t *testing.T) {
	t.Parallel()

	const schema = `
		interface Named {
			name: String!
		}

		type User implements Named {
			name(style: NameType = FULL, upper: Boolean = false): String!
		}

		type Droid implements Named {
			name(style: NameType = FULL): String!
		}

		enum NameType {
			FULL
			FIRST
			LAST
		}

		type Query {
			named: [Named!]!
		}
	`

	parsedSchema := graphql.MustParseSchema(schema, &mixedArgsQueryResolver{}, graphql.UseFieldResolvers())

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedUserNamed(), mixedDroidNamed()}),
			Schema:         parsedSchema,
			Query:          interfaceArgsQuery,
			ExpectedResult: `{"named":[{"name":"FULL"},{"name":"droid:FULL"}]}`,
		},
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedDroidNamed(), mixedUserNamed()}),
			Schema:         parsedSchema,
			Query:          interfaceArgsQuery,
			ExpectedResult: `{"named":[{"name":"droid:FULL"},{"name":"FULL"}]}`,
		},
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedUserNamed(), mixedUserNamed(), mixedDroidNamed()}),
			Schema:         parsedSchema,
			Query:          interfaceArgsQuery,
			ExpectedResult: `{"named":[{"name":"FULL"},{"name":"FULL"},{"name":"droid:FULL"}]}`,
		},
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedUserNamed(), mixedDroidNamed(), mixedUserNamed()}),
			Schema:         parsedSchema,
			Query:          `{ named { name ... on User { userName: name(style: LAST) } ... on Droid { droidName: name(style: LAST) } } }`,
			ExpectedResult: `{"named":[{"name":"FULL","userName":"LAST"},{"name":"droid:FULL","droidName":"droid:LAST"},{"name":"FULL","userName":"LAST"}]}`,
		},
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedUserNamed(), mixedDroidNamed()}),
			Schema:         parsedSchema,
			Query:          `{ named { name ... on User { userName: name(style: FIRST, upper: true) } ... on Droid { droidName: name(style: FIRST) } } }`,
			ExpectedResult: `{"named":[{"name":"FULL","userName":"upper"},{"name":"droid:FULL","droidName":"droid:FIRST"}]}`,
		},
		{
			Context:        context.WithValue(context.Background(), interfaceArgsContextKey{}, []*mixedArgsNamedResolver{mixedDroidNamed()}),
			Schema:         parsedSchema,
			Query:          interfaceArgsQuery,
			ExpectedResult: `{"named":[{"name":"droid:FULL"}]}`,
		},
	})
}
