package main

import (
	"context"

	"github.com/jinzhu/gorm"
	graphql "github.com/neelance/graphql-go"
)

// Pet is the base type for pets to be used by the db and gql
type Pet struct {
	gorm.Model
	OwnerID uint
	Name    string
	Tags    []Tag `gorm:"many2many:pet_tags"`
}

// RESOLVERS ===========================================================================
// ID resolves the ID field for Pet
func (p *Pet) ID(ctx context.Context) *graphql.ID {
	return gqlIDP(p.Model.ID)
}

// OWNER resolves the owner field for Pet
func (p *Pet) OWNER(ctx context.Context) (*User, error) {
	return db.getPetOwner(ctx, int32(p.OwnerID))
}

// NAME resolves the name field for Pet
func (p *Pet) NAME(ctx context.Context) *string {
	return &p.Name
}

// TAGS resolves the pet tags
func (p *Pet) TAGS(ctx context.Context) (*[]*Tag, error) {
	return db.getPetTags(ctx, p)
}

// DB ===================================================================================
// GetPet should authorize the user in ctx and return a pet or error
func (db *DB) getPet(ctx context.Context, id int32) (*Pet, error) {
	var p Pet
	err := db.DB.First(&p, id).Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (db *DB) getPetOwner(ctx context.Context, id int32) (*User, error) {
	var u User
	err := db.DB.First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) getPetTags(ctx context.Context, p *Pet) (*[]*Tag, error) {
	var t []*Tag
	err := db.DB.Model(p).Related(&t, "Tags").Error
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (db *DB) getPetsByID(ctx context.Context, ids []int, from, to int) ([]Pet, error) {
	var p []Pet
	err := db.DB.Where("id in (?)", ids[from:to]).Find(&p).Error
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) updatePet(ctx context.Context, args *petInput) (*Pet, error) {
	// get the pet to be updated from the db
	var p Pet
	err := db.DB.First(&p, args.ID).Error
	if err != nil {
		return nil, err
	}

	// if there are tags to be updated, go through that process
	var newTags []Tag
	if len(args.TagIDs) > 0 {
		err = db.DB.Where("id in (?)", args.TagIDs).Find(&newTags).Error
		if err != nil {
			return nil, err
		}

		// replace the old tag set with the new one
		err = db.DB.Model(&p).Association("Tags").Replace(newTags).Error
		if err != nil {
			return nil, err
		}
	}

	updated := Pet{
		Name:    args.Name,
		OwnerID: uint(args.OwnerID),
	}

	err = db.DB.Model(&p).Updates(updated).Error
	if err != nil {
		return nil, err
	}

	err = db.DB.First(&p, args.ID).Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (db *DB) deletePet(ctx context.Context, userID, petID int32) (*bool, error) {
	// make sure the record exist
	var p Pet
	err := db.DB.First(&p, petID).Error
	if err != nil {
		return nil, err
	}

	// delete tags
	err = db.DB.Model(&p).Association("Tags").Clear().Error
	if err != nil {
		return nil, err
	}

	// delete record
	err = db.DB.Delete(&p).Error
	if err != nil {
		return nil, err
	}

	return boolP(true), err
}

func (db *DB) addPet(ctx context.Context, input petInput) (*Pet, error) {
	// get the M2M relation tags from the DB and put them in the pet to be saved
	var t []Tag
	err := db.DB.Where("id in (?)", input.TagIDs).Find(&t).Error
	if err != nil {
		return nil, err
	}

	pet := Pet{
		Name:    input.Name,
		OwnerID: uint(input.OwnerID),
		Tags:    t,
	}

	err = db.DB.Create(&pet).Error
	if err != nil {
		return nil, err
	}

	return &pet, nil
}
