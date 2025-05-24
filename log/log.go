package log

import (
	"context"
	"log"
	"runtime"
)

// Logger is the interface used to log panics that occur during query execution. It is settable via graphql.ParseSchema.
type Logger interface {
	LogPanic(ctx context.Context, value interface{})
}

// LoggerFunc is a function type that implements the Logger interface.
type LoggerFunc func(ctx context.Context, value interface{})

// LogPanic calls the LoggerFunc with the given context and panic value.
func (f LoggerFunc) LogPanic(ctx context.Context, value interface{}) {
	f(ctx, value)
}

// DefaultLogger is the default logger used to log panics that occur during query execution.
type DefaultLogger struct{}

// LogPanic is used to log recovered panic values that occur during query execution.
func (l *DefaultLogger) LogPanic(ctx context.Context, value interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("graphql: panic occurred: %v\n%s\ncontext: %v", value, buf, ctx)
}
