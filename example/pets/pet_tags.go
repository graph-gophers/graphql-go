package main

import (
	"context"

	"github.com/jinzhu/gorm"
	graphql "github.com/neelance/graphql-go"
)

// Tag is the base type for a pet tag to be used by the db and gql
type Tag struct {
	gorm.Model
	Title string
	Pets  []Pet `gorm:"many2many:pet_tags"`
}

// RESOLVERS ===========================================================================
// ID resolves the ID for Tag
func (t *Tag) ID(ctx context.Context) *graphql.ID {
	return gqlIDP(t.Model.ID)
}

// TITLE resolves the title field
func (t *Tag) TITLE(ctx context.Context) *string {
	return &t.Title
}

// PETS resolves the pets field
func (t *Tag) PETS(ctx context.Context) (*[]*Pet, error) {
	return db.getTagPets(ctx, t)
}

// DB ==================================================================================
func (db *DB) getTagPets(ctx context.Context, t *Tag) (*[]*Pet, error) {
	var p []*Pet
	err := db.DB.Model(t).Related(&p, "Pets").Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (db *DB) getTagBytTitle(ctx context.Context, title string) (*Tag, error) {
	var t Tag
	err := db.DB.Where("title = ?", title).First(&t).Error
	if err != nil {
		return nil, err
	}

	return &t, nil
}
