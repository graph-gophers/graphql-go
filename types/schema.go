/*
Package types represents all types from the GraphQL specification in code.

The names of the Go types, whenever possible, match 1:1 with the names from
the specification.

Deprecated: Use package [ast] instead. This package will be deleted in future versions of the library.
*/
package types

import "github.com/graph-gophers/graphql-go/ast"

// Schema is here for backwards compatibility.
//
// Deprecated: use [ast.Schema] instead.
type Schema = ast.Schema

// Deprecated: use [ast.Argument] instead.
type Argument = ast.Argument

// Deprecated: use [ast.ArgumentList] instead.
type ArgumentList = ast.ArgumentList

// Deprecated: use [ast.ArgumentsDefinition] instead.
type ArgumentsDefinition = ast.ArgumentsDefinition

// Deprecated: use [ast.Directive] instead.
type Directive = ast.Directive

// Deprecated: use [ast.DirectiveDefinition] instead.
type DirectiveDefinition = ast.DirectiveDefinition

// Deprecated: use [ast.DirectiveList] instead.
type DirectiveList = ast.DirectiveList

// Deprecated: use [ast.EnumTypeDefinition] instead.
type EnumTypeDefinition = ast.EnumTypeDefinition

// Deprecated: use [ast.EnumValueDefinition] instead.
type EnumValueDefinition = ast.EnumValueDefinition

// Deprecated: use [ast.Extension] instead.
type Extension = ast.Extension

// Deprecated: use [ast.FieldDefinition] instead.
type FieldDefinition = ast.FieldDefinition

// Deprecated: use [ast.FieldsDefinition] instead.
type FieldsDefinition = ast.FieldsDefinition

// Deprecated: use [ast.Fragment] instead.
type Fragment = ast.Fragment

// Deprecated: use [ast.InlineFragment] instead.
type InlineFragment = ast.InlineFragment

// Deprecated: use [ast.FragmentDefinition] instead.
type FragmentDefinition = ast.FragmentDefinition

// Deprecated: use [ast.FragmentSpread] instead.
type FragmentSpread = ast.FragmentSpread

// Deprecated: use [ast.FragmentList] instead.
type FragmentList = ast.FragmentList

// Deprecated: use [ast.InputValueDefinition] instead.
type InputValueDefinition = ast.InputValueDefinition

// Deprecated: use [ast.InputValueDefinitionList] instead.
type InputValueDefinitionList = ast.InputValueDefinitionList

// Deprecated: use [ast.InputObject] instead.
type InputObject = ast.InputObject

// Deprecated: use [ast.InterfaceTypeDefinition] instead.
type InterfaceTypeDefinition = ast.InterfaceTypeDefinition

// Deprecated: use [ast.ObjectTypeDefinition] instead.
type ObjectTypeDefinition = ast.ObjectTypeDefinition

// Deprecated: use [ast.ExecutableDefinition] instead.
type ExecutableDefinition = ast.ExecutableDefinition

// Deprecated: use [ast.OperationDefinition] instead.
type OperationDefinition = ast.OperationDefinition

// Deprecated: use [ast.OperationType] instead.
type OperationType = ast.OperationType

// Deprecated: use [ast.Selection] instead.
type Selection = ast.Selection

// Deprecated: use [ast.SelectionSet] instead.
type SelectionSet = ast.SelectionSet

// Deprecated: use [ast.Field] instead.
type Field = ast.Field

// Deprecated: use [ast.OperationList] instead.
type OperationList = ast.OperationList

// Deprecated: use [ast.ScalarTypeDefinition] instead.
type ScalarTypeDefinition = ast.ScalarTypeDefinition

// Deprecated: use [ast.TypeName] instead.
type TypeName = ast.TypeName

// Deprecated: use [ast.NamedType] instead.
type NamedType = ast.NamedType

// Deprecated: use [ast.Ident] instead.
type Ident = ast.Ident

// Deprecated: use [ast.Type] instead.
type Type = ast.Type

// Deprecated: use [ast.List] instead.
type List = ast.List

// Deprecated: use [ast.NonNull] instead.
type NonNull = ast.NonNull

// Deprecated: use [ast.Union] instead.
type Union = ast.Union

// Deprecated: use [ast.Value] instead.
type Value = ast.Value

// Deprecated: use [ast.PrimitiveValue] instead.
type PrimitiveValue = ast.PrimitiveValue

// Deprecated: use [ast.ListValue] instead.
type ListValue = ast.ListValue

// Deprecated: use [ast.ObjectValue] instead.
type ObjectValue = ast.ObjectValue

// Deprecated: use [ast.ObjectField] instead.
type ObjectField = ast.ObjectField

// Deprecated: use [ast.NullValue] instead.
type NullValue = ast.NullValue

// Deprecated: use [ast.Variable] instead.
type Variable = ast.Variable
