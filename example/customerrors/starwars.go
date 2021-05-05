package customerrors

import (
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

var Schema = `
	schema {
		query: Query
	}
	type Query {
		droid(id: ID!): Droid!
	}
	# An autonomous mechanical character in the Star Wars universe
	type Droid {
		# The ID of the droid
		id: ID!
		# What others call this droid
		name: String!
	}
`

type droid struct {
	ID   graphql.ID
	Name string
}

var droids = []*droid{
	{ID: "2000", Name: "C-3PO"},
	{ID: "2001", Name: "R2-D2"},
}

var droidData = make(map[graphql.ID]*droid)

func init() {
	for _, d := range droids {
		droidData[d.ID] = d
	}
}

type Resolver struct{}

func (r *Resolver) Droid(args struct{ ID graphql.ID }) (*droidResolver, error) {
	if d := droidData[args.ID]; d != nil {
		return &droidResolver{d: d}, nil
	}
	return nil, &droidNotFoundError{Code: "NotFound", Message: "This is not the droid you are looking for"}
}

type droidResolver struct {
	d *droid
}

func (r *droidResolver) ID() graphql.ID {
	return r.d.ID
}

func (r *droidResolver) Name() string {
	return r.d.Name
}

type droidNotFoundError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e droidNotFoundError) Error() string {
	return fmt.Sprintf("error [%s]: %s", e.Code, e.Message)
}

func (e droidNotFoundError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	}
}
