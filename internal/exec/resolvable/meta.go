package resolvable

import (
	"fmt"
	"reflect"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

var MetaSchema *Object
var MetaType *Object

func InitMeta(resolvers TypeToResolversMap, schemaType, typeType reflect.Type) {
	var err error
	b := newBuilder(schema.Meta, resolvers)

	metaSchema := schema.Meta.Types["__Schema"].(*schema.Object)
	MetaSchema, err = b.makeObjectExec(metaSchema, schemaType, false)
	if err != nil {
		panic(err)
	}

	metaType := schema.Meta.Types["__Type"].(*schema.Object)
	MetaType, err = b.makeObjectExec(metaType, typeType, false)
	if err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}
}

var MetaFieldTypename = Field{
	Field: schema.Field{
		Name: "__typename",
		Type: &common.NonNull{OfType: schema.Meta.Types["String"]},
	},
	TraceLabel: fmt.Sprintf("GraphQL field: __typename"),
}

var MetaFieldSchema = Field{
	Field: schema.Field{
		Name: "__schema",
		Type: schema.Meta.Types["__Schema"],
	},
	TraceLabel: fmt.Sprintf("GraphQL field: __schema"),
}

var MetaFieldType = Field{
	Field: schema.Field{
		Name: "__type",
		Type: schema.Meta.Types["__Type"],
	},
	TraceLabel: fmt.Sprintf("GraphQL field: __type"),
}
