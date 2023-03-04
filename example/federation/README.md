# Apollo Federation

A basic example of integration with apollo federation as subgraph. Tested with Go v1.18, Node.js v16.14.2 and yarn 1.22.18.

To run this server

`go run ./example/apollo-federation/subgraph-one/server.go`

`go run ./example/apollo-federation/subgraph-two/server.go`

`cd example/apollo-federation/gateway`

`yarn start`

and go to localhost:4000 to interact

Execute the query:

```
query {
  hello
  hi
}
```

and you should see a result similar to this:

```json
{
  "data": {
    "hello": "Hello from subgraph one!",
    "hi": "Hi from subgraph two!"
  }
}
```
