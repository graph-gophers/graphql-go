package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/graph-gophers/graphql-go"
)

type exampleResolver struct{}

func (*exampleResolver) Greet(ctx context.Context, args struct{ Name string }) string {
	return fmt.Sprintf("Hello, %s!", args.Name)
}

// Example demonstrates how to parse a GraphQL schema and execute a query against it.
func Example() {
	s := `
	  schema {
	    query: Query
	  }
	  
	  type Query {
	    greet(name: String!): String!
	  }
	`
	opts := []graphql.SchemaOpt{
		// schema options go here
	}
	schema := graphql.MustParseSchema(s, &exampleResolver{}, opts...)
	query := `
		query {
			greet(name: "GraphQL")
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
	//     "greet": "Hello, GraphQL!"
	//   }
	// }
}
