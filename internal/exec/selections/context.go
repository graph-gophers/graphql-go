// Package selections is for internal use to share selection context between
// the execution engine and the public graphql package without creating an
// import cycle.
//
// The execution layer stores the flattened child selection set for the field
// currently being resolved. The public API converts this into user-friendly
// helpers (SelectedFieldNames, etc.).
package selections

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/graph-gophers/graphql-go/decode"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/exec/selected"
)

type ctxKey struct{}

// Lazy holds raw selections and computes the flattened, deduped name list once on demand.
type Lazy struct {
	raw     []selected.Selection
	once    sync.Once
	names   []string
	set     map[string]struct{}
	decoded map[string]map[reflect.Type]reflect.Value // path -> type -> value copy
}

// Args returns the argument map for the first occurrence of the provided
// dot-delimited field path under the current resolver. The boolean reports
// if a matching field was found. The returned map MUST NOT be mutated by
// callers (it is the internal map). Paths follow the same format produced by
// SelectedFieldNames (e.g. "books", "books.reviews").
func (l *Lazy) Args(path string) (map[string]interface{}, bool) {
	if l == nil || len(path) == 0 {
		return nil, false
	}
	// Fast path: ensure raw exists.
	for _, sel := range l.raw {
		if m, ok := matchArgsRecursive(sel, path, ""); ok {
			return m, true
		}
	}
	return nil, false
}

func matchArgsRecursive(sel selected.Selection, want, prefix string) (map[string]interface{}, bool) {
	switch s := sel.(type) {
	case *selected.SchemaField:
		name := s.Name
		if len(name) >= 2 && name[:2] == "__" { // skip meta
			return nil, false
		}
		cur := name
		if prefix != "" {
			cur = prefix + "." + name
		}
		if cur == want {
			return s.Args, true
		}
		for _, child := range s.Sels {
			if m, ok := matchArgsRecursive(child, want, cur); ok {
				return m, true
			}
		}
	case *selected.TypeAssertion:
		for _, child := range s.Sels {
			if m, ok := matchArgsRecursive(child, want, prefix); ok {
				return m, true
			}
		}
	case *selected.TypenameField:
		return nil, false
	}
	return nil, false
}

// Names returns the deduplicated child field names computing them once.
func (l *Lazy) Names() []string {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		seen := make(map[string]struct{}, len(l.raw))
		ordered := make([]string, 0, len(l.raw))
		collectNestedPaths(&ordered, seen, "", l.raw)
		l.names = ordered
		l.set = seen
	})
	out := make([]string, len(l.names))
	copy(out, l.names)
	return out
}

// Has reports if a field name is in the selection list.
func (l *Lazy) Has(name string) bool {
	if l == nil {
		return false
	}
	if l.set == nil { // ensure computed
		_ = l.Names()
	}
	_, ok := l.set[name]
	return ok
}

func collectNestedPaths(dst *[]string, seen map[string]struct{}, prefix string, sels []selected.Selection) {
	for _, sel := range sels {
		switch s := sel.(type) {
		case *selected.SchemaField:
			name := s.Name
			if len(name) >= 2 && name[:2] == "__" {
				continue
			}
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				*dst = append(*dst, path)
			}
			if len(s.Sels) > 0 {
				collectNestedPaths(dst, seen, path, s.Sels)
			}
		case *selected.TypeAssertion:
			collectNestedPaths(dst, seen, prefix, s.Sels)
		case *selected.TypenameField:
			continue
		}
	}
}

// With stores a lazy wrapper for selections in the context.
func With(ctx context.Context, sels []selected.Selection) context.Context {
	if len(sels) == 0 {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, &Lazy{raw: sels})
}

// FromContext retrieves the lazy wrapper (may be nil).
func FromContext(ctx context.Context) *Lazy {
	v, _ := ctx.Value(ctxKey{}).(*Lazy)
	return v
}

// DecodeArgsInto decodes the argument map for the dot path into dst (pointer to struct).
// Returns (true,nil) if decoded, (false,nil) if path missing. Caches per path+type.
func (l *Lazy) DecodeArgsInto(path string, dst interface{}) (bool, error) {
	if l == nil || dst == nil {
		return false, nil
	}
	args, ok := l.Args(path)
	if !ok || len(args) == 0 {
		return false, nil
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false, fmt.Errorf("destination must be non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("destination must point to struct")
	}
	rt := rv.Type()
	if l.decoded == nil {
		l.decoded = make(map[string]map[reflect.Type]reflect.Value)
	}
	if m := l.decoded[path]; m != nil {
		if cached, ok := m[rt]; ok {
			rv.Set(cached)
			return true, nil
		}
	}
	// decode
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		name := sf.Tag.Get("graphql")
		if name == "" {
			name = lowerFirst(sf.Name)
		}
		raw, present := args[name]
		if !present || raw == nil {
			continue
		}
		if err := assignArg(rv.Field(i), raw); err != nil {
			return false, fmt.Errorf("arg %s: %w", name, err)
		}
	}
	if l.decoded[path] == nil {
		l.decoded[path] = make(map[reflect.Type]reflect.Value)
	}
	// create a copy to cache so future mutations to dst by caller don't taint cache
	copyVal := reflect.New(rt).Elem()
	copyVal.Set(rv)
	l.decoded[path][rt] = copyVal
	return true, nil
}

func assignArg(dst reflect.Value, src interface{}) error {
	if !dst.IsValid() {
		return nil
	}
	// Support custom scalars implementing decode.Unmarshaler (pointer receiver).
	if dst.CanAddr() {
		if um, ok := dst.Addr().Interface().(decode.Unmarshaler); ok {
			if err := um.UnmarshalGraphQL(src); err != nil {
				return err
			}
			return nil
		}
	}
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.String, reflect.Bool, reflect.Float32, reflect.Float64:
		coerced, err := packer.UnmarshalInput(dst.Type(), src)
		if err != nil {
			return err
		}
		dst.Set(reflect.ValueOf(coerced))
	case reflect.Struct:
		m, ok := src.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map for struct, got %T", src)
		}
		for i := 0; i < dst.NumField(); i++ {
			sf := dst.Type().Field(i)
			if sf.PkgPath != "" { // unexported
				continue
			}
			name := sf.Tag.Get("graphql")
			if name == "" {
				name = lowerFirst(sf.Name)
			}
			if v, ok2 := m[name]; ok2 {
				if err := assignArg(dst.Field(i), v); err != nil {
					return err
				}
			}
		}
	case reflect.Slice:
		sv := reflect.ValueOf(src)
		if sv.Kind() != reflect.Slice {
			return fmt.Errorf("cannot convert %T to slice", src)
		}
		out := reflect.MakeSlice(dst.Type(), sv.Len(), sv.Len())
		for i := 0; i < sv.Len(); i++ {
			if err := assignArg(out.Index(i), sv.Index(i).Interface()); err != nil {
				return err
			}
		}
		dst.Set(out)
	case reflect.Ptr:
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return assignArg(dst.Elem(), src)
	default:
		// silently ignore unsupported kinds
	}
	return nil
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
