package exec

import (
	"bytes"
	"encoding/json"
	"reflect"
	"unicode/utf8"
)

// largeValueSize is the size from which a value is kept as a separate
// segment of an output instead of being copied into its parent's buffer.
const largeValueSize = 64 << 10

// output accumulates the JSON encoding of part of a response.
//
// Small writes are appended to a contiguous buffer, as with a bytes.Buffer.
// Large values are referenced as separate segments instead, so that writing a
// child output into its parent does not copy them, and large strings are only
// escaped once, directly into the final response by bytes. This keeps the
// memory needed for a response with a huge scalar close to the size of the
// value plus the size of the response.
//
// Outputs are pooled. writeOutput detaches whatever it references from the
// child, so a child can always be released after being written to its parent.
type output struct {
	segs []segment    // precede tail
	tail bytes.Buffer // detached once it becomes a segment
}

// segment is either pre-encoded JSON or a string to encode as a JSON string.
type segment struct {
	json []byte
	str  string
	n    int // length of the encoded segment
}

func (o *output) Write(p []byte) (int, error) {
	return o.tail.Write(p)
}

func (o *output) WriteByte(c byte) error {
	return o.tail.WriteByte(c)
}

func (o *output) WriteString(s string) {
	o.tail.WriteString(s)
}

func (o *output) Len() int {
	n := o.tail.Len()
	for _, s := range o.segs {
		n += s.n
	}
	return n
}

func (o *output) Reset() {
	o.segs = nil
	o.tail.Reset()
}

func (o *output) appendSegment(s segment) {
	if o.tail.Len() > 0 {
		o.segs = append(o.segs, segment{json: o.tail.Bytes(), n: o.tail.Len()})
		o.tail = bytes.Buffer{}
	}
	o.segs = append(o.segs, s)
}

// writeOutput appends the contents of c. Small contents are copied. Large
// contents are referenced instead and detached from c, which is left empty, so
// that c can be reused without overwriting them.
func (o *output) writeOutput(c *output) {
	if len(c.segs) == 0 && c.tail.Len() < largeValueSize {
		o.tail.Write(c.tail.Bytes())
		return
	}
	for _, s := range c.segs {
		o.appendSegment(s)
	}
	if c.tail.Len() > 0 {
		o.appendSegment(segment{json: c.tail.Bytes(), n: c.tail.Len()})
	}
	c.segs = nil
	c.tail = bytes.Buffer{}
}

// bytes returns the complete output. Large strings are escaped directly into
// the result, and segments are released as soon as they have been copied, so
// o must not be used afterwards.
func (o *output) bytes() []byte {
	if len(o.segs) == 0 {
		return o.tail.Bytes()
	}
	b := make([]byte, 0, o.Len())
	for i, s := range o.segs {
		if s.json != nil {
			b = append(b, s.json...)
		} else {
			b = appendJSONString(b, s.str)
		}
		o.segs[i] = segment{}
	}
	o.segs = nil
	return append(b, o.tail.Bytes()...)
}

// writeScalar writes v exactly as json.Marshal would encode it.
func (o *output) writeScalar(v any) error {
	switch v := v.(type) {
	case string:
		if len(v) >= largeValueSize {
			o.appendSegment(segment{str: v, n: jsonStringLen(v)})
			return nil
		}
		o.tail.Grow(len(v) + 2)
		o.tail.Write(appendJSONString(o.tail.AvailableBuffer(), v))
		return nil
	case json.Marshaler:
		if rv := reflect.ValueOf(v); rv.Kind() != reflect.Pointer || !rv.IsNil() {
			return o.writeMarshaler(v)
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	o.tail.Write(data)
	return nil
}

// writeMarshaler writes m like json.Marshal does: compacted, with <, >, &,
// U+2028 and U+2029 escaped, and with the same errors.
func (o *output) writeMarshaler(m json.Marshaler) error {
	b, err := m.MarshalJSON()
	if err == nil {
		dst := &o.tail
		var large bytes.Buffer
		if len(b) >= largeValueSize {
			dst = &large // json.Compact grows it to exactly len(b)
		}
		if extra := htmlEscapeGrowth(b); extra == 0 {
			err = json.Compact(dst, b)
		} else {
			var compacted bytes.Buffer
			if err = json.Compact(&compacted, b); err == nil {
				dst.Grow(compacted.Len() + extra)
				json.HTMLEscape(dst, compacted.Bytes())
			}
		}
		if err == nil && dst == &large {
			o.appendSegment(segment{json: large.Bytes(), n: large.Len()})
		}
	}
	if err != nil {
		// The error json.Marshal reports differs between the v1 and v2 backed
		// encoding/json, so let it report the error. This only calls
		// m.MarshalJSON again when it failed or returned invalid JSON.
		if _, mErr := json.Marshal(m); mErr != nil {
			return mErr
		}
		return &json.MarshalerError{Type: reflect.TypeOf(m), Err: err}
	}
	return nil
}

// htmlEscapeGrowth returns by how many bytes json.HTMLEscape grows b.
func htmlEscapeGrowth(b []byte) int {
	n := 0
	for i, c := range b {
		switch {
		case c == '<' || c == '>' || c == '&':
			n += len(`\u003c`) - 1
		case c == 0xE2 && i+2 < len(b) && b[i+1] == 0x80 && b[i+2]&^1 == 0xA8: // U+2028 or U+2029
			n += len(`\u2028`) - len("\u2028")
		}
	}
	return n
}

// htmlSafeSet holds the ASCII characters that json.Marshal does not escape
// in strings.
var htmlSafeSet = func() (set [utf8.RuneSelf]bool) { //nolint:gochecknoglobals
	for c := ' '; c < utf8.RuneSelf; c++ {
		set[c] = c != '"' && c != '\\' && c != '<' && c != '>' && c != '&'
	}
	return set
}()

const hex = "0123456789abcdef"

// invalidUTF8 is what json.Marshal encodes each invalid UTF-8 byte in a string
// as. It is the escaped `\ufffd` for encoding/json v1, but the raw U+FFFD
// character when encoding/json is backed by v2 (GOEXPERIMENT=jsonv2, the
// default since Go 1.27).
var invalidUTF8 = func() string { //nolint:gochecknoglobals
	b, err := json.Marshal("\xff")
	if err != nil {
		panic(err)
	}
	return string(b[1 : len(b)-1])
}()

// appendJSONString appends s encoded as a JSON string, exactly like
// json.Marshal. It is adapted from appendString in encoding/json with
// escapeHTML set.
func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] {
				i++
				continue
			}
			dst = append(dst, s[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			dst = append(dst, s[start:i]...)
			dst = append(dst, invalidUTF8...)
			i += size
			start = i
			continue
		}
		if c == '\u2028' || c == '\u2029' {
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, s[start:]...)
	dst = append(dst, '"')
	return dst
}

// jsonStringLen returns len(appendJSONString(nil, s)) without encoding s.
func jsonStringLen(s string) int {
	n := len(`""`)
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			switch {
			case htmlSafeSet[b]:
				n++
			case b == '\\' || b == '"' || b == '\b' || b == '\f' || b == '\n' || b == '\r' || b == '\t':
				n += 2
			default:
				n += len(`\u0000`)
			}
			i++
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case c == utf8.RuneError && size == 1:
			n += len(invalidUTF8)
		case c == '\u2028' || c == '\u2029':
			n += len(`\u2028`)
		default:
			n += size
		}
		i += size
	}
	return n
}
