package otelgraphql_test

import (
	"testing"

	"go.opentelemetry.io/otel"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/tracer"
	"github.com/graph-gophers/graphql-go/tracer/otelgraphql"
)

func TestInterfaceImplementation(t *testing.T) {
	var _ tracer.ValidationTracer = &otelgraphql.OpenTelemetryTracer{}
	var _ tracer.Tracer = &otelgraphql.OpenTelemetryTracer{}
}

func TestTracerOption(t *testing.T) {
	_, err := graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(otelgraphql.DefaultOpenTelemetryTracer()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = graphql.ParseSchema(starwars.Schema, nil, graphql.Tracer(&otelgraphql.OpenTelemetryTracer{Tracer: otel.Tracer("example")}))
	if err != nil {
		t.Fatal(err)
	}
}

