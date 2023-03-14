package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/graph-gophers/graphql-go"
)

type Map map[string]interface{}

func (Map) ImplementsGraphQLType(name string) bool {
	return name == "Map"
}

func (m *Map) UnmarshalGraphQL(input interface{}) error {
	val, ok := input.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type")
	}
	*m = val
	return nil
}

type Args struct {
	Name string
	Data Map
}

type mutation struct{}

func (*mutation) Hello(args Args) string {
	fmt.Println(args)
	return "Args accepted!"
}

func Example_customScalarMap() {
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

	query := `
	  mutation {
		hello(name: "GraphQL", data: {
			num: 5,
			code: "example"
		})
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
	// {GraphQL map[code:example num:5]}
	// {
	//   "data": {
	//     "hello": "Args accepted!"
	//   }
	// }
}
