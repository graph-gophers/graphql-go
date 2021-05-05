package graphql_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/internal/query"
)

func equalLoggedOperation(a, b graphql.LoggedOperation) bool {
	if a.Type != b.Type {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	for k, va := range a.Variables {
		if vb, ok := b.Variables[k]; !ok || va != vb {
			return false
		}
	}
	for k, vb := range b.Variables {
		if va, ok := a.Variables[k]; !ok || vb != va {
			return false
		}
	}
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for i := range a.Fields {
		if !equalLoggedField(a.Fields[i], b.Fields[i]) {
			return false
		}
	}
	return true
}

func equalLoggedField(a, b graphql.LoggedField) bool {
	if a.Name != b.Name {
		return false
	}
	for k, va := range a.Arguments {
		if vb, ok := b.Arguments[k]; !ok || va != vb {
			return false
		}
	}
	for k, vb := range b.Arguments {
		if va, ok := a.Arguments[k]; !ok || vb != va {
			return false
		}
	}
	return true
}

func TestValidateAndLog(t *testing.T) {
	tests := []struct {
		name        string
		schema      *graphql.Schema
		queryString string
		want        []graphql.LoggedOperation
		wantErrs    bool
	}{
		{
			"basic test",
			starwarsSchema,
			`
			{
				hero {
					id
					name
					friends {
						name
					}
				}
			}
			`,
			[]graphql.LoggedOperation{
				{
					Type:      query.Query,
					Variables: map[string]string{},
					Fields: []graphql.LoggedField{
						{
							Name:      "hero",
							Arguments: map[string]string{},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, got := tt.schema.ValidateAndLog(tt.queryString, map[string]interface{}{})
			if !tt.wantErrs && len(errs) > 0 {
				t.Errorf("Schema.ValidateAndLog() got errors %+v", errs)
			}
			if len(tt.want) != len(got) {
				t.Errorf("Schema.ValidateAndLog() = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if !equalLoggedOperation(tt.want[i], got[i]) {
					t.Errorf("Schema.ValidateAndLog()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
