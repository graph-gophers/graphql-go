package graphql

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/internal/validation"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/log"
	"github.com/graph-gophers/graphql-go/resolvers"
	"github.com/graph-gophers/graphql-go/trace"
)

type Engine struct {
	schema           *schema.Schema
	MaxDepth         int
	MaxParallelism   int
	Tracer           trace.Tracer
	ValidationTracer trace.ValidationTracer
	Logger           log.Logger
	ResolverFactory  resolvers.ResolverFactory
	Root             interface{}
}

type EngineRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type EngineResponse struct {
	Data       json.RawMessage        `json:"data,omitempty"`
	Errors     []*errors.QueryError   `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func CreateEngine(schemaText string) (*Engine, error) {

	engine := Engine{
		schema:           schema.New(),
		Tracer:           trace.NoopTracer{},
		MaxParallelism:	  10,
		MaxDepth:         50,
		ValidationTracer: trace.NoopValidationTracer{},
		Logger:           &log.DefaultLogger{},
		ResolverFactory:  resolvers.DynamicResolverFactory(),
	}

	err := engine.schema.Parse(schemaText)
	if err != nil {
		return nil, err
	}

	return &engine, nil
}

// Execute the given request.
func (engine *Engine) Execute(ctx context.Context, request *EngineRequest, root interface{}) *Response {

	doc, qErr := query.Parse(request.Query)
	if qErr != nil {
		return &Response{Errors: []*errors.QueryError{qErr}}
	}

	validationFinish := engine.ValidationTracer.TraceValidation()
	errs := validation.Validate(engine.schema, doc, engine.MaxDepth)
	validationFinish(errs)

	if len(errs) != 0 {
		return &Response{Errors: errs}
	}

	op, err := getOperation(doc, request.OperationName)
	if err != nil {
		return &Response{Errors: []*errors.QueryError{errors.Errorf("%s", err)}}
	}

	varTypes := make(map[string]*introspection.Type)
	for _, v := range op.Vars {
		t, err := common.ResolveType(v.Type, engine.schema.Resolve)
		if err != nil {
			return &Response{Errors: []*errors.QueryError{err}}
		}
		varTypes[v.Name.Name] = introspection.WrapType(t)
	}

	if root == nil {
		root = engine.Root
	}

	traceContext, finish := engine.Tracer.TraceQuery(ctx, request.Query, request.OperationName, request.Variables, varTypes)
	out := bytes.Buffer{}

	r := exec.Execution{
		Schema:          engine.schema,
		Tracer:          engine.Tracer,
		Logger:          engine.Logger,
		ResolverFactory: engine.ResolverFactory,
		Doc:             doc,
		Operation:       op,
		Vars:            request.Variables,
		VarTypes:        varTypes,
		Limiter:         make(chan byte, engine.MaxParallelism),
		Context:         traceContext,
		Root:            root,
		Out:             bufio.NewWriter(&out),
	}

	errs = r.Execute()
	finish(errs)

	if(len(errs) > 0 ) {
		return &Response{
			Errors: errs,
		}
	}

	return &Response{
		Data:   out.Bytes(),
	}
}
