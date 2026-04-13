// Package graphiql serves the shared GraphiQL UI used by multiple examples.
package graphiql

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var page []byte

// Handler serves the shared GraphiQL UI used by multiple examples.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
}
