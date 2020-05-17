package main

import (
	"fmt"
	"github.com/ricklxm/graphql-go/internal/schema"
	"log"
	"net/http"
	"reflect"

	"github.com/ricklxm/graphql-go"
	"github.com/ricklxm/graphql-go/example/social"
	"github.com/ricklxm/graphql-go/relay"
)

type MyProvider struct {

}

func (MyProvider) Misc() (*MiscResolver, error)  {
	return &MiscResolver{}, nil
}

type MiscResolver struct {

}

func (m *MiscResolver) ID() string {
	return "100"
}

func (m *MiscResolver) Name() string {
	return "xxxx"
}

type MiscResolverProvider struct {}

func (m MiscResolverProvider) Misc() (*MiscResolver, error)  {
	return &MiscResolver{}, nil
}

func (m MyProvider) GetResolver(field schema.Field) *reflect.Value {
	fmt.Println(field.Type.String())
	fmt.Println(field.Type.Kind())
	if field.Type.String() == "Misc!" {
		ty := reflect.ValueOf(MiscResolverProvider{})
		return &ty
	}
	return nil
}

func main() {
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.MaxParallelism(20), graphql.UseResolverProvider(MyProvider{})}
	schema := graphql.MustParseSchema(social.Schema, &social.Resolver{}, opts...)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":9011", nil))
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
				return fetch("/query", {
					method: "post",
					body: JSON.stringify(graphQLParams),
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
