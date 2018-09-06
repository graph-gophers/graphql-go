package graphql

import (
	"context"
	"encoding/json"

	"github.com/graph-gophers/graphql-go/introspection"
)

// Inspect allows inspection of the given schema.
func (s *Schema) Inspect() *introspection.Schema {
	return introspection.WrapSchema(s.engine.schema)
}

// ToJSON encodes the schema in a JSON format used by tools like Relay.
func (s *Schema) ToJSON() ([]byte, error) {
	r := EngineRequest{
		Query: introspectionQuery,
	}
	result := s.engine.Execute(context.Background(), &r, s.resolver)
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
        args {
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
      args {
        ...InputValue
      }
      type {
        ...TypeRef
      }
      isDeprecated
      deprecationReason
    }
    inputFields {
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
