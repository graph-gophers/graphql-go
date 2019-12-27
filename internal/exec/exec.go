package exec

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
	"github.com/graph-gophers/graphql-go/internal/exec/selected"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/log"
	"github.com/graph-gophers/graphql-go/trace"
)

type Request struct {
	selected.Request
	Limiter chan struct{}
	Tracer  trace.Tracer
	Logger  log.Logger
}

func (r *Request) handlePanic(ctx context.Context) {
	if value := recover(); value != nil {
		r.Logger.LogPanic(ctx, value)
		r.AddError(makePanicError(value))
	}
}

type extensionser interface {
	Extensions() map[string]interface{}
}

func makePanicError(value interface{}) *errors.QueryError {
	return errors.Errorf("graphql: panic occurred: %v", value)
}

func (r *Request) process(ctx context.Context, nodes []*execNode, s *resolvable.Schema) {
	queue := nodes
	for len(queue) > 0 {
		batch := queue
		queue = nil

		// run the whole batch (everything from the current level) concurrently.
		result := make(chan *execNode, len(batch))
		for _, n := range batch {
			go func(n *execNode) {
				if n.field != nil {
					r.resolveNode(ctx, n, true)
					expandNode(n, s)
				}
				result <- n
			}(n)
		}

		// wait for the batch to complete and refill the queue.
		for range batch {
			n := <-result
			queue = append(queue, n.children...)
		}
	}
}

func (r *Request) Execute(ctx context.Context, s *resolvable.Schema, op *query.Operation) ([]byte, []*errors.QueryError) {
	out := &bytes.Buffer{}

	func() {
		defer r.handlePanic(ctx)
		sels := selected.ApplyOperation(&r.Request, s, op)
		nodes := collectNodes(sels, s, s.Resolver, nil, make(map[string]*execNode))

		w := newObjWriter(out)
		defer w.Flush()

		if op.Type == query.Mutation {
			// process mutations sequentially.
			for _, n := range nodes {
				r.process(ctx, []*execNode{n}, s)
				w.Write(r, n)
			}
		} else {
			r.process(ctx, nodes, s)
			for _, n := range nodes {
				w.Write(r, n)
			}
		}
	}()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return out.Bytes(), r.Errs
}

// execNode is used to build a tree structure that closely assembles the returned data. This in-memory representation
// allows nodes to be resolved in a different order (e.g. concurrently or breath-first) than they are printed (which
// is always depth-first).
type execNode struct {
	label    interface{}           // label for the full path (used in errors and debug output)
	parent   *execNode             // parent node
	children []*execNode           // child nodes
	typ      common.Type           // GraphQL type of the node
	field    *selected.SchemaField // field information (might be nil within lists)
	sels     []selected.Selection  // selected fields
	resolver reflect.Value         // resolver for resolving the selected fields
	value    reflect.Value         // resolved value
	err      *errors.QueryError    // error while resolving the value
}

// fullPath returns the full path of the node. This path is included in all graphQL error messages.
func (n *execNode) fullPath() []interface{} {
	if n == nil {
		return nil
	}
	return append(n.parent.fullPath(), n.label)
}

// add appends additional children to this node.
func (n *execNode) add(cs []*execNode) {
	for i := range cs {
		cs[i].parent = n
	}
	n.children = append(n.children, cs...)
}

// collectNodes transforms a selection into a set of nodes. Typename fields are resolved by adding a
// appropriate SchemaField to the selection and type assertions are resolved by adding additional fields to the
// targeted selections.
func collectNodes(sels []selected.Selection, s *resolvable.Schema, resolver reflect.Value, nodes []*execNode, nodeByAlias map[string]*execNode) []*execNode {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *selected.SchemaField:
			node, ok := nodeByAlias[sel.Alias]
			if !ok { // validation already checked for conflict (TODO)
				node = &execNode{label: sel.Alias, field: sel, resolver: resolver, typ: sel.Type}
				nodeByAlias[sel.Alias] = node
				nodes = append(nodes, node)
			}
			node.sels = append(node.sels, sel.Sels...)

		case *selected.TypenameField:
			sf := &selected.SchemaField{
				Field:       s.Meta.FieldTypename,
				Alias:       sel.Alias,
				FixedResult: reflect.ValueOf(typeOf(sel, resolver)),
			}
			nodes = append(nodes, &execNode{label: sel.Alias, field: sf, resolver: resolver, typ: sf.Type})

		case *selected.TypeAssertion:
			out := resolver.Method(sel.MethodIndex).Call(nil)
			if !out[1].Bool() {
				continue
			}
			nodes = collectNodes(sel.Sels, s, out[0], nodes, nodeByAlias)

		default:
			panic("unreachable")
		}
	}
	return nodes
}

