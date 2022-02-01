module github.com/graph-gophers/graphql-go/tracer/otelgraphql

go 1.17

replace github.com/graph-gophers/graphql-go => ../../

require (
	github.com/graph-gophers/graphql-go v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
)

require (
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
)
