package graphql_test

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/graph-gophers/graphql-go"
)

type tquery struct{}

func (*tquery) CurrentTime() graphql.Time {
	return graphql.Time{Time: time.Date(2023, 2, 6, 12, 3, 22, 0, time.UTC)}
}

func ExampleTime() {
	const s = `
		scalar Time

		type Query {
			currentTime: Time!
		}
	`
	schema := graphql.MustParseSchema(s, &tquery{})

	const query = "{ currentTime }"
	res := schema.Exec(context.Background(), query, "", nil)

	err := json.NewEncoder(os.Stdout).Encode(res)
	if err != nil {
		panic(err)
	}

	// output:
	// {"data":{"currentTime":"2023-02-06T12:03:22Z"}}
}
