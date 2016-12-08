package graphql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"reflect"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/exec"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

const OpenTracingTagQuery = "graphql.query"
const OpenTracingTagOperationName = "graphql.operationName"
const OpenTracingTagVariables = "graphql.variables"

const OpenTracingTagType = "graphql.type"
const OpenTracingTagField = "graphql.field"
const OpenTracingTagTrivial = "graphql.trivial"
const OpenTracingTagArgsPrefix = "graphql.args."
const OpenTracingTagError = "graphql.error"

type ID string

// compile time check
var (
	_ sql.Scanner   = (*ID)(nil)
	_ driver.Valuer = (*ID)(nil)
)

func (id *ID) Scan(src interface{}) error {
	switch t := src.(type) {
	case string:
		*id = ID(t)
	case int64:
		*id = ID(strconv.FormatInt(t, 10))
	case []byte:
		*id = ID(t)
	case nil:
		// do nothing
	default:
		return fmt.Errorf("wrong type")
	}
	return nil
}

func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}

func ParseSchema(schemaString string, resolver interface{}) (*Schema, error) {
	b := New()
	if err := b.Parse(schemaString); err != nil {
		return nil, err
	}
	return b.ApplyResolver(resolver)
}

func MustParseSchema(schemaString string, resolver interface{}) *Schema {
	s, err := ParseSchema(schemaString, resolver)
	if err != nil {
		panic(err)
	}
	return s
}

type SchemaBuilder struct {
	schema *schema.Schema
}

func New() *SchemaBuilder {
	s := schema.New()
	exec.AddBuiltinScalars(s)
	exec.AddCustomScalar(s, "ID", reflect.TypeOf(ID("")), func(input interface{}) (interface{}, error) {
		switch input := input.(type) {
		case ID:
			return input, nil
		case string:
			return ID(input), nil
		default:
			return nil, fmt.Errorf("wrong type")
		}
	})
	return &SchemaBuilder{
		schema: s,
	}
}

func (b *SchemaBuilder) Parse(schemaString string) error {
	return b.schema.Parse(schemaString)
}

func (b *SchemaBuilder) AddCustomScalar(name string, scalar *ScalarConfig) {
	exec.AddCustomScalar(b.schema, name, scalar.ReflectType, scalar.CoerceInput)
}

func (b *SchemaBuilder) ApplyResolver(resolver interface{}) (*Schema, error) {
	e, err2 := exec.Make(b.schema, resolver)
	if err2 != nil {
		return nil, err2
	}
	return &Schema{
		schema: b.schema,
		exec:   e,
	}, nil
}

func (b *SchemaBuilder) ToJSON() ([]byte, error) {
	result, err := exec.IntrospectSchema(b.schema)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(result, "", "\t")
}

type Schema struct {
	schema *schema.Schema
	exec   *exec.Exec
}

type Response struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []*errors.QueryError   `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (s *Schema) Exec(ctx context.Context, queryString string, operationName string, variables map[string]interface{}) *Response {
	document, err := query.Parse(queryString, s.schema.Resolve)
	if err != nil {
		return &Response{
			Errors: []*errors.QueryError{err},
		}
	}

	span, subCtx := opentracing.StartSpanFromContext(ctx, "GraphQL request")
	span.SetTag(OpenTracingTagQuery, queryString)
	if operationName != "" {
		span.SetTag(OpenTracingTagOperationName, operationName)
	}
	if len(variables) != 0 {
		span.SetTag(OpenTracingTagVariables, variables)
	}
	defer span.Finish()

	data, errs := exec.ExecuteRequest(subCtx, s.exec, document, operationName, variables)
	if len(errs) != 0 {
		ext.Error.Set(span, true)
		span.SetTag(OpenTracingTagError, errs)
	}
	return &Response{
		Data:   data,
		Errors: errs,
	}
}

type ScalarConfig struct {
	ReflectType reflect.Type
	CoerceInput func(input interface{}) (interface{}, error)
}
