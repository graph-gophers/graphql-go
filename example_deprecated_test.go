package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	graphql "github.com/graph-gophers/graphql-go"
)

type testValidateDeprecatedResolver struct{}

func (r *testValidateDeprecatedResolver) DeprecatedField() string {
	return "old value"
}

func Example_validateDeprecated() {
	const sdl = `
		schema {
			query: Query
		}
		type Query {
			deprecatedField: String! @deprecated(reason: "Use replacementField")
		}
	`

	schema := graphql.MustParseSchema(sdl, &testValidateDeprecatedResolver{})
	res := schema.Exec(context.Background(), `{ deprecatedField }`, "", nil)
	fmt.Println("Without validation:")
	_ = json.NewEncoder(os.Stdout).Encode(res)

	clone := schema.MustClone(&testValidateDeprecatedResolver{}, graphql.ValidateDeprecated())
	res = clone.Exec(context.Background(), `{ deprecatedField }`, "", nil)
	fmt.Println("With validation:")
	_ = json.NewEncoder(os.Stdout).Encode(res)

	// Output:
	// Without validation:
	// {"data":{"deprecatedField":"old value"}}
	// With validation:
	// {"errors":[{"message":"The field Query.deprecatedField is deprecated. Use replacementField","locations":[{"line":1,"column":3}]}]}
}
