// Package cache implements caching of GraphQL requests by allowing resolvers to provide hints about their cacheability,
// which can be used by the transport handlers (e.g. HTTP) to provide caching indicators in the response.
package cache

import (
	"context"
	"fmt"
	"time"
)

type ctxKey string

const (
	hintsKey ctxKey = "hints"
)

type scope int

// Cache control scopes.
const (
	ScopePublic scope = iota
	ScopePrivate
)

const (
	hintsBuffer = 20
)

// Hint defines a hint as to how long something should be cached for.
type Hint struct {
	MaxAge *time.Duration
	Scope  scope
}

// String resolves the HTTP Cache-Control value of the Hint.
func (h Hint) String() string {
	var s string
	switch h.Scope {
	case ScopePublic:
		s = "public"
	case ScopePrivate:
		s = "private"
	}
	return fmt.Sprintf("%s, max-age=%d", s, int(h.MaxAge.Seconds()))
}

// TTL defines the cache duration.
func TTL(d time.Duration) *time.Duration {
	return &d
}

// AddHint applies a caching hint to the request context.
func AddHint(ctx context.Context, hint Hint) {
	c := hints(ctx)
	if c == nil {
		return
	}
	c <- hint
}

// Hintable extends the context with the ability to add cache hints.
func Hintable(ctx context.Context) (hintCtx context.Context, hint <-chan Hint, done func()) {
	hints := make(chan Hint, hintsBuffer)
	h := make(chan Hint)
	go func() {
		h <- resolve(hints)
	}()
	done = func() {
		close(hints)
	}
	return context.WithValue(ctx, hintsKey, hints), h, done
}

func hints(ctx context.Context) chan Hint {
	h, ok := ctx.Value(hintsKey).(chan Hint)
	if !ok {
		return nil
	}
	return h
}

func resolve(hints <-chan Hint) Hint {
	var minAge *time.Duration
	s := ScopePublic
	for h := range hints {
		if h.Scope == ScopePrivate {
			s = h.Scope
		}
		if h.MaxAge != nil && (minAge == nil || *h.MaxAge < *minAge) {
			minAge = h.MaxAge
		}
	}
	if minAge == nil {
		var noCache time.Duration
		minAge = &noCache
	}
	return Hint{MaxAge: minAge, Scope: s}
}
