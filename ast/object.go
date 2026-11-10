package ast

import "github.com/graph-gophers/graphql-go/errors"

// ObjectTypeDefinition represents a GraphQL ObjectTypeDefinition.
//
//	type FooObject {
//			foo: String
//	}
//
// https://spec.graphql.org/draft/#sec-Objects
type ObjectTypeDefinition struct {
	Name           string
	Interfaces     []*InterfaceTypeDefinition
	Fields         FieldsDefinition
	Desc           string
	DescIsSet      bool
	Directives     DirectiveList
	InterfaceNames []string
	Loc            errors.Location
}

func (*ObjectTypeDefinition) Kind() string             { return "OBJECT" }
func (t *ObjectTypeDefinition) String() string         { return t.Name }
func (t *ObjectTypeDefinition) TypeName() string       { return t.Name }
func (t *ObjectTypeDefinition) Description() string    { return t.Desc }
func (t *ObjectTypeDefinition) DescriptionIsSet() bool { return t.DescIsSet || t.Desc != "" }
