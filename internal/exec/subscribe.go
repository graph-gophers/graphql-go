package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/resolvable"
	"github.com/graph-gophers/graphql-go/internal/exec/selected"
	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
)

type Response struct {
	Data   json.RawMessage
	Errors []*errors.QueryError
}

func (r *Request) Subscribe(ctx context.Context, s *resolvable.Schema, op *query.Operation) <-chan *Response {
	var result reflect.Value
	var f *fieldToExec
	var err *errors.QueryError
	func() {
		defer r.handlePanic(ctx)

		sels := selected.ApplyOperation(&r.Request, s, op)
		var fields []*fieldToExec
		collectFieldsToResolve(sels, s, s.Resolver, &fields, make(map[string]*fieldToExec))

		// TODO: move this check into validation.Validate
		if len(fields) != 1 {
			err = errors.Errorf("%s", "can subscribe to at most one subscription at a time")
			return
		}
		f = fields[0]

		var in []reflect.Value
		if f.field.HasContext {
			in = append(in, reflect.ValueOf(ctx))
		}
		if f.field.ArgsPacker != nil {
			in = append(in, f.field.PackedArgs)
		}
		callOut := f.resolver.Method(f.field.MethodIndex).Call(in)
		result = callOut[0]

		if f.field.HasError && !callOut[1].IsNil() {
			resolverErr := callOut[1].Interface().(error)
			err = errors.Errorf("%s", resolverErr)
			err.ResolverError = resolverErr
		}
	}()

	if err != nil {
		if _, nonNullChild := f.field.Type.(*common.NonNull); nonNullChild {
			return sendAndReturnClosed(&Response{Errors: []*errors.QueryError{err}})
		}
		return sendAndReturnClosed(&Response{Data: []byte(fmt.Sprintf(`{"%s":null}`, f.field.Alias)), Errors: []*errors.QueryError{err}})
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return sendAndReturnClosed(&Response{Errors: []*errors.QueryError{errors.Errorf("%s", ctxErr)}})
	}

	c := make(chan *Response)
	// TODO: handle resolver nil channel better?
	if result == reflect.Zero(result.Type()) {
		close(c)
		return c
	}

	go func() {
		for {
			// Check subscription context
			chosen, resp, ok := reflect.Select([]reflect.SelectCase{
				{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(ctx.Done()),
				},
				{
					Dir:  reflect.SelectRecv,
					Chan: result,
				},
			})
			switch chosen {
			// subscription context done
			case 0:
				close(c)
				return
			// upstream received
			case 1:
				// upstream closed
				if !ok {
					close(c)
					return
				}

				subR := &Request{
					Request: selected.Request{
						Doc:    r.Request.Doc,
						Vars:   r.Request.Vars,
						Schema: r.Request.Schema,
					},
					Limiter: r.Limiter,
					Tracer:  r.Tracer,
					Logger:  r.Logger,
				}
				var out bytes.Buffer
				func() {
					// TODO: configurable timeout
					subCtx, cancel := context.WithTimeout(ctx, time.Second)
					defer cancel()

					// resolve response
					func() {
						defer subR.handlePanic(subCtx)

						var buf bytes.Buffer
						subR.execSelectionSet(subCtx, f.sels, f.field.Type, &pathSegment{nil, f.field.Alias}, s, resp, &buf)

						propagateChildError := false
						if _, nonNullChild := f.field.Type.(*common.NonNull); nonNullChild && resolvedToNull(&buf) {
							propagateChildError = true
						}

						if !propagateChildError {
							out.WriteString(fmt.Sprintf(`{"%s":`, f.field.Alias))
							out.Write(buf.Bytes())
							out.WriteString(`}`)
						}
					}()

					if err := subCtx.Err(); err != nil {
						c <- &Response{Errors: []*errors.QueryError{errors.Errorf("%s", err)}}
						return
					}

					// Send response within timeout
					// TODO: maybe block until sent?
					select {
					case <-subCtx.Done():
					case c <- &Response{Data: out.Bytes(), Errors: subR.Errs}:
					}
				}()
			}
		}
	}()

	return c
}

func sendAndReturnClosed(resp *Response) chan *Response {
	c := make(chan *Response, 1)
	c <- resp
	close(c)
	return c
}

