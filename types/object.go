package types

// ObjectTypeDefinition represents a GraphQL ObjectTypeDefinition.
//
// type FooObject {
// 		foo: String
// }
//
// https://spec.graphql.org/draft/#sec-Objects
type ObjectTypeDefinition struct {
	Name       string
	Interfaces []*InterfaceTypeDefinition
	Fields     FieldsDefinition
	Desc       string
	Directives DirectiveList

	InterfaceNames []string
}

func (*ObjectTypeDefinition) Kind() string          { return "OBJECT" }
func (t *ObjectTypeDefinition) String() string      { return t.Name }
func (t *ObjectTypeDefinition) TypeName() string    { return t.Name }
func (t *ObjectTypeDefinition) Description() string { return t.Desc }
