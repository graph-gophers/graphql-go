package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/graph-gophers/graphql-go"
)

type mutnb struct{}

func (*mutnb) Toggle(args struct{ Enabled graphql.NullBool }) string {
	if !args.Enabled.Set {
		return "input value was not provided"
	} else if args.Enabled.Value == nil {
		return "enabled is 'null'"
	}
	return fmt.Sprintf("enabled '%v'", *args.Enabled.Value)
}

// ExampleNullBool demonstrates how to use nullable Bool type when it is necessary to differentiate between nil and not set.
func ExampleNullBool() {
	const s = `
		schema {
			query: Query
			mutation: Mutation
		}
		type Query{}
		type Mutation{
			toggle(enabled: Boolean): String!
		}
	`
	schema := graphql.MustParseSchema(s, &mutnb{})

	const query = `mutation{
		toggle1: toggle()
		toggle2: toggle(enabled: null)
		toggle3: toggle(enabled: true)
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
	//   "data": {
	//     "toggle1": "input value was not provided",
	//     "toggle2": "enabled is 'null'",
	//     "toggle3": "enabled 'true'"
	//   }
	// }
}