type fieldToExec struct {
	field    *selected.SchemaField
	sels     []selected.Selection
	resolver reflect.Value
	out      *bytes.Buffer
}

func resolvedToNull(b *bytes.Buffer) bool {
	return bytes.Equal(b.Bytes(), []byte("null"))
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


func (r *Request) execSelections(ctx context.Context, sels []selected.Selection, path *pathSegment, s *resolvable.Schema, resolver reflect.Value, out *bytes.Buffer, serially bool) {
	async := !serially && selected.HasAsyncSel(sels)

	var fields []*fieldToExec
	collectFieldsToResolve(sels, s, resolver, &fields, make(map[string]*fieldToExec))

	if async {
		var wg sync.WaitGroup
		wg.Add(len(fields))
		for _, f := range fields {
			go func(f *fieldToExec) {
				defer wg.Done()
				defer r.handlePanic(ctx)
				f.out = new(bytes.Buffer)
				execNodeSelection(ctx, r, s, f, &pathSegment{path, f.field.Alias}, true)
			}(f)
		}
		wg.Wait()
	} else {
		for _, f := range fields {
			f.out = new(bytes.Buffer)
			execNodeSelection(ctx, r, s, f, &pathSegment{path, f.field.Alias}, true)
		}
	}

	out.WriteByte('{')
	for i, f := range fields {
		// If a non-nullable child resolved to null, an error was added to the
		// "errors" list in the response, so this field resolves to null.
		// If this field is non-nullable, the error is propagated to its parent.
		if _, ok := f.field.Type.(*common.NonNull); ok && resolvedToNull(f.out) {
			out.Reset()
			out.Write([]byte("null"))
			return
		}

		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('"')
		out.WriteString(f.field.Alias)
		out.WriteByte('"')
		out.WriteByte(':')
		out.Write(f.out.Bytes())
	}
	out.WriteByte('}')
}

func collectFieldsToResolve(sels []selected.Selection, s *resolvable.Schema, resolver reflect.Value, fields *[]*fieldToExec, fieldByAlias map[string]*fieldToExec) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *selected.SchemaField:
			field, ok := fieldByAlias[sel.Alias]
			if !ok { // validation already checked for conflict (TODO)
				field = &fieldToExec{field: sel, resolver: resolver}
				fieldByAlias[sel.Alias] = field
				*fields = append(*fields, field)
			}
			field.sels = append(field.sels, sel.Sels...)

		case *selected.TypenameField:
			sf := &selected.SchemaField{
				Field:       s.Meta.FieldTypename,
				Alias:       sel.Alias,
				FixedResult: reflect.ValueOf(typeOf(sel, resolver)),
			}
			*fields = append(*fields, &fieldToExec{field: sf, resolver: resolver})

		case *selected.TypeAssertion:
			out := resolver.Method(sel.MethodIndex).Call(nil)
			if !out[1].Bool() {
				continue
			}
			collectFieldsToResolve(sel.Sels, s, out[0], fields, fieldByAlias)

		default:
			panic("unreachable")
		}
	}
}

func execNodeSelection(ctx context.Context, r *Request, s *resolvable.Schema, f *fieldToExec, path *pathSegment, applyLimiter bool) {
	if applyLimiter {
		r.Limiter <- struct{}{}
	}

	var result reflect.Value
	var err *errors.QueryError

	traceCtx, finish := r.Tracer.TraceField(ctx, f.field.TraceLabel, f.field.TypeName, f.field.Name, !f.field.Async, f.field.Args)
	defer func() {
		finish(err)
	}()

	err = func() (err *errors.QueryError) {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				r.Logger.LogPanic(ctx, panicValue)
				err = makePanicError(panicValue)
				err.Path = path.toSlice()
			}
		}()

		if f.field.FixedResult.IsValid() {
			result = f.field.FixedResult
			return nil
		}

		if err := traceCtx.Err(); err != nil {
			return errors.Errorf("%s", err) // don't execute any more resolvers if context got cancelled
		}

		res := f.resolver
		if f.field.UseMethodResolver() {
			var in []reflect.Value
			if f.field.HasContext {
				in = append(in, reflect.ValueOf(traceCtx))
			}
			if f.field.ArgsPacker != nil {
				in = append(in, f.field.PackedArgs)
			}
			callOut := res.Method(f.field.MethodIndex).Call(in)
			result = callOut[0]
			if f.field.HasError && !callOut[1].IsNil() {
				resolverErr := callOut[1].Interface().(error)
				err := errors.Errorf("%s", resolverErr)
				err.Path = path.toSlice()
				err.ResolverError = resolverErr
				if ex, ok := callOut[1].Interface().(extensionser); ok {
					err.Extensions = ex.Extensions()
				}
				return err
			}
		} else {
			// TODO extract out unwrapping ptr logic to a common place
			if res.Kind() == reflect.Ptr {
				res = res.Elem()
			}
			result = res.FieldByIndex(f.field.FieldIndex)
		}
		return nil
	}()

	if applyLimiter {
		<-r.Limiter
	}

	if err != nil {
		// If an error occurred while resolving a field, it should be treated as though the field
		// returned null, and an error must be added to the "errors" list in the response.
		r.AddError(err)
		f.out.WriteString("null")
		return
	}

	r.execSelectionSet(traceCtx, f.sels, f.field.Type, path, s, result, f.out)
}

