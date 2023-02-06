package graphql_test

import (
	"context"
	"encoding/json"
	"os"

	"github.com/graph-gophers/graphql-go"
)

type query struct{}

type IntTuple struct {
	A int32
	B int32
}

func (*query) Reversed(args struct{ Values []string }) []string {
	result := make([]string, len(args.Values))

	for i, value := range args.Values {
		for _, v := range value {
			result[i] = string(v) + result[i]
		}
	}
	return result
}

func (*query) Sums(args struct{ Values []IntTuple }) []int32 {
	result := make([]int32, len(args.Values))

	for i, value := range args.Values {
		result[i] = value.A + value.B
	}
	return result
}

// Example_inputArray shows a simple GraphQL schema which defines a custom input type.
// Then it executes a query against it passing array arguments.
func Example_inputArray() {
	s := `
	  input IntTuple {
	    a: Int!
	    b: Int!
	  }
	
	  type Query {
	    reversed(values: [String!]!): [String!]!
	    sums(values: [IntTuple!]!): [Int!]!
	  }
	`
	schema := graphql.MustParseSchema(s, &query{})

	query := `
	  query{
	    reversed(values:["hello", "hi"])
	    sums(values:[{a:2,b:3},{a:-10,b:-1}])
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
	//     "reversed": [
	//       "olleh",
	//       "ih"
	//     ],
	//     "sums": [
	//       5,
	//       -11
	//     ]
	//   }
	// }
}
