package graphql_test

import (
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/log"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/stretchr/testify/assert"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"testing"
)

func TestJaegerTracing(t *testing.T) {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		t.Skipf("skipping test; Could not initialize jaeger: %s", err)
		return
	}
	queryAPI := os.Getenv("JAEGER_QUERY_ENDPOINT")
	if queryAPI == "" {
		t.Skipf("skipping test; JAEGER_QUERY_ENDPOINT env not defined.")
		return
	}


	svcName := t.Name() + "-" + ksuid.New().String()
	queryURL := fmt.Sprintf(
		"%s?lookback=1h&limit=1&service=%s",
		queryAPI,
		svcName,
	)

	cfg.ServiceName = svcName
	cfg.Sampler.Type = jaeger.SamplerTypeConst
	cfg.Sampler.Param = 1
	cfg.Reporter.LogSpans = true

	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(log.StdLogger))
	if err != nil {
		t.Skipf("skipping test; Could not initialize jaeger: %s", err)
		return
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	engine, err := graphql.CreateEngine(`
		schema {
			query: Query
		}
		type Query {
			hello: String
		}`)
	if ! assert.NoError(t, err) {
		t.FailNow()
	}
	engine.Tracer = trace.OpenTracingTracer{}

	// No traces should be in the system yet..
	assertTraceCount(t, queryURL, 0)

	assertGraphQL(t, engine,
		`{"query":"{ hello }"}`,
		`{"data":{"hello":"World"}}`,
		map[string]interface{}{
			"hello": "World",
		},
	)

	time.Sleep(1 * time.Second)
	assertTraceCount(t, queryURL, 1)

}

func assertTraceCount(t *testing.T, queryURL string, count int)  {
	data := map[string]interface{}{}
	httpGetJson(t, queryURL, &data)
	datas := data["data"].([]interface{})
	assert.Equal(t, count, len(datas))
}

func httpGetJson(t *testing.T, url string, target interface{}) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonUnmarshal(t, string(bodyBytes), target)
}
