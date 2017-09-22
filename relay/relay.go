package relay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	graphql "github.com/neelance/graphql-go"
)

func MarshalID(kind string, spec interface{}) graphql.ID {
	d, err := json.Marshal(spec)
	if err != nil {
		panic(fmt.Errorf("relay.MarshalID: %s", err))
	}
	return graphql.ID(base64.URLEncoding.EncodeToString(append([]byte(kind+":"), d...)))
}

func UnmarshalKind(id graphql.ID) string {
	s, err := base64.URLEncoding.DecodeString(string(id))
	if err != nil {
		return ""
	}
	i := strings.IndexByte(string(s), ':')
	if i == -1 {
		return ""
	}
	return string(s[:i])
}

func UnmarshalSpec(id graphql.ID, v interface{}) error {
	s, err := base64.URLEncoding.DecodeString(string(id))
	if err != nil {
		return err
	}
	i := strings.IndexByte(string(s), ':')
	if i == -1 {
		return errors.New("invalid graphql.ID")
	}
	return json.Unmarshal([]byte(s[i+1:]), v)
}

const (
	ContentTypeJSON           = "application/json"
	ContentTypeGraphQL        = "application/graphql"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type Handler struct {
	Schema   *graphql.Schema
	pretty   bool
	graphiql bool
}

type RequestOptions struct {
	Query         string                 `json:"query" url:"query" schema:"query"`
	Variables     map[string]interface{} `json:"variables" url:"variables" schema:"variables"`
	OperationName string                 `json:"operationName" url:"operationName" schema:"operationName"`
}

// A workaround for getting `variables` as a JSON string
type requestOptionsCompatibility struct {
	Query         string `json:"query" url:"query" schema:"query"`
	Variables     string `json:"variables" url:"variables" schema:"variables"`
	OperationName string `json:"operationName" url:"operationName" schema:"operationName"`
}

func getFromForm(values url.Values) *RequestOptions {
	query := values.Get("query")
	if query != "" {
		// get variables map
		var variables map[string]interface{}
		variablesStr := values.Get("variables")
		json.Unmarshal([]byte(variablesStr), variables)

		return &RequestOptions{
			Query:         query,
			Variables:     variables,
			OperationName: values.Get("operationName"),
		}
	}
	return nil
}

// NewRequestOptions Parses an http.Request into GraphQL request options struct
func NewRequestOptions(r *http.Request) *RequestOptions {
	if reqOpt := getFromForm(r.URL.Query()); reqOpt != nil {
		return reqOpt
	}

	if r.Method != "POST" {
		return &RequestOptions{}
	}

	if r.Body == nil {
		return &RequestOptions{}
	}

	// TODO: improve Content-Type handling
	contentTypeStr := r.Header.Get("Content-Type")
	contentTypeTokens := strings.Split(contentTypeStr, ";")
	contentType := contentTypeTokens[0]

	switch contentType {
	case ContentTypeGraphQL:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &RequestOptions{}
		}
		return &RequestOptions{
			Query: string(body),
		}
	case ContentTypeFormURLEncoded:
		if err := r.ParseForm(); err != nil {
			return &RequestOptions{}
		}

		if reqOpt := getFromForm(r.PostForm); reqOpt != nil {
			return reqOpt
		}

		return &RequestOptions{}

	case ContentTypeJSON:
		fallthrough
	default:
		var opts RequestOptions
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &opts
		}
		err = json.Unmarshal(body, &opts)
		if err != nil {
			// Probably `variables` was sent as a string instead of an object.
			// So, we try to be polite and try to parse that as a JSON string
			var optsCompatible requestOptionsCompatibility
			json.Unmarshal(body, &optsCompatible)
			json.Unmarshal([]byte(optsCompatible.Variables), &opts.Variables)
		}
		return &opts
	}
}

type param struct {
	Schema          *graphql.Schema
	RequestString   string
	VariablesValues map[string]interface{}
	OperationName   string
	Context         context.Context
}

// ContextHandler provides an entrypoint into executing graphQL queries with a
// user-provided context.
func (h *Handler) ContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Gets Query
	opts := NewRequestOptions(r)
	params := param{
		Schema:          h.Schema,
		RequestString:   opts.Query,
		VariablesValues: opts.Variables,
		OperationName:   opts.OperationName,
		Context:         ctx,
	}
	response := h.Schema.Exec(ctx, params.RequestString, params.OperationName, params.VariablesValues)

	if h.graphiql {
		acceptHeader := r.Header.Get("Accept")
		_, raw := r.URL.Query()["raw"]
		if !raw && !strings.Contains(acceptHeader, ContentTypeJSON) && strings.Contains(acceptHeader, "text/html") {
			renderGraphiQL(w, params)
			return
		}
	}

	// Use proper JSON Header
	w.Header().Add("Content-Type", ContentTypeJSON)

	if h.pretty {
		w.WriteHeader(http.StatusOK)
		responseJSON, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(responseJSON)
	} else {
		w.WriteHeader(http.StatusOK)
		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(responseJSON)
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ContextHandler(r.Context(), w, r)
}

type Config struct {
	Schema   *graphql.Schema
	Pretty   bool
	GraphiQL bool
}

func NewConfig() *Config {
	return &Config{
		Schema:   nil,
		Pretty:   true,
		GraphiQL: true,
	}
}

func New(p *Config) *Handler {
	if p == nil {
		p = NewConfig()
	}
	if p.Schema == nil {
		panic("Undefined GraphQL Schema")
	}

	return &Handler{
		Schema:   p.Schema,
		pretty:   p.Pretty,
		graphiql: p.GraphiQL,
	}
}
