package social

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
)

const Schema = `
	schema {
		query: Query
	}
	
	type Query {
		admin(id: ID!, role: Role = ADMIN): Admin!
		user(id: ID!): User!
		search(text: String!): [SearchResult]!
	}
	
	interface Admin {
		id: ID!
		name: String!
		role: Role!
	}

	scalar Time	

	type User implements Admin {
		id: ID!
		name: String!
		email: String!
		role: Role!
		phone: String!
		address: [String!]
		friends(page: Pagination): [User]
		createdAt: Time!
	}

	input Pagination {
	  	first: Int
	  	last: Int
	}
	
	enum Role {
		ADMIN
		USER
	}

	union SearchResult = User
`

type page struct {
	First *float64
	Last  *float64
}

type admin interface {
	ID() graphql.ID
	Name() string
	Role() string
}

type searchResult struct {
	result interface{}
}

func (r *searchResult) ToUser() (*user, bool) {
	res, ok := r.result.(*user)
	return res, ok
}

type user struct {
	IDField   string
	NameField string
	RoleField string
	Email     string
	Phone     string
	Address   *[]string
	Friends   *[]*user
	CreatedAt graphql.Time
}

func (u user) ID() graphql.ID {
	return graphql.ID(u.IDField)
}

func (u user) Name() string {
	return u.NameField
}

func (u user) Role() string {
	return u.RoleField
}

func (u user) FriendsResolver(args struct{ Page *page }) (*[]*user, error) {
	var from int
	numFriends := len(*u.Friends)
	to := numFriends

	if args.Page != nil {
		if args.Page.First != nil {
			from = int(*args.Page.First)
			if from > numFriends {
				return nil, errors.New("not enough users")
			}
		}
		if args.Page.Last != nil {
			to = int(*args.Page.Last)
			if to == 0 || to > numFriends {
				to = numFriends
			}
		}
	}

	friends := (*u.Friends)[from:to]

	return &friends, nil
}

var users = []*user{
	{
		IDField:   "0x01",
		NameField: "Albus Dumbledore",
		RoleField: "ADMIN",
		Email:     "Albus@hogwarts.com",
		Phone:     "000-000-0000",
		Address:   &[]string{"Office @ Hogwarts", "where Horcruxes are"},
		CreatedAt: graphql.Time{Time: time.Now()},
	},
	{
		IDField:   "0x02",
		NameField: "Harry Potter",
		RoleField: "USER",
		Email:     "harry@hogwarts.com",
		Phone:     "000-000-0001",
		Address:   &[]string{"123 dorm room @ Hogwarts", "456 random place"},
		CreatedAt: graphql.Time{Time: time.Now()},
	},
	{
		IDField:   "0x03",
		NameField: "Hermione Granger",
		RoleField: "USER",
		Email:     "hermione@hogwarts.com",
		Phone:     "000-000-0011",
		Address:   &[]string{"233 dorm room @ Hogwarts", "786 @ random place"},
		CreatedAt: graphql.Time{Time: time.Now()},
	},
	{
		IDField:   "0x04",
		NameField: "Ronald Weasley",
		RoleField: "USER",
		Email:     "ronald@hogwarts.com",
		Phone:     "000-000-0111",
		Address:   &[]string{"411 dorm room @ Hogwarts", "981 @ random place"},
		CreatedAt: graphql.Time{Time: time.Now()},
	},
}

var usersMap = make(map[string]*user)

func init() {
	users[0].Friends = &[]*user{users[1]}
	users[1].Friends = &[]*user{users[0], users[2], users[3]}
	users[2].Friends = &[]*user{users[1], users[3]}
	users[3].Friends = &[]*user{users[1], users[2]}
	for _, usr := range users {
		usersMap[usr.IDField] = usr
	}
}

type Resolver struct{}

func (r *Resolver) Admin(ctx context.Context, args struct {
	ID   string
	Role string
}) (admin, error) {
	if usr, ok := usersMap[args.ID]; ok {
		if usr.RoleField == args.Role {
			return *usr, nil
		}
	}
	err := fmt.Errorf("user with id=%s and role=%s does not exist", args.ID, args.Role)
	return user{}, err
}

func (r *Resolver) User(ctx context.Context, args struct{ Id string }) (user, error) {
	if usr, ok := usersMap[args.Id]; ok {
		return *usr, nil
	}
	err := fmt.Errorf("user with id=%s does not exist", args.Id)
	return user{}, err
}

func (r *Resolver) Search(ctx context.Context, args struct{ Text string }) ([]*searchResult, error) {
	var result []*searchResult
	for _, usr := range users {
		if strings.Contains(usr.NameField, args.Text) {
			result = append(result, &searchResult{usr})
		}
	}
	return result, nil
}
