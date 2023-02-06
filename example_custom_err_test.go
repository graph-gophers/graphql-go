package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/graph-gophers/graphql-go"
)

type product struct {
	ID   graphql.ID
	Name string
}

type custErrResolver struct {
	products map[graphql.ID]*product
}

func (r *custErrResolver) Product(ctx context.Context, args struct{ ID graphql.ID }) (*productResolver, error) {
	if p := r.products[args.ID]; p != nil {
		return &productResolver{p: p}, nil
	}
	traceID := "your-trace-id-here" // get trace ID from ctx
	return nil, &productNotFoundError{Code: "NotFound", Message: "Product not found", TraceID: traceID}
}

type productResolver struct {
	p *product
}

func (r *productResolver) ID() graphql.ID {
	return r.p.ID
}

func (r *productResolver) Name() string {
	return r.p.Name
}

type productNotFoundError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId"`
}

func (e productNotFoundError) Error() string {
	return fmt.Sprintf("error [%s]: %s.", e.Code, e.Message)
}

// Extensions provides additional error context according to the spec https://spec.graphql.org/October2021/#sel-GAPHRPZCAACCBx6b.
func (e productNotFoundError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
		"traceId": e.TraceID,
	}
}

// Example_customErrors demonstrates the use of custom errors and error extensions.
func Example_customErrors() {
	var products = []*product{
		{ID: "1000", Name: "Product1"},
		{ID: "1001", Name: "Product2"},
	}
	resolver := &custErrResolver{
		products: map[graphql.ID]*product{},
	}
	for _, p := range products {
		resolver.products[p.ID] = p
	}
	s := `
	schema {
		query: Query
	}

	type Query {
		product(id: ID!): Product!
	}

	type Product {
		id: ID!
		name: String!
	}
	`
	schema := graphql.MustParseSchema(s, resolver)

	query := `
	  query {
		product(id: "1007") {
			id
			name
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
	//   "errors": [
	//     {
	//       "message": "error [NotFound]: Product not found.",
	//       "path": [
	//         "product"
	//       ],
	//       "extensions": {
	//         "code": "NotFound",
	//         "message": "Product not found",
	//         "traceId": "your-trace-id-here"
	//       }
	//     }
	//   ],
	//   "data": null
	// }
}
