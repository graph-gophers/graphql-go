package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/example/caching"
	"github.com/graph-gophers/graphql-go/example/caching/cache"
)

var schema *graphql.Schema

func init() {
	schema = graphql.MustParseSchema(caching.Schema, &caching.Resolver{})
}

func main() {
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	http.Handle("/query", &Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Handler struct {
	Schema *graphql.Schema
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, ok := h.parseRequest(w, r)
	if !ok {
		return
	}
	var response *graphql.Response
	var hint *cache.Hint
	if cacheable(r) {
		ctx, hints, done := cache.Hintable(r.Context())
		response = h.Schema.Exec(ctx, p.Query, p.OperationName, p.Variables)
		done()
		v := <-hints
		hint = &v
	} else {
		response = h.Schema.Exec(r.Context(), p.Query, p.OperationName, p.Variables)
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if hint != nil {
		w.Header().Set("Cache-Control", hint.String())
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func (h *Handler) parseRequest(w http.ResponseWriter, r *http.Request) (params, bool) {
	var p params
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		if p.Query = q.Get("query"); p.Query == "" {
			http.Error(w, "A non-empty 'query' parameter is required", http.StatusBadRequest)
			return params{}, false
		}
		p.OperationName = q.Get("operationName")
		if vars := q.Get("variables"); vars != "" {
			if err := json.Unmarshal([]byte(vars), &p.Variables); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return params{}, false
			}
		}
		return p, true
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return params{}, false
		}
		return p, true
	default:
		http.Error(w, fmt.Sprintf("unsupported HTTP method: %s", r.Method), http.StatusMethodNotAllowed)
		return params{}, false
	}
}

func cacheable(r *http.Request) bool {
	return r.Method == http.MethodGet
}

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

var page = []byte(`
<!DOCTYPE html>
<html>
	<head>
		<link href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.11.11/graphiql.min.css" rel="stylesheet" />
		<script src="https://cdnjs.cloudflare.com/ajax/libs/es6-promise/4.1.1/es6-promise.auto.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/2.0.3/fetch.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/16.2.0/umd/react.production.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/16.2.0/umd/react-dom.production.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.11.11/graphiql.min.js"></script>
	</head>
	<body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
		<div id="graphiql" style="height: 100vh;">Loading...</div>
		<script>
			function graphQLFetcher(graphQLParams) {
				const uri = "/query?query=" + encodeURIComponent(graphQLParams.query || "") + "&operationName=" + encodeURIComponent(graphQLParams.operationName || "") + "&variables=" + encodeURIComponent(graphQLParams.variables || "");
				return fetch(uri, {
					method: "get",
					credentials: "include",
				}).then(function (response) {
					return response.text();
				}).then(function (responseBody) {
					try {
						return JSON.parse(responseBody);
					} catch (error) {
						return responseBody;
					}
				});
			}

			ReactDOM.render(
				React.createElement(GraphiQL, {fetcher: graphQLFetcher}),
				document.getElementById("graphiql")
			);
		</script>
	</body>
</html>
`)
