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
	"sync"

	"github.com/graph-gophers/graphql-go/internal/exec/selected"
)

// ctxKey is an unexported unique type used as context key.
type ctxKey struct{}

// Lazy holds raw selections and computes the flattened, deduped name list once on demand.
type Lazy struct {
	raw   []selected.Selection
	once  sync.Once
	names []string
	set   map[string]struct{}
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
