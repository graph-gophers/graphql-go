package noop

import (
	"context"

	"github.com/graph-gophers/graphql-go/introspection"
)

// RateLimiter is a no-op rate limiter that does nothing.
type RateLimiter struct{}

func (r *RateLimiter) LimitQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) bool {
	return false
}
