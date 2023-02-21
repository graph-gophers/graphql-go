package resolvable

import (
	"reflect"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/introspection"
)

// Meta defines the details of the metadata schema for introspection.
type Meta struct {
	FieldSchema   Field
	FieldType     Field
	FieldTypename Field
	FieldService  Field
	Schema        *Object
	Type          *Object
	Service       *Object
}

func newMeta(s *ast.Schema) *Meta {
	var err error
	b := newBuilder(s, nil, false)

	metaSchema := s.Types["__Schema"].(*ast.ObjectTypeDefinition)
	so, err := b.makeObjectExec(metaSchema.Name, metaSchema.Fields, nil, nil, false, reflect.TypeOf(&introspection.Schema{}))
	if err != nil {
		panic(err)
	}

	metaType := s.Types["__Type"].(*ast.ObjectTypeDefinition)
	t, err := b.makeObjectExec(metaType.Name, metaType.Fields, nil, nil, false, reflect.TypeOf(&introspection.Type{}))
	if err != nil {
		panic(err)
	}

	metaService := s.Types["_Service"].(*ast.ObjectTypeDefinition)
	sv, err := b.makeObjectExec(metaService.Name, metaService.Fields, nil, nil, false, reflect.TypeOf(&introspection.Service{}))
	if err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}

	fieldTypename := Field{
		FieldDefinition: ast.FieldDefinition{
			Name: "__typename",
			Type: &ast.NonNull{OfType: s.Types["String"]},
		},
		TraceLabel: "GraphQL field: __typename",
	}

	fieldSchema := Field{
		FieldDefinition: ast.FieldDefinition{
			Name: "__schema",
			Type: s.Types["__Schema"],
		},
		TraceLabel: "GraphQL field: __schema",
	}

	fieldType := Field{
		FieldDefinition: ast.FieldDefinition{
			Name: "__type",
			Type: s.Types["__Type"],
		},
		TraceLabel: "GraphQL field: __type",
	}

	fieldService := Field{
		FieldDefinition: ast.FieldDefinition{
			Name: "_service",
			Type: s.Types["_Service"],
		},
		TraceLabel: "GraphQL field: _service",
	}

	return &Meta{
		FieldSchema:   fieldSchema,
		FieldTypename: fieldTypename,
		FieldType:     fieldType,
		FieldService:  fieldService,
		Schema:        so,
		Type:          t,
		Service:       sv,
	}
}
