package main

import (
	"log"
	"net/http"
	"time"

	graphql "github.com/tribunadigital/graphql-go"
	"github.com/tribunadigital/graphql-go/relay"
)

type query struct{}

func (_ *query) CurrentTime() graphql.Time {
	return graphql.Time{Time: time.Now()}
}

func main() {
	s := `
		scalar Time

		type Query {
			currentTime: Time!
		}
	`
	schema := graphql.MustParseSchema(s, &query{})
	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Println("Listen in port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
