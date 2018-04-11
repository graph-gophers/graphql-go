package social

import (
	"context"
	"fmt"
)

const Schema = `
	schema {
		query: Query
	}
	
	type Query {
		admin(id: ID!, role: Role = ADMIN): Admin!
		user(id: ID!): User!
	}
	
	interface Admin {
		id: ID!
		name: String!
		role: Role!
	}

	type User implements Admin {
		id: ID!
		name: String!
		email: String!
		role: Role!
		phone: String!
		address: [String!]
		friends(page: Pagination): [User]
	}

	input Pagination {
	  first: Int
	  last: Int
	}
	
	enum Role {
		ADMIN
		USER
	}
`

type page struct {
	First *int
	Last  *int
}

type admin interface {
	IdResolver() string
	NameResolver() string
	RoleResolver() string
}

type user struct {
	Id      string
	Name    string
	Role    string
	Email   string
	Phone   string
	Address *[]string
	Friends *[]*user
}

func (u user) IdResolver() string {
	return u.Id
}

func (u user) NameResolver() string {
	return u.Name
}

func (u user) RoleResolver() string {
	return u.Role
}

func (u user) FriendsResolver(args struct{ Page *page }) (*[]*user, error) {

	from := 0
	numFriends := len(*u.Friends)
	to := numFriends

	if args.Page != nil {
		if args.Page.First != nil {
			from = *args.Page.First
		}
		if args.Page.Last != nil {
			to = *args.Page.Last
			if to > numFriends {
				to = numFriends
			}
		}
	}

	friends := (*u.Friends)[from:to]

	return &friends, nil
}

var users = []*user{
	{
		Id:      "0x01",
		Name:    "Albus Dumbledore",
		Role:    "ADMIN",
		Email:   "Albus@hogwarts.com",
		Phone:   "000-000-0000",
		Address: &[]string{"Office @ Hogwarts", "where Horcruxes are"},
	},
	{
		Id:      "0x02",
		Name:    "Harry Potter",
		Role:    "USER",
		Email:   "harry@hogwarts.com",
		Phone:   "000-000-0001",
		Address: &[]string{"123 dorm room @ Hogwarts", "456 random place"},
	},
	{
		Id:      "0x03",
		Name:    "Hermione Granger",
		Role:    "USER",
		Email:   "hermione@hogwarts.com",
		Phone:   "000-000-0011",
		Address: &[]string{"233 dorm room @ Hogwarts", "786 @ random place"},
	},
	{
		Id:      "0x04",
		Name:    "Ronald Weasley",
		Role:    "USER",
		Email:   "ronald@hogwarts.com",
		Phone:   "000-000-0111",
		Address: &[]string{"411 dorm room @ Hogwarts", "981 @ random place"},
	},
}

var usersMap = make(map[string]*user)

func init() {
	users[0].Friends = &[]*user{users[1]}
	users[1].Friends = &[]*user{users[0], users[2], users[3]}
	users[2].Friends = &[]*user{users[1], users[3]}
	users[3].Friends = &[]*user{users[1], users[2]}
	for _, usr := range users {
		usersMap[usr.Id] = usr
	}
}

type Resolver struct{}

func (r *Resolver) Admin(ctx context.Context, args struct {
	Id   string
	Role string
}) (admin, error) {
	if usr, ok := usersMap[args.Id]; ok {
		if usr.Role == args.Role {
			return *usr, nil
		}
	}
	err := fmt.Errorf("user with id=%s and role=%s does not exist", args.Id, args.Role)
	return user{}, err
}

func (r *Resolver) User(ctx context.Context, args struct{ Id string }) (user, error) {
	if usr, ok := usersMap[args.Id]; ok {
		return *usr, nil
	}
	err := fmt.Errorf("user with id=%s does not exist", args.Id)
	return user{}, err
}
