/*
This example demonstrates a 3-level hierarchy (Author -> Books -> Reviews)
with data prefetching at each level to avoid N+1 query problems.
To run the example, execute:

	go run example/prefetch/main.go

Then send a query like this (using curl or any GraphQL client):

	curl -X POST http://localhost:8080/query \
	   -H 'Content-Type: application/json' \
	   -d '{"query":"query GetAuthors($top:Int!,$last:Int!){ authors { id name books(top:$top){ id title reviews(last:$last){ id content rating } } }}","variables":{"top":2,"last":2}}'
*/
package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

//go:embed schema.graphql
var sdl string

type Author struct {
	ID   string
	Name string
}
type Book struct {
	ID       string
	AuthorID string
	Title    string
}
type Review struct {
	ID      string
	BookID  string
	Content string
	Rating  int32
}

var (
	allAuthors = []Author{{"A1", "Ann"}, {"A2", "Bob"}}
	allBooks   = []Book{{"B1", "A1", "Go Tips"}, {"B2", "A1", "Concurrency"}, {"B3", "A2", "GraphQL"}}
	allReviews = []Review{{"R1", "B1", "Great", 5}, {"R2", "B1", "Okay", 3}, {"R3", "B3", "Wow", 4}}
)

type root struct {
	booksByAuthor map[string][]Book
	reviewsByBook map[string][]Review
}

func (r *root) Authors(ctx context.Context) ([]*authorResolver, error) {
	authors := allAuthors
	// level 1 prefetch: authors already available
	if graphql.HasSelectedField(ctx, "books") {
		// level 2 prefetch: books for selected authors only
		authorSet := make(map[string]struct{}, len(authors))
		for _, a := range authors {
			authorSet[a.ID] = struct{}{}
		}
		booksByAuthor := make(map[string][]Book)
		// capture potential Top argument once (shared across authors)
		var topLimit int
		var booksArgs struct{ Top int32 }
		if ok, _ := graphql.DecodeSelectedFieldArgs(ctx, "books", &booksArgs); ok && booksArgs.Top > 0 {
			topLimit = int(booksArgs.Top)
		}
		for _, b := range allBooks {
			if _, ok := authorSet[b.AuthorID]; ok {
				list := booksByAuthor[b.AuthorID]
				if topLimit == 0 || len(list) < topLimit {
					list = append(list, b)
					booksByAuthor[b.AuthorID] = list
				}
			}
		}
		if graphql.HasSelectedField(ctx, "books.reviews") {
			var lastLimit int
			var rvArgs struct{ Last int32 }
			if ok, _ := graphql.DecodeSelectedFieldArgs(ctx, "books.reviews", &rvArgs); ok && rvArgs.Last > 0 {
				lastLimit = int(rvArgs.Last)
			}
			bookSet := map[string]struct{}{}
			for _, slice := range booksByAuthor {
				for _, b := range slice {
					bookSet[b.ID] = struct{}{}
				}
			}
			reviewsByBook := make(map[string][]Review)
			for _, rv := range allReviews {
				if _, ok := bookSet[rv.BookID]; ok {
					grp := reviewsByBook[rv.BookID]
					grp = append(grp, rv)
					if lastLimit > 0 && len(grp) > lastLimit {
						grp = grp[len(grp)-lastLimit:]
					}
					reviewsByBook[rv.BookID] = grp
				}
			}
			r.reviewsByBook = reviewsByBook
		}
		r.booksByAuthor = booksByAuthor
	}
	res := make([]*authorResolver, len(authors))
	for i, a := range authors {
		res[i] = &authorResolver{root: r, a: &a}
	}
	return res, nil
}

type authorResolver struct {
	root *root
	a    *Author
}

func (ar *authorResolver) ID() graphql.ID { return graphql.ID(ar.a.ID) }
func (ar *authorResolver) Name() string   { return ar.a.Name }

func (ar *authorResolver) Books(ctx context.Context, args struct{ Top int32 }) ([]*bookResolver, error) {
	// books already limited during prefetch phase (Authors resolver)
	books := ar.root.booksByAuthor[ar.a.ID]
	out := make([]*bookResolver, len(books))
	for i, b := range books {
		out[i] = &bookResolver{root: ar.root, b: &b}
	}
	return out, nil
}

type bookResolver struct {
	root *root
	b    *Book
}

func (br *bookResolver) ID() graphql.ID { return graphql.ID(br.b.ID) }
func (br *bookResolver) Title() string  { return br.b.Title }
func (br *bookResolver) Reviews(ctx context.Context, args struct{ Last int32 }) ([]*reviewResolver, error) {
	revs := br.root.reviewsByBook[br.b.ID]
	if take := int(args.Last); take > 0 && take < len(revs) {
		start := len(revs) - take
		if start < 0 {
			start = 0
		}
		revs = revs[start:]
	}
	out := make([]*reviewResolver, len(revs))
	for i, r := range revs {
		out[i] = &reviewResolver{r: &r}
	}
	return out, nil
}

type reviewResolver struct{ r *Review }

func (rr *reviewResolver) ID() graphql.ID  { return graphql.ID(rr.r.ID) }
func (rr *reviewResolver) Content() string { return rr.r.Content }
func (rr *reviewResolver) Rating() int32   { return rr.r.Rating }

func main() {
	schema := graphql.MustParseSchema(sdl, &root{})
	http.Handle("/query", &relay.Handler{Schema: schema})
	log.Println("Prefetch example listening on :8080 -> POST /query")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
