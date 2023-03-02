package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

//go:embed index.html
var page []byte

//go:embed schema.graphql
var sdl string

type entitiesFunc func(args struct{ Representations []Any }) ([]*Entity, error)

type resolver struct {
	posts map[graphql.ID]Post

	Entities entitiesFunc   `graphql:"_entities"`
	Service  func() Service `graphql:"_service"`
}

func (r *resolver) Post(args struct{ ID graphql.ID }) (*Post, error) {
	p, ok := r.posts[args.ID]
	if !ok {
		return nil, fmt.Errorf("post with id %q not found", args.ID)
	}
	return &p, nil
}

type Post struct {
	ID    graphql.ID `json:"id"`
	Title string     `json:"title"`
}

type Any struct {
	TypeName string `json:"__typename"`
	Keys     map[string]interface{}
}

func (a *Any) UnmarshalJSON(d []byte) error {
	m := map[string]interface{}{}
	err := json.Unmarshal(d, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal keys: %w", err)
	}
	delete(m, "__typename") // remove duplicate key
	(*a).Keys = m

	var temp struct {
		T string `json:"__typename"`
	}
	err = json.Unmarshal(d, &temp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal typename: %w", err)
	}
	a.TypeName = temp.T
	return nil
}

func (Any) ImplementsGraphQLType(name string) bool {
	return name == "_Any"
}

func (a *Any) UnmarshalGraphQL(input interface{}) error {
	var data []byte
	switch val := input.(type) {
	case string:
		v := fmt.Sprint(val) // this line turns all the `\"` into `"`
		data = []byte(v)
	default:
		return fmt.Errorf("want string input, got %T", input)
	}

	return json.Unmarshal(data, a)
}

func entities(posts map[graphql.ID]Post) entitiesFunc {
	return func(args struct{ Representations []Any }) ([]*Entity, error) {
		var res []*Entity
		for _, rep := range args.Representations {
			switch rep.TypeName {
			case "Post":
				val, found := rep.Keys["id"]
				if !found {
					return nil, fmt.Errorf("required key id was not provided")
				}
				id, ok := val.(string)
				if !ok {
					return nil, fmt.Errorf("expected a string, got %T", val)
				}
				p, ok := posts[graphql.ID(id)]
				if !ok {
					return nil, fmt.Errorf("post with id %q, not found", id)
				}
				res = append(res, &Entity{entity: &p})

			default:
				return nil, fmt.Errorf("unexpected representation type %q", rep.TypeName)
			}
		}

		return res, nil
	}
}

type Entity struct {
	entity interface{}
}

func (e *Entity) ToPost() (*Post, bool) {
	p, ok := e.entity.(*Post)
	return p, ok
}

func service(s string) func() Service {
	return func() Service {
		return Service{SDL: s}
	}
}

type Service struct {
	SDL string
}

func main() {
	// If you want to generate dynamically all the entities which have the `key` directive
	//
	//   s := graphql.MustParseSchema(sdl, nil)
	//   ast := s.AST()
	//   var ent []string
	//   for _, o := range ast.Objects {
	//       for _, d := range o.Directives {
	//           if d.Name.Name == "key" {
	//               ent = append(ent, o.Name)
	//               break
	//           }
	//       }
	//   }
	//
	// ent contains all the type names.

	posts := map[graphql.ID]Post{
		graphql.ID("1"): {ID: graphql.ID("1"), Title: "Title 1"},
		graphql.ID("2"): {ID: graphql.ID("2"), Title: "Title 2"},
		graphql.ID("3"): {ID: graphql.ID("3"), Title: "Title 3"},
	}

	r := &resolver{
		posts:    posts,
		Entities: entities(posts),
		Service:  service(sdl),
	}
	opts := []graphql.SchemaOpt{graphql.UseStringDescriptions(), graphql.UseFieldResolvers()}
	schema := graphql.MustParseSchema(sdl, r, opts...)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write(page) })
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
