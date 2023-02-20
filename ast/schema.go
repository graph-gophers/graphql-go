package ast

import "github.com/graph-gophers/graphql-go/errors"

// Schema represents a GraphQL service's collective type system capabilities.
// A schema is defined in terms of the types and directives it supports as well as the root
// operation types for each kind of operation: `query`, `mutation`, and `subscription`.
//
// For a more formal definition, read the relevant section in the specification:
//
// http://spec.graphql.org/draft/#sec-Schema
type Schema struct {
	// SchemaDefinition corresponds to the `schema` sdl keyword.
	SchemaDefinition

	// Types are the fundamental unit of any GraphQL schema.
	// There are six kinds of named type definitions in GraphQL, and two wrapping types.
	//
	// http://spec.graphql.org/draft/#sec-Types
	Types map[string]NamedType

	// Directives are used to annotate various parts of a GraphQL document as an indicator that they
	// should be evaluated differently by a validator, executor, or client tool such as a code
	// generator.
	//
	// http://spec.graphql.org/#sec-Type-System.Directives
	Directives map[string]*DirectiveDefinition

	Objects      []*ObjectTypeDefinition
	Unions       []*Union
	Enums        []*EnumTypeDefinition
	Extensions   []*Extension
	SchemaString string
}

func (s *Schema) Resolve(name string) Type {
	return s.Types[name]
}

// SchemaDefinition is an optional schema block.
// If the schema definition is present it might contain a description and directives. It also contains a map of root operations. For example:
//
//	schema {
//	  query: Query
//	  mutation: Mutation
//	  subscription: Subscription
//	}
//
//	type Query {
//	  # query fields go here
//	}
//
//	type Mutation {
//	  # mutation fields go here
//	}
//
//	type Subscription {
//	  # subscription fields go here
//	}
//
// If the root operations have default names (i.e. Query, Mutation and Subscription), then the schema definition can be omitted. For example, this is equivalent to the above schema:
//
//	type Query {
//	  # query fields go here
//	}
//
//	type Mutation {
//	  # mutation fields go here
//	}
//
//	type Subscription {
//	  # subscription fields go here
//	}
//
// https://spec.graphql.org/October2021/#sec-Schema
type SchemaDefinition struct {
	// Present is true if the schema definition is not omitted, false otherwise. For example, in the following schema
	//
	//	type Query {
	//		hello: String!
	//	}
	//
	// the schema keyword is omitted since the default name for Query is used. In that case Present would be false.
	Present bool

	// RootOperationTypes determines the place in the type system where `query`, `mutation`, and
	// `subscription` operations begin.
	//
	// http://spec.graphql.org/draft/#sec-Root-Operation-Types
	RootOperationTypes map[string]NamedType

	EntryPointNames map[string]string
	Desc            string
	Directives      DirectiveList
	Loc             errors.Location
}
