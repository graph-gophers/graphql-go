package graphql_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/social"
	"github.com/graph-gophers/graphql-go/example/starwars"
)

func TestSchema_ToJSON(t *testing.T) {
	t.Parallel()

	type args struct {
		Schema *graphql.Schema
	}
	type want struct {
		JSON []byte
	}
	testTable := []struct {
		Name string
		Args args
		Want want
	}{
		{
			Name: "Social Schema",
			Args: args{Schema: graphql.MustParseSchema(social.Schema, &social.Resolver{}, graphql.UseFieldResolvers())},
			Want: want{JSON: mustReadFile("example/social/introspect.json")},
		},
		{
			Name: "Star Wars Schema",
			Args: args{Schema: graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})},
			Want: want{JSON: mustReadFile("example/starwars/introspect.json")},
		},
		{
			Name: "Star Wars Schema without Resolver",
			Args: args{Schema: graphql.MustParseSchema(starwars.Schema, nil)},
			Want: want{JSON: mustReadFile("example/starwars/introspect.json")},
		},
	}

	for _, tt := range testTable {
		t.Run(tt.Name, func(t *testing.T) {
			j, err := tt.Args.Schema.ToJSON()
			if err != nil {
				t.Fatalf("invalid schema %s", err.Error())
			}

			// Verify JSON to avoid red herring errors.
			got, err := formatJSON(j)
			if err != nil {
				t.Fatalf("got: invalid JSON: %s", err)
			}
			want, err := formatJSON(tt.Want.JSON)
			if err != nil {
				t.Fatalf("want: invalid JSON: %s", err)
			}

			if !bytes.Equal(got, want) {
				t.Logf("got:  %s", got)
				t.Logf("want: %s", want)
				t.Fail()
			}
		})
	}
}

func formatJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	formatted, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func mustReadFile(filename string) []byte {
	b, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return b
}
