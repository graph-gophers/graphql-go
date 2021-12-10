package main

import (
	"fmt"
	"log"
	"net/http"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/scalar_map/types"
	"github.com/graph-gophers/graphql-go/relay"
)

type Args struct {
	Name string
	Data types.Map
}

type mutation struct{}

func (_ *mutation) Hello(args Args) string {

	fmt.Println(args)

	return "Args accept!"
}

func main() {
	s := `
		scalar Map
	
		type Query {}
		
		type Mutation {
			hello(
				name: String!
				data: Map!
			): String!
		}
	`
	schema := graphql.MustParseSchema(s, &mutation{})
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Println("Listen in port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
