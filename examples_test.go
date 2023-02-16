package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

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

func ExampleSchema_ASTSchema() {
	schema := graphql.MustParseSchema(starwars.Schema, nil)
	ast := schema.ASTSchema()

	for _, e := range ast.Enums {
		fmt.Printf("Enum %q has the following options:\n", e.Name)
		for _, o := range e.EnumValuesDefinition {
			fmt.Printf("  - %s\n", o.EnumValue)
		}
	}
	// output:
	// Enum "Episode" has the following options:
	//   - NEWHOPE
	//   - EMPIRE
	//   - JEDI
	// Enum "LengthUnit" has the following options:
	//   - METER
	//   - FOOT
}

func ExampleSchema_ASTSchema_generateEnum() {
	s := `
		schema {
			query: Query
		}

		type Query {
			currentSeason: Season!
		}

		"""
		Season represents a season of the year.
		"""
		enum Season {
			SPRING
			SUMMER
			AUTUMN
			WINTER
		}
	`

	gocode := `
{{ $enum := . }}
// {{ $enum.Desc }}
type {{ $enum.Name }} int

const (
	{{ range $i, $e :=  $enum.EnumValuesDefinition }}{{ if ne $i 0 }}{{ printf "\n\t" }}{{ end }}
		{{- $e.EnumValue | toVar }}
		{{- if eq $i 0 }} {{ $enum.Name }} = iota{{ end }}
	{{- end }}
)

func (s Season) String() string {
	switch s {
	{{ range $i, $e :=  $enum.EnumValuesDefinition }}{{ if ne $i 0 }}{{ printf "\n\t" }}{{ end -}}
		case {{ $e.EnumValue | toVar }}:
			return "{{ $e.EnumValue }}"
	{{- end }}
	}
	panic("unreachable")
}
	`
	toVar := func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	}
	funcMap := template.FuncMap{
		"toVar": toVar,
	}
	tpl, err := template.New("enum").Funcs(funcMap).Parse(gocode)
	if err != nil {
		panic(err)
	}

	opts := []graphql.SchemaOpt{
		graphql.UseStringDescriptions(),
	}

	schema := graphql.MustParseSchema(s, nil, opts...)
	ast := schema.ASTSchema()
	seasons := ast.Enums[0]

	err = tpl.Execute(os.Stdout, seasons)
	if err != nil {
		panic(err)
	}
	// output:
	// // Season represents a season of the year.
	// type Season int
	//
	// const (
	// 	Spring Season = iota
	// 	Summer
	// 	Autumn
	// 	Winter
	// )
	//
	// func (s Season) String() string {
	// 	switch s {
	// 	case Spring:
	// 			return "SPRING"
	// 	case Summer:
	// 			return "SUMMER"
	// 	case Autumn:
	// 			return "AUTUMN"
	// 	case Winter:
	// 			return "WINTER"
	// 	}
	// 	panic("unreachable")
	// }
}