func (r *Request) execSelectionSet(ctx context.Context, sels []selected.Selection, typ common.Type, path *pathSegment, s *resolvable.Schema, resolver reflect.Value, out *bytes.Buffer) {
	t, nonNull := unwrapNonNull(typ)
	switch t := t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		// a reflect.Value of a nil interface will show up as an Invalid value
		if resolver.Kind() == reflect.Invalid || ((resolver.Kind() == reflect.Ptr || resolver.Kind() == reflect.Interface) && resolver.IsNil()) {
			// If a field of a non-null type resolves to null (either because the
			// function to resolve the field returned null or because an error occurred),
			// add an error to the "errors" list in the response.
			if nonNull {
				err := errors.Errorf("graphql: got nil for non-null %q", t)
				err.Path = path.toSlice()
				r.AddError(err)
			}
			out.WriteString("null")
			return
		}

		r.execSelections(ctx, sels, path, s, resolver, out, false)
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
		r.execList(ctx, sels, t, path, s, resolver, out)

	case *schema.Scalar:
		v := resolver.Interface()
		data, err := json.Marshal(v)
		if err != nil {
			panic(errors.Errorf("could not marshal %v: %s", v, err))
		}
		out.Write(data)

	case *schema.Enum:
		var stringer fmt.Stringer = resolver
		if s, ok := resolver.Interface().(fmt.Stringer); ok {
			stringer = s
		}
		name := stringer.String()
		var valid bool
		for _, v := range t.Values {
			if v.Name == name {
				valid = true
				break
			}
		}
		if !valid {
			err := errors.Errorf("Invalid value %s.\nExpected type %s, found %s.", name, t.Name, name)
			err.Path = path.toSlice()
			r.AddError(err)
			out.WriteString("null")
			return
		}
		out.WriteByte('"')
		out.WriteString(name)
		out.WriteByte('"')

	default:
		panic("unreachable")
	}
}

func (r *Request) execList(ctx context.Context, sels []selected.Selection, typ *common.List, path *pathSegment, s *resolvable.Schema, resolver reflect.Value, out *bytes.Buffer) {
	l := resolver.Len()
	entryouts := make([]bytes.Buffer, l)

	if selected.HasAsyncSel(sels) {
		var wg sync.WaitGroup
		wg.Add(l)
		for i := 0; i < l; i++ {
			go func(i int) {
				defer wg.Done()
				defer r.handlePanic(ctx)
				r.execSelectionSet(ctx, sels, typ.OfType, &pathSegment{path, i}, s, resolver.Index(i), &entryouts[i])
			}(i)
		}
		wg.Wait()
	} else {
		for i := 0; i < l; i++ {
			r.execSelectionSet(ctx, sels, typ.OfType, &pathSegment{path, i}, s, resolver.Index(i), &entryouts[i])
		}
	}

	_, listOfNonNull := typ.OfType.(*common.NonNull)

	out.WriteByte('[')
	for i, entryout := range entryouts {
		// If the list wraps a non-null type and one of the list elements
		// resolves to null, then the entire list resolves to null.
		if listOfNonNull && resolvedToNull(&entryout) {
			out.Reset()
			out.WriteString("null")
			return
		}

		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(entryout.Bytes())
	}
	out.WriteByte(']')
}
