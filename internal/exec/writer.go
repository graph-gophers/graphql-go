package exec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/schema"
)

// writeNode writes an execNode to the output buffer. It reports any errors encountered and returns true if
// the parent must become NULL too (error propagation).
func writeNode(r *Request, out *bytes.Buffer, n *execNode) bool {
	typ, nonNull := unwrapNonNull(n.typ)
	if n.err != nil {
		r.AddError(n.err)
		out.WriteString("null")
		return nonNull
	}
	if isNull(n.value) {
		out.WriteString("null")
		if nonNull {
			err := errors.Errorf("graphql: got nil for non-null %q", typ)
			err.Path = n.fullPath()
			r.AddError(err)
			return true
		}
		return false
	}
	switch typ := typ.(type) {
	case *schema.Scalar:
		return writeScalar(r, out, n)
	case *common.List:
		return writeList(r, out, n)
	case *schema.Object, *schema.Interface, *schema.Union:
		return writeObj(r, out, n)
	case *schema.Enum:
		return writeEnum(r, out, n, typ)
	default:
		panic(fmt.Sprintf("unknown schema type %T", typ))
	}
}

// writeScalar writes a graphQL scalar to the output buffer. It follows the writeNode semantic.
func writeScalar(r *Request, out *bytes.Buffer, n *execNode) bool {
	_, nonNull := unwrapNonNull(n.typ)
	w := newResetWriter(out)
	if err := json.NewEncoder(w).Encode(n.value.Interface()); err != nil {
		writeErr := errors.Errorf("json.Encode: %v", err)
		writeErr.Path = n.fullPath()
		r.AddError(writeErr)
		w.PropagateNull()
		return nonNull
	}
	if nonNull && w.IsNull() {
		err := errors.Errorf("graphql: got nil for non-null %q", n.typ)
		err.Path = n.fullPath()
		r.AddError(err)
		return true
	}
	return false
}

// writeList writes a GraphQL list to the output buffer. It follows the writeNode semantic.
func writeList(r *Request, out *bytes.Buffer, n *execNode) bool {
	w := newResetWriter(out)
	propNull := false
	w.WriteByte('[')
	for i, c := range n.children {
		if i > 0 {
			w.WriteByte(',')
		}
		if writeNode(r, w.Buffer, c) {
			propNull = true
		}
	}
	w.WriteByte(']')
	if propNull {
		w.PropagateNull()
		_, nonNull := unwrapNonNull(n.typ)
		return nonNull
	}
	return false
}

// writeObj writes a GraphQL object. It reports any error it encounters and follows the writeNode semantic.
func writeObj(r *Request, out *bytes.Buffer, n *execNode) bool {
	w := newResetWriter(out)
	propNull := false
	w.WriteByte('{')
	for i, c := range n.children {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('"')
		w.WriteString(c.field.Alias)
		w.WriteByte('"')
		w.WriteByte(':')
		if writeNode(r, w.Buffer, c) {
			propNull = true
		}
	}
	w.WriteByte('}')
	if propNull {
		w.PropagateNull()
		_, nonNull := unwrapNonNull(n.typ)
		return nonNull
	}
	return false
}

// writeEnum writes a graphQL enum to the output buffer. It follows the writeNode semantic.
func writeEnum(r *Request, out *bytes.Buffer, n *execNode, t *schema.Enum) bool {
	value := n.value
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	var stringer fmt.Stringer = value
	if s, ok := value.Interface().(fmt.Stringer); ok {
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
		err.Path = n.fullPath()
		r.AddError(err)
		out.WriteString("null")
		_, nonNull := unwrapNonNull(n.typ)
		return nonNull
	}
	out.WriteByte('"')
	out.WriteString(name)
	out.WriteByte('"')
	return false
}

// resetWriter is a writer that appends data to an existing bytes.Buffer. The PropagateNull method can
// be used to change the written data afterwards.
type resetWriter struct {
	*bytes.Buffer
	start int
}

// newResetWriter initializes a new reset-able writer.
func newResetWriter(out *bytes.Buffer) *resetWriter {
	return &resetWriter{Buffer: out, start: out.Len()}
}

// PropagateNull changes the data appended by this writer to "null", the JSON null value. Data that has
// been written to the buffer before is unaffected.
func (w *resetWriter) PropagateNull() {
	w.Truncate(w.start)
	w.WriteString("null")
}

// IsNull checks whatever the data written by this writer is the null value in JSON.
func (w *resetWriter) IsNull() bool {
	return bytes.Equal(w.Bytes()[w.start:], []byte("null"))
}

// objWriter is a streaming-able API for writing GraphQL objects similar to writeObj.
type objWriter struct {
	rw *resetWriter
	propNull bool
}

// newObjWriter initializes a new object writer.
func newObjWriter(out *bytes.Buffer) *objWriter {
	return &objWriter{rw: newResetWriter(out)}
}

// Write writes a single node (key/value pair) within the object.
func (w *objWriter) Write(r *Request, n *execNode) {
	if w.rw.Len() == w.rw.start {
		w.rw.WriteByte('{')
	} else {
		w.rw.WriteByte(',')
	}
	w.rw.WriteByte('"')
	w.rw.WriteString(n.field.Alias)
	w.rw.WriteByte('"')
	w.rw.WriteByte(':')

	if writeNode(r, w.rw.Buffer, n) {
		w.propNull = true
	}
}

// Flush finishes the object and propagates the NULL value if necessary.
func (w *objWriter) Flush() {
	if w.propNull {
		w.rw.PropagateNull()
		return
	}
	if w.rw.Len() == w.rw.start {
		w.rw.WriteByte('{')
	}
	w.rw.WriteByte('}')
}