func typeOf(tf *selected.TypenameField, resolver reflect.Value) string {
	if len(tf.TypeAssertions) == 0 {
		return tf.Name
	}
	for name, a := range tf.TypeAssertions {
		out := resolver.Method(a.MethodIndex).Call(nil)
		if out[1].Bool() {
			return name
		}
	}
	return ""
}

func (r *Request) resolveNode(ctx context.Context, n *execNode, applyLimiter bool) {
	if applyLimiter {
		r.Limiter <- struct{}{}
		defer func() {
			<-r.Limiter
		}()
	}

	n.value, n.err = func() (result reflect.Value, err *errors.QueryError) {
		traceCtx, finish := r.Tracer.TraceField(ctx, n.field.TraceLabel, n.field.TypeName, n.field.Name, !n.field.Async, n.field.Args)
		defer func() {
			finish(err)
		}()

		defer func() {
			if panicValue := recover(); panicValue != nil {
				r.Logger.LogPanic(ctx, panicValue)
				err = makePanicError(panicValue)
				err.Path = n.fullPath()
			}
		}()

		if n.field.FixedResult.IsValid() {
			return n.field.FixedResult, nil
		}

		if err := traceCtx.Err(); err != nil {
			return reflect.Value{}, errors.Errorf("%s", err) // don't execute any more resolvers if context got cancelled
		}

		res := n.resolver
		if isNull(res) {
			return
		}
		if n.field.UseMethodResolver() {
			var in []reflect.Value
			if n.field.HasContext {
				in = append(in, reflect.ValueOf(traceCtx))
			}
			if n.field.ArgsPacker != nil {
				in = append(in, n.field.PackedArgs)
			}
			callOut := res.Method(n.field.MethodIndex).Call(in)
			result = callOut[0]
			if n.field.HasError && !callOut[1].IsNil() {
				resolverErr := callOut[1].Interface().(error)
				err := errors.Errorf("%s", resolverErr)
				err.ResolverError = resolverErr
				err.Path = n.fullPath()
				if ex, ok := resolverErr.(extensionser); ok {
					err.Extensions = ex.Extensions()
				}
				return reflect.Value{}, err
			}
			return result, nil
		}
		// TODO extract out unwrapping ptr logic to a common place
		if res.Kind() == reflect.Ptr {
			res = res.Elem()
		}
		return res.FieldByIndex(n.field.FieldIndex), nil
	}()
}

// expandNode adds the next level of unresolved children to a node n. This function must be called after
// n itself has been resolved.
func expandNode(n *execNode, s *resolvable.Schema) {
	t, _ := unwrapNonNull(n.typ)
	switch t := t.(type) {
	case *schema.Scalar, *schema.Enum:
		// nothing to do.
	case *schema.Object, *schema.Interface, *schema.Union:
		if isNull(n.value) {
			return
		}
		children := collectNodes(n.sels, s, n.value, nil, make(map[string]*execNode))
		n.add(children)
	case *common.List:
		if isNull(n.value) {
			return
		}
		value := n.value
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		children := make([]*execNode, value.Len())
		for i := range children {
			children[i] = &execNode{label: i, value: value.Index(i), typ:   t.OfType}
			grandChildren := collectNodes(n.sels, s, children[i].value, nil, make(map[string]*execNode))
			children[i].add(grandChildren)
		}
		n.add(children)
	default:
		panic(fmt.Sprintf("unknown schema type %T", t))
	}
}

// unwrapNonNull removes the not-null type annotation of a type and returns the original type. The second return
// value is true if a not-null annotation has been removed.
func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}

// isNull checks whatever a reflect.Value is invalid or nil.
func isNull(v reflect.Value) bool {
	k := v.Kind()
	return k == reflect.Invalid || ((k == reflect.Ptr || k == reflect.Interface) && v.IsNil())
}
