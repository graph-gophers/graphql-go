package graphql_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestInputFieldDefinitionDirectiveLocation(t *testing.T) {
	sdl := `
		directive @custom on INPUT_FIELD_DEFINITION
		input Filter { value: String @custom }
		type Query { search(filter: Filter): String }
	`
	if _, err := graphql.ParseSchema(sdl, nil); err != nil {
		t.Fatalf("INPUT_FIELD_DEFINITION directive on an input field should be accepted: %v", err)
	}
}

func TestArgumentDefinitionDirectiveLocation(t *testing.T) {
	sdl := `
		directive @onarg on ARGUMENT_DEFINITION
		type Query { search(filter: String @onarg): String }
	`
	if _, err := graphql.ParseSchema(sdl, nil); err != nil {
		t.Fatalf("ARGUMENT_DEFINITION directive on an argument should be accepted: %v", err)
	}
}
