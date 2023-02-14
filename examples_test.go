package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
)

type Resolver struct {
	post *Post
}

func (r *Resolver) Post() *Post {
	return r.post
}

type Post struct {
	id    graphql.ID
	title string
}

func (p *Post) ID() graphql.ID {
	return p.id
}

func (p *Post) Title() string {
	return p.title
}

func ExampleID() {
	schemaString := `
		schema {
			query: Query
		}

		type Query {
			post: Post!
		}

		type Post {
			id: ID!
			title: String!
		}
	`

	resolver := &Resolver{
		post: &Post{
			id:    graphql.ID("5"),
			title: "title",
		},
	}

	schema := graphql.MustParseSchema(schemaString, resolver)

	query := `
		query {
			post {
				id
				title
			}
		}
	`

	res := schema.Exec(context.Background(), query, "", nil)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	if err != nil {
		panic(err)
	}
	// output:
	// {
	//   "data": {
	//     "post": {
	//       "id": "5",
	//       "title": "title"
	//     }
	//   }
	// }
}

func ExampleMaxDepth() {
	schema := graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{}, graphql.MaxDepth(3))

	// this query has a depth of 4
	query := `
	  query {
	    hero(episode:EMPIRE) { # level 1
	      name                 # level 2
	      friends { 
	        name               # level 3
	        friends { 
	          id               # level 4 - this would exceed the max depth
	        }
	      }
	    }
	  }`

	res := schema.Exec(context.Background(), query, "", nil)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	if err != nil {
		panic(err)
	}
	// output:
	// {
	//   "errors": [
	//     {
	//       "message": "Field \"id\" has depth 4 that exceeds max depth 3",
	//       "locations": [
	//         {
	//           "line": 8,
	//           "column": 12
	//         }
	//       ]
	//     }
	//   ]
	// }
}

func ExampleMaxQueryLength() {
	schema := graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{}, graphql.MaxQueryLength(50))

	// this query has a length of 53
	query := `{
	  hero(episode:EMPIRE) {
	    id
	    name
	  }
	}`

	res := schema.Exec(context.Background(), query, "", nil)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	if err != nil {
		panic(err)
	}
	// output:
	// {
	//   "errors": [
	//     {
	//       "message": "query length 53 exceeds the maximum allowed query length of 50 bytes"
	//     }
	//   ]
	// }
}

func ExampleRestrictIntrospection() {
	allowKey := struct{}{}
	// only allow introspection if the function below returns true
	filter := func(ctx context.Context) bool {
		allow, found := ctx.Value(allowKey).(bool)
		return found && allow
	}
	schema := graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{}, graphql.RestrictIntrospection(filter))

	query := `{
		__type(name: "Episode") {
			enumValues {
				name
			}
		}
	}`

	cases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "Empty context",
			ctx:  context.Background(),
		},
		{
			name: "Introspection forbidden",
			ctx:  context.WithValue(context.Background(), allowKey, false),
		},
		{
			name: "Introspection allowed",
			ctx:  context.WithValue(context.Background(), allowKey, true),
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
	// Empty context result:
	// {
	//   "data": {}
	// }
	// Introspection forbidden result:
	// {
	//   "data": {}
	// }
	// Introspection allowed result:
	// {
	//   "data": {
	//     "__type": {
	//       "enumValues": [
	//         {
	//           "name": "NEWHOPE"
	//         },
	//         {
	//           "name": "EMPIRE"
	//         },
	//         {
	//           "name": "JEDI"
	//         }
	//       ]
	//     }
	//   }
	// }
}
