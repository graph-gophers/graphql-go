package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	graphql "github.com/neelance/graphql-go"
)

var db DB

func init() {
	var err error
	d, err := newDB("./db.sqlite")
	if err != nil {
		panic(err)
	}

	db = *d
}

// Resolver is the root resolver
type Resolver struct{}

// GetUser resolves the getUser query
func (r *Resolver) GetUser(ctx context.Context, args struct{ ID int32 }) (*User, error) {
	return db.getUser(ctx, int32(args.ID))
}

// GetPet resolves the getPet query
func (r *Resolver) GetPet(ctx context.Context, args struct{ ID int32 }) (*Pet, error) {
	pet, err := db.getPet(ctx, args.ID)
	if err != nil {
		return nil, err
	}

	return pet, nil
}

// GetTag resolves the getTag query
func (r *Resolver) GetTag(ctx context.Context, args struct{ Title string }) (*Tag, error) {
	return db.getTagBytTitle(ctx, args.Title)
}

// petInput has everything needed to do adds and updates on a pet
type petInput struct {
	ID      int32
	OwnerID int32
	Name    string
	TagIDs  []*int32
}

// AddPet Resolves the addPet mutation
func (r *Resolver) AddPet(ctx context.Context, args struct{ Pet petInput }) (*Pet, error) {
	return db.addPet(ctx, args.Pet)
}

// UpdatePet takes care of updating any field on the pet
func (r *Resolver) UpdatePet(ctx context.Context, args struct{ Pet petInput }) (*Pet, error) {
	return db.updatePet(ctx, &args.Pet)
}

// DeletePet takes care of deleting a pet record
func (r *Resolver) DeletePet(ctx context.Context, args struct{ UserID, PetID int32 }) (*bool, error) {
	return db.deletePet(ctx, args.UserID, args.PetID)
}

// encode cursor encodes the cursot position in base64
func encodeCursor(i int) graphql.ID {
	return graphql.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", i))))
}

// decode cursor decodes the base 64 encoded cursor and resturns the integer
func decodeCursor(s string) (int, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return 0, err
	}

	i, err := strconv.Atoi(strings.TrimPrefix(string(b), "cursor"))
	if err != nil {
		return 0, err
	}

	return i, nil
}

func int32P(i uint) *int32 {
	r := int32(i)
	return &r
}

func boolP(b bool) *bool {
	return &b
}

func gqlIDP(id uint) *graphql.ID {
	r := graphql.ID(id)
	return &r
}
