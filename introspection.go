package graphql

import (
	"context"
	"encoding/json"

	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
	"github.com/graph-gophers/graphql-go/introspection"
)

// Inspect allows inspection of the given schema.
func (s *Schema) Inspect() *introspection.Schema {
	return introspection.WrapSchema(s.schema)
}

// ToJSON encodes the schema in a JSON format used by tools like Relay.
func (s *Schema) ToJSON() ([]byte, error) {
	result := s.exec(context.Background(), introspectionQuery, "", nil, &resolvable.Schema{
		Meta:   s.res.Meta,
		Query:  &resolvable.Object{},
		Schema: *s.schema,
	})
	if len(result.Errors) != 0 {
		panic(result.Errors[0])
	}
	return json.MarshalIndent(result.Data, "", "\t")
}

var introspectionQuery = `
  query {
    __schema {
      queryType { name }
      mutationType { name }
      subscriptionType { name }
      types {
        ...FullType
      }
      directives {
        name
        description
        locations
        args(includeDeprecated: true) {
          ...InputValue
        }
      }
    }
  }
  fragment FullType on __Type {
    kind
    name
    description
    fields(includeDeprecated: true) {
      name
      description
      args(includeDeprecated: true) {
        ...InputValue
      }
      type {
        ...TypeRef
      }
      isDeprecated
      deprecationReason
    }
    inputFields(includeDeprecated: true) {
      ...InputValue
    }
    interfaces {
      ...TypeRef
    }
    enumValues(includeDeprecated: true) {
      name
      description
      isDeprecated
      deprecationReason
    }
    possibleTypes {
      ...TypeRef
    }
  }
  fragment InputValue on __InputValue {
    name
    description
    type { ...TypeRef }
    defaultValue
    deprecationReason
  }
  fragment TypeRef on __Type {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
        }
      }
    }
  }
`
