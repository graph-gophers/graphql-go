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
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/some/path/here", strings.NewReader(`{"query":"{ hero { name } }", "operationName":"", "variables": null}`))
	h := relay.Handler{Schema: starwarsSchema}

	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("Expected status code 200, got %d.", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Invalid content-type. Expected [application/json], but instead got [%s]", contentType)
	}

	expectedResponse := `{"data":{"hero":{"name":"R2-D2"}}}`
	actualResponse := w.Body.String()
	if expectedResponse != actualResponse {
		t.Fatalf("Invalid response. Expected [%s], but instead got [%s]", expectedResponse, actualResponse)
	}
}

func TestServeHTTPWithGraphqli(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/graphql", nil)
	r.Header.Add("Accept", "text/html")
	h := relay.Handler{Schema: starwarsSchema, Graphqli: true}

	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("Expected status code 200, got %d.", w.Code)
	}

	actualContentType := w.Header().Get("Content-Type")
	expectedContentType := "text/html; charset=utf-8"
	if actualContentType != expectedContentType {
		t.Fatalf("Invalid content-type. Expected [%s], but instead got [%s]", expectedContentType, actualContentType)
	}

	actualResponse := w.Body.String()
	expectedResponse := string(relay.GraphqliPage)
	if string(relay.GraphqliPage) != actualResponse {
		t.Fatalf("Invalid response. Expected [%s], but instead got [%s]", expectedResponse, actualResponse)
	}
}
