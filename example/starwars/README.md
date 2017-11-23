# Star Wars example documentation

![Star Wars](http://static1.businessinsider.com/image/58fa22197522ca38008b539f-1190-625/a-brand-new-star-wars-game-was-just-announced--heres-everything-we-know.jpg)

## Getting started
These instructions will get you a copy of the project up and running on your local machine. You will be able to make a simple GraphQL query to get started easily.

### Prerequisites
This documentations assumes that you have 

- Go installed and your GOPATH set up. [Install Golang](https://golang.org/doc/install)
- svn installed just for the next step Ubuntu: `sudo apt install subversion` Mac: `brew install subversion`

### Setup

Get the Star Wars example directory on your local machine.

```bash
svn export https://github.com/neelance/graphql-go.git/trunk/example/starwars graphql-go-starwars
```

Enter the directory

```bash
cd graphql-go-starwars/server
```

Get the required library.

```bash
go get github.com/neelance/graphql-go
```

### Start playing

Start the server.

```bash
go run server.go
```

Now visit http://localhost:8080. You will get the GraphiQL interface. Open the docs to see what Queries and Mutations are available.

Here's an example query, enter the query and press play.

```graphql
{
  character(id: 1000) {
    name
    appearsIn
  }
}
```
