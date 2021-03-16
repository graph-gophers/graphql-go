package types

// InterfaceTypeDefinition represents a list of named fields and their arguments.
//
// GraphQL objects can then implement these interfaces which requires that the object type will
// define all fields defined by those interfaces.
//
// http://spec.graphql.org/draft/#sec-Interfaces
type InterfaceTypeDefinition struct {
	Name          string
	PossibleTypes []*ObjectTypeDefinition
	Fields        FieldsDefinition
	Desc          string
	Directives    DirectiveList
}

func (*InterfaceTypeDefinition) Kind() string          { return "INTERFACE" }
func (t *InterfaceTypeDefinition) String() string      { return t.Name }
func (t *InterfaceTypeDefinition) TypeName() string    { return t.Name }
func (t *InterfaceTypeDefinition) Description() string { return t.Desc }
