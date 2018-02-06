package main

import (
	"math/rand"

	"github.com/jinzhu/gorm"
	// nolint: gotype
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// DB is the DB that will performs all operation
type DB struct {
	DB *gorm.DB
}

// NewDB returns a new DB connection
func newDB(path string) (*DB, error) {
	// connect to the example db, create if it doesn't exist
	db, err := gorm.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// drop tables and all data, and recreate them fresh for this run
	db.DropTableIfExists(&User{}, &Pet{}, &Tag{})
	db.AutoMigrate(&User{}, &Pet{}, &Tag{})

	// put all the users into the db
	for _, u := range users {
		if err := db.Create(&u).Error; err != nil {
			return nil, err
		}
	}

	var tg = []Tag{}
	for _, t := range tags {
		if err := db.Create(&t).Error; err != nil {
			return nil, err
		}

		tg = append(tg, t)
	}

	// put all the pets into the db
	for _, p := range pets {
		p.Tags = tg[:rand.Intn(5)]
		if err := db.Create(&p).Error; err != nil {
			return nil, err
		}
	}

	return &DB{db}, nil
}

// TEST DATA TO BE PUT INTO THE DB
var users = []User{
	User{Name: "Alice"},
	User{Name: "Bob"},
	User{Name: "Charlie"},
}

// Since the db is torn down and created on every run, I know the above users will have
// ID's 1, 2, 3
var pets = []Pet{
	Pet{Name: "rex", OwnerID: 1},
	Pet{Name: "goldie", OwnerID: 1},
	Pet{Name: "spot", OwnerID: 1},
	Pet{Name: "pokey", OwnerID: 1},
	Pet{Name: "sneezy", OwnerID: 1},
	Pet{Name: "duke", OwnerID: 1},
	Pet{Name: "duchess", OwnerID: 1},
	Pet{Name: "bernard", OwnerID: 2},
	Pet{Name: "William III of Chesterfield", OwnerID: 3},
	Pet{Name: "hops", OwnerID: 3},
}

// Tags to be put in the database
var tags = []Tag{
	Tag{Title: "funny"},
	Tag{Title: "energetic"},
	Tag{Title: "lazy"},
	Tag{Title: "hungry"},
	Tag{Title: "dangerous"},
}
