package ratelimit

import (
	"context"

	"github.com/graph-gophers/graphql-go/introspection"
)

type RateLimiter interface {
	LimitQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) bool
}
