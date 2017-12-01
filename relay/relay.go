package relay

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	graphql "github.com/graph-gophers/graphql-go"
)

const (
	ContentTypeJSON    = "application/json"
	ContentTypeGraphQL = "application/graphql"
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

type Handler struct {
	Schema *graphql.Schema
	pretty bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "GraphQL only supports POST requests.", http.StatusMethodNotAllowed)
		return
	}

	if r.Body == nil {
		http.Error(w, "No body provided.", http.StatusBadRequest)
		return
	}

	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}

	// Check Content-Type
	contentTypeStr := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentTypeStr, ContentTypeJSON):
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "POST body is invalid JSON.", http.StatusBadRequest)
			return
		}
	case strings.HasPrefix(contentTypeStr, ContentTypeGraphQL):
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			if err == bytes.ErrTooLarge {
				http.Error(w, "POST body is too large.", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "POST body is invalid.", http.StatusInternalServerError)
			}
			return
		}
		params.Query = string(data)
	default:
		http.Error(w, "Not supported content type.", http.StatusBadRequest)
		return
	}

	if params.Query == "" {
		http.Error(w, "Must provide query string.", http.StatusBadRequest)
		return
	}

	response := h.Schema.Exec(r.Context(), params.Query, params.OperationName, params.Variables)

	// Process response
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if h.pretty {
		encoder.SetIndent("", "\t")
	}
	if err := encoder.Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func NewHandler(schema *graphql.Schema, pretty bool) *Handler {
	if schema == nil {
		panic("nil GraphQL schema")
	}

	return &Handler{
		Schema: schema,
		pretty: pretty,
	}
}
