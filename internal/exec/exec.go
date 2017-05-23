package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/exec/selected"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/log"
	"github.com/neelance/graphql-go/trace"
)

type Request struct {
	selected.Request
	Limiter chan struct{}
	Tracer  trace.Tracer
	Logger  log.Logger
}

type fieldResult struct {
	name  string
	value []byte
}

func (r *Request) handlePanic(ctx context.Context) {
	if value := recover(); value != nil {
		r.Logger.LogPanic(ctx, value)
		r.AddError(makePanicError(value))
	}
}

func makePanicError(value interface{}) *errors.QueryError {
	return errors.Errorf("graphql: panic occurred: %v", value)
}

func (r *Request) Execute(ctx context.Context, s *resolvable.Schema, op *query.Operation) ([]byte, []*errors.QueryError) {
	var out bytes.Buffer
	func() {
		defer r.handlePanic(ctx)
		sels := selected.ApplyOperation(&r.Request, s, op)
		r.execSelections(ctx, sels, nil, s.Resolver, &out, op.Type == query.Mutation)
	}()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return out.Bytes(), r.Errs
}

type fieldToExec struct {
	field  *selected.Field
	sels   []selected.SubSelection
	parent reflect.Value
	out    *bytes.Buffer
}

func (r *Request) execSelections(ctx context.Context, sels []*selected.Field, path *pathSegment, resolver reflect.Value, out *bytes.Buffer, serially bool) {
	async := !serially && selected.HasAsyncSel1(sels)

	var fields []*fieldToExec
	collectFieldsToResolve(sels, resolver, &fields, make(map[string]*fieldToExec))

	if async {
		var wg sync.WaitGroup
		wg.Add(len(fields))
		for _, f := range fields {
			go func(f *fieldToExec) {
				defer wg.Done()
				defer r.handlePanic(ctx)
				f.out = new(bytes.Buffer)
				execFieldSelection(ctx, r, f, &pathSegment{path, f.field.Alias}, true)
			}(f)
		}
		wg.Wait()
	}

	out.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('"')
		out.WriteString(f.field.Alias)
		out.WriteByte('"')
		out.WriteByte(':')
		if async {
			out.Write(f.out.Bytes())
			continue
		}
		f.out = out
		execFieldSelection(ctx, r, f, &pathSegment{path, f.field.Alias}, false)
	}
	out.WriteByte('}')
}

func collectFieldsToResolve(sels []*selected.Field, resolver reflect.Value, fields *[]*fieldToExec, fieldByAlias map[string]*fieldToExec) {
	for _, sel := range sels {
		field, ok := fieldByAlias[sel.Alias]
		if !ok { // validation already checked for conflict
			field = &fieldToExec{field: sel, parent: resolver}
			fieldByAlias[sel.Alias] = field
			*fields = append(*fields, field)
		}
		if sel.SubSel != nil {
			field.sels = append(field.sels, sel.SubSel)
		}
	}
}

func execFieldSelection(ctx context.Context, r *Request, f *fieldToExec, path *pathSegment, applyLimiter bool) {
	if applyLimiter {
		r.Limiter <- struct{}{}
	}

	var result reflect.Value
	var err *errors.QueryError
	traceCtx, finish := r.Tracer.TraceField(ctx, f.field.TraceLabel, f.field.TypeName, f.field.Name, !f.field.Async, f.field.Args)
	defer func() {
		finish(err)
	}()

	result, err = func() (result reflect.Value, err *errors.QueryError) {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				r.Logger.LogPanic(ctx, panicValue)
				err = makePanicError(panicValue)
				err.Path = path.toSlice()
			}
		}()

		if err := traceCtx.Err(); err != nil {
			return reflect.Value{}, errors.Errorf("%s", err) // don't execute any more resolvers if context got cancelled
		}

		if f.field.FixedResult.IsValid() {
			return f.field.FixedResult, nil
		}

		result, resolverErr := f.field.Resolver(traceCtx, f.parent)
		if resolverErr != nil {
			err := errors.Errorf("%s", resolverErr)
			err.Path = path.toSlice()
			err.ResolverError = resolverErr
			return reflect.Value{}, err
		}
		return result, nil
	}()

	if applyLimiter {
		<-r.Limiter
	}

	if err != nil {
		r.AddError(err)
		f.out.WriteString("null") // TODO handle non-nil
		return
	}

	r.execSelectionSet(traceCtx, f.sels, f.field.Type, path, result, f.out)
}

func (r *Request) execSelectionSet(ctx context.Context, sels []selected.SubSelection, typ common.Type, path *pathSegment, resolver reflect.Value, out *bytes.Buffer) {
	t, nonNull := unwrapNonNull(typ)
	switch t := t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		if resolver.Kind() == reflect.Ptr && resolver.IsNil() {
			if nonNull {
				panic(errors.Errorf("got nil for non-null %q", t))
			}
			out.WriteString("null")
			return
		}

		if resolver.Kind() == reflect.Interface {
			resolver = resolver.Elem()
		}
		var fieldSels []*selected.Field
		for _, subSel := range sels {
			fieldSels = append(fieldSels, subSel.Sels(resolver.Type())...)
		}
		r.execSelections(ctx, fieldSels, path, resolver, out, false)
		return
	}

	if !nonNull {
		if resolver.IsNil() {
			out.WriteString("null")
			return
		}
		resolver = resolver.Elem()
	}

	switch t := t.(type) {
	case *common.List:
		l := resolver.Len()

		if selected.HasAsyncSel(sels) {
			var wg sync.WaitGroup
			wg.Add(l)
			entryouts := make([]bytes.Buffer, l)
			for i := 0; i < l; i++ {
				go func(i int) {
					defer wg.Done()
					defer r.handlePanic(ctx)
					r.execSelectionSet(ctx, sels, t.OfType, &pathSegment{path, i}, resolver.Index(i), &entryouts[i])
				}(i)
			}
			wg.Wait()

			out.WriteByte('[')
			for i, entryout := range entryouts {
				if i > 0 {
					out.WriteByte(',')
				}
				out.Write(entryout.Bytes())
			}
			out.WriteByte(']')
			return
		}

		out.WriteByte('[')
		for i := 0; i < l; i++ {
			if i > 0 {
				out.WriteByte(',')
			}
			r.execSelectionSet(ctx, sels, t.OfType, &pathSegment{path, i}, resolver.Index(i), out)
		}
		out.WriteByte(']')

	case *schema.Scalar:
		v := resolver.Interface()
		data, err := json.Marshal(v)
		if err != nil {
			panic(errors.Errorf("could not marshal %v", v))
		}
		out.Write(data)

	case *schema.Enum:
		out.WriteByte('"')
		out.WriteString(resolver.String())
		out.WriteByte('"')

	default:
		panic("unreachable")
	}
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}

type marshaler interface {
	MarshalJSON() ([]byte, error)
}

type pathSegment struct {
	parent *pathSegment
	value  interface{}
}

func (p *pathSegment) toSlice() []interface{} {
	if p == nil {
		return nil
	}
	return append(p.parent.toSlice(), p.value)
}
