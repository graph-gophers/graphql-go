package resolvable

import (
	"bytes"
	"reflect"

	"github.com/graph-gophers/graphql-go/ast"
)

var disabledBufferPool = newBufferPool(0)

type Schema struct {
	*Meta
	ast.Schema
	Query                Resolvable
	Mutation             Resolvable
	Subscription         Resolvable
	QueryResolver        reflect.Value
	MutationResolver     reflect.Value
	SubscriptionResolver reflect.Value

	bufferPool Pool[*bytes.Buffer]
}

func (s *Schema) BufferPool() Pool[*bytes.Buffer] {
	if s.bufferPool == nil {
		return disabledBufferPool
	}

	return s.bufferPool
}

func newSchema(astSchema *ast.Schema, resolvers map[string]interface{}, query, mutation, subscription Resolvable, maxPooledBufferCap int) *Schema {
	var bufferPool Pool[*bytes.Buffer]
	if maxPooledBufferCap > 0 {
		bufferPool = newBufferPool(maxPooledBufferCap)
	}

	return &Schema{
		Meta:                 newMeta(astSchema),
		Schema:               *astSchema,
		QueryResolver:        reflect.ValueOf(resolvers[Query]),
		MutationResolver:     reflect.ValueOf(resolvers[Mutation]),
		SubscriptionResolver: reflect.ValueOf(resolvers[Subscription]),
		Query:                query,
		Mutation:             mutation,
		Subscription:         subscription,

		bufferPool: bufferPool,
	}
}
