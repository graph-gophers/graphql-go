package log_test

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/log"
)

func ExampleLoggerFunc() {
	logfn := log.LoggerFunc(func(ctx context.Context, err interface{}) {
		// Here you can handle the panic, e.g., log it or send it to an error tracking service.
		fmt.Printf("graphql: panic occurred: %v", err)
	})

	opts := []graphql.SchemaOpt{
		graphql.Logger(logfn),
		graphql.UseFieldResolvers(),
	}

	schemadef := `
		type Query {
			hello: String!
		}
	`
	resolver := &struct {
		Hello func() string
	}{
		Hello: func() string {
			// Simulate a panic
			panic("something went wrong")
		},
	}

	schema := graphql.MustParseSchema(schemadef, resolver, opts...)
	// Now, when you execute a query that causes a panic, it will be logged using the provided LoggerFunc.
	schema.Exec(context.Background(), "{ hello }", "", nil)

	// Output:
	// graphql: panic occurred: something went wrong
}
