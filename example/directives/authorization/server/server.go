package main

import (
	"context"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/directives"
	"github.com/graph-gophers/graphql-go/example/directives/authorization"
	"github.com/graph-gophers/graphql-go/example/directives/authorization/user"
	"github.com/graph-gophers/graphql-go/relay"
)

func main() {
	opts := []graphql.SchemaOpt{
		graphql.DirectiveVisitors(map[string]directives.Visitor{
			"hasRole": &authorization.HasRoleDirective{},
		}),
		// other options go here
	}
	schema := graphql.MustParseSchema(authorization.Schema, &authorization.Resolver{}, opts...)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	http.Handle("/query", auth(&relay.Handler{Schema: schema}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := &user.User{}
		role := r.Header.Get("role")
		if role != "" {
			u.AddRole(role)
		}
		ctx := user.AddToContext(context.Background(), u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var page = []byte(`
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>GraphiQL</title>
    <style>
      body {
        height: 100%;
        margin: 0;
        width: 100%;
        overflow: hidden;
      }
      #graphiql {
        height: 100vh;
      }
    </style>
    <script src="https://unpkg.com/react@17/umd/react.development.js" integrity="sha512-Vf2xGDzpqUOEIKO+X2rgTLWPY+65++WPwCHkX2nFMu9IcstumPsf/uKKRd5prX3wOu8Q0GBylRpsDB26R6ExOg==" crossorigin="anonymous"></script>
    <script src="https://unpkg.com/react-dom@17/umd/react-dom.development.js" integrity="sha512-Wr9OKCTtq1anK0hq5bY3X/AvDI5EflDSAh0mE9gma+4hl+kXdTJPKZ3TwLMBcrgUeoY0s3dq9JjhCQc7vddtFg==" crossorigin="anonymous"></script>
    <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
  </head>
  <body>
    <div id="graphiql">Loading...</div>
    <script src="https://unpkg.com/graphiql/graphiql.min.js" type="application/javascript"></script>
    <script>
      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: GraphiQL.createFetcher({url: '/query'}),
          defaultEditorToolsVisibility: true,
        }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
`)
