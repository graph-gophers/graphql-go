package playground

import (
	"bytes"
	"html/template"
	"net/http"
)

func Handler(endpoint string, options ...Option) http.HandlerFunc {
	// set options
	c := &config{title: "GraphQL Playground", playgroundVersion: "1.4.6"}
	for _, opt := range options {
		opt(c)
	}

	// execute template
	var buff bytes.Buffer
	err := page.Execute(&buff, map[string]string{
		"title":    c.title,
		"endpoint": endpoint,
		"version":  c.playgroundVersion,
	})
	if err != nil {
		panic(err)
	}
	out := buff.Bytes()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(out)
	}
}

type Option func(*config)

type config struct {
	title             string
	playgroundVersion string
}

func WithTitle(title string) Option {
	return func(config *config) {
		config.title = title
	}
}

func WithVersion(version string) Option {
	return func(config *config) {
		config.playgroundVersion = version
	}
}

var page = template.Must(template.New("graphql-playground").Parse(`<!DOCTYPE html>
<html>
<head>
	<meta charset=utf-8/>
	<meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
	<link rel="shortcut icon" href="https://graphcool-playground.netlify.com/favicon.png">
	<link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .version }}/build/static/css/index.css"/>
	<link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .version }}/build/favicon.png"/>
	<script src="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .version }}/build/static/js/middleware.js"></script>
	<title>{{.title}}</title>
</head>
<body>
<style type="text/css">
	html { font-family: "Open Sans", sans-serif; overflow: hidden; }
	body { margin: 0; background: #172a3a; }
</style>
<div id="root"/>
<script type="text/javascript">
	window.addEventListener('load', function (event) {
		const root = document.getElementById('root');
		root.classList.add('playgroundIn');
		const wsProto = location.protocol == 'https:' ? 'wss:' : 'ws:'
		GraphQLPlayground.init(root, {
			endpoint: location.protocol + '//' + location.host + '{{.endpoint}}',
		})
	})
</script>
</body>
</html>
`))
