package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type query struct{}

type IntInput struct {
	A int32
	B int32
}

func (_ *query) Reversed(args struct{ Values []string }) []string {
	result := make([]string, len(args.Values))

	for i, value := range args.Values {
		for _, v := range value {
			result[i] = string(v) + result[i]
		}
	}
	return result
}

func (_ *query) Sums(args struct{ Values []IntInput }) []int32 {
	result := make([]int32, len(args.Values))

	for i, value := range args.Values {
		result[i] = value.A + value.B
	}
	return result
}

func main() {
	s := `
		input IntInput {
			a: Int!
			b: Int!
		}

		type Query {
			reversed(values: [String!]!): [String!]!
			sums(values: [IntInput!]!): [Int!]!
		}
	`
	schema := graphql.MustParseSchema(s, &query{})
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Println("Listen in port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/*
	* The following query

	query{
		reversed(values:["hello", "hi"])
		sums(values:[{a:2,b:3},{a:-10,b:-1}])
	}

	* will return

	{
		"data": {
			"reversed": [
				"olleh",
				"ih"
			],
			"sums": [
				5,
				-11
			]
		}
	}
*/
