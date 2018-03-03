package main

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	graphql "github.com/neelance/graphql-go"
)

// User is the base user model to be used throughout the app
type User struct {
	gorm.Model
	Name string
	pets []Pet
}

// RESOLVER METHODS ====================================================================
// ID resolves the user ID
func (u *User) ID(ctx context.Context) *graphql.ID {
	return gqlIDP(u.Model.ID)
}

// NAME resolves the Name field for User, it is all caps to avoid name clashes
func (u *User) NAME(ctx context.Context) *string {
	return &u.Name
}

// PETS resolves the Pets field for User
func (u *User) PETS(ctx context.Context) (*[]*Pet, error) {
	return db.GetUserPets(ctx, int32(u.Model.ID))
}

// GetUserPets gets pets associated with the user
func (db *DB) GetUserPets(ctx context.Context, id int32) (*[]*Pet, error) {
	var p []*Pet
	err := db.DB.Model(&User{}).Association("pets").Find(&p).Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// DB METHODS ==========================================================================
func (db *DB) getUserPetIDs(ctx context.Context, userID uint) ([]int, error) {
	var ids []int
	err := db.DB.Where("owner_id = ?", userID).Find(&[]Pet{}).Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (db *DB) getUser(ctx context.Context, id int32) (*User, error) {
	var user User
	err := db.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// PAGINATION ==========================================================================
type petsConnArgs struct {
	First *int32
	After *graphql.ID
}

// PETSCONNECTION returns nodes (pets) connected by edges (relationships)
func (u *User) PETSCONNECTION(ctx context.Context, args petsConnArgs) (*UserPetsConnection, error) {

	// query only the ID fields from the pets otherwise it would be wasteful
	ids, err := db.getUserPetIDs(ctx, u.Model.ID)
	if err != nil {
		return nil, err
	}

	from := 0
	if args.After != nil {
		b, err := base64.StdEncoding.DecodeString(string(*args.After))
		if err != nil {
			return nil, err
		}
		i, err := strconv.Atoi(strings.TrimPrefix(string(b), "cursor"))
		if err != nil {
			return nil, err
		}
		from = i + 1
	}

	to := len(ids)
	if args.First != nil {
		to = from + int(*args.First)
		if to > len(ids) {
			to = len(ids)
		}
	}

	upc := UserPetsConnection{
		ids:  ids,
		from: from,
		to:   to,
	}
	return &upc, nil
}

// UserPetEdge is an edge (related node) that is returned in pagination
type UserPetEdge struct {
	cursor graphql.ID
	node   Pet
}

// CURSOR resolves the cursor for pagination
func (u *UserPetEdge) CURSOR(ctx context.Context) graphql.ID {
	return u.cursor
}

// NODE resolves the node for pagination
func (u *UserPetEdge) NODE(ctx context.Context) *Pet {
	return &u.node
}

// PageInfo gives page info for pagination
type PageInfo struct {
	StartCursor     graphql.ID
	EndCursor       graphql.ID
	HasNextPage     bool
	HasPreviousPage bool
}

// STARTCURSOR ...
func (u *PageInfo) STARTCURSOR(ctx context.Context) *graphql.ID {
	return &u.StartCursor
}

// ENDCURSOR ...
func (u *PageInfo) ENDCURSOR(ctx context.Context) *graphql.ID {
	return &u.EndCursor
}

// HASNEXTPAGE returns true if there are more results to show
func (u *PageInfo) HASNEXTPAGE(ctx context.Context) bool {
	return u.HasNextPage
}

// HASPREVIOUSPAGE returns true if there are results behind the current cursor position
func (u *PageInfo) HASPREVIOUSPAGE(ctx context.Context) bool {
	return u.HasPreviousPage
}

// UserPetsConnection is all the pets that are connected to a certain user
type UserPetsConnection struct {
	ids  []int
	from int
	to   int
}

// TOTALCOUNT gives the total amount of pets in UserPetsConnection
func (u UserPetsConnection) TOTALCOUNT(ctx context.Context) int32 {
	return int32(len(u.ids))
}

// EDGES gives a list of all the edges (related pets) that belong to a user
func (u *UserPetsConnection) EDGES(ctx context.Context) (*[]*UserPetEdge, error) {
	// query goes here because I know all of the ids that are needed. If I queried in the
	// UserPetEdge resolver method, it would run multiple single queries
	pets, err := db.getPetsByID(ctx, u.ids, u.from, u.to)
	if err != nil {
		return nil, err
	}

	l := make([]*UserPetEdge, u.to-u.from)
	for i := range l {
		l[i] = &UserPetEdge{
			cursor: encodeCursor(u.from + i),
			node:   pets[i],
		}
	}

	return &l, nil
}

// PAGEINFO resolves page info
func (u *UserPetsConnection) PAGEINFO(ctx context.Context) (*PageInfo, error) {
	p := PageInfo{
		StartCursor:     encodeCursor(u.from),
		EndCursor:       encodeCursor(u.to - 1),
		HasNextPage:     u.to < len(u.ids),
		HasPreviousPage: u.from > 0,
	}
	return &p, nil
}
