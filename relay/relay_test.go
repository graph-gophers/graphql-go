package relay_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/starwars"
	"github.com/graph-gophers/graphql-go/relay"
)

var starwarsSchema = graphql.MustParseSchema(starwars.Schema, &starwars.Resolver{})

func TestServeHTTP(t *testing.T) {
	h := relay.Handler{Schema: starwarsSchema}
	// Test json
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/some/path/here", strings.NewReader(`{"query":"{ hero { name } }", "operationName":"", "variables": null}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	assertResponse(w, t)

	// Test graphql
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/", strings.NewReader(`{ hero { name } }`))
	r.Header.Set("Content-Type", "application/graphql")
	h.ServeHTTP(w, r)
	assertResponse(w, t)
}

func assertResponse(w *httptest.ResponseRecorder, t *testing.T) {

	if w.Code != 200 {
		t.Fatalf("Expected status code 200, got %d.", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Invalid content-type. Expected [application/json], but instead got [%s]", contentType)
	}

	expectedResponse := `{"data":{"hero":{"name":"R2-D2"}}}`
	actualResponse := strings.TrimSpace(w.Body.String())
	if expectedResponse != actualResponse {
		t.Fatalf("Invalid response. Expected [%s], but instead got [%s]", expectedResponse, actualResponse)
	}
}
