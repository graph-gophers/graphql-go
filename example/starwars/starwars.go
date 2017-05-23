// Package starwars provides a example schema and resolver based on Star Wars characters.
//
// Source: https://github.com/graphql/graphql.github.io/blob/source/site/_core/swapiSchema.js
package starwars

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	graphql "github.com/neelance/graphql-go"
)

var schemaIDL = `
	schema {
		query: Query
		mutation: Mutation
	}
	# The query type, represents all of the entry points into our object graph
	type Query {
		hero(episode: Episode = NEWHOPE): Character
		reviews(episode: Episode!): [Review]!
		search(text: String!): [SearchResult]!
		character(id: ID!): Character
		droid(id: ID!): Droid
		human(id: ID!): Human
		starship(id: ID!): Starship
	}
	# The mutation type, represents all updates we can make to our data
	type Mutation {
		createReview(episode: Episode!, review: ReviewInput!): Review
	}
	# The episodes in the Star Wars trilogy
	enum Episode {
		# Star Wars Episode IV: A New Hope, released in 1977.
		NEWHOPE
		# Star Wars Episode V: The Empire Strikes Back, released in 1980.
		EMPIRE
		# Star Wars Episode VI: Return of the Jedi, released in 1983.
		JEDI
	}
	# A character from the Star Wars universe
	interface Character {
		# The ID of the character
		id: ID!
		# The name of the character
		name: String!
		# The friends of the character, or an empty list if they have none
		friends: [Character]
		# The friends of the character exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this character appears in
		appearsIn: [Episode!]!
	}
	# Units of height
	enum LengthUnit {
		# The standard unit around the world
		METER
		# Primarily used in the United States
		FOOT
	}
	# A humanoid creature from the Star Wars universe
	type Human implements Character {
		# The ID of the human
		id: ID!
		# What this human calls themselves
		name: String!
		# Height in the preferred unit, default is meters
		height(unit: LengthUnit = METER): Float!
		# Mass in kilograms, or null if unknown
		mass: Float
		# This human's friends, or an empty list if they have none
		friends: [Character]
		# The friends of the human exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this human appears in
		appearsIn: [Episode!]!
		# A list of starships this person has piloted, or an empty list if none
		starships: [Starship]
	}
	# An autonomous mechanical character in the Star Wars universe
	type Droid implements Character {
		# The ID of the droid
		id: ID!
		# What others call this droid
		name: String!
		# This droid's friends, or an empty list if they have none
		friends: [Character]
		# The friends of the droid exposed as a connection with edges
		friendsConnection(first: Int, after: ID): FriendsConnection!
		# The movies this droid appears in
		appearsIn: [Episode!]!
		# This droid's primary function
		primaryFunction: String
	}
	# A connection object for a character's friends
	type FriendsConnection {
		# The total number of friends
		totalCount: Int!
		# The edges for each of the character's friends.
		edges: [FriendsEdge]
		# A list of the friends, as a convenience when edges are not needed.
		friends: [Character]
		# Information for paginating this connection
		pageInfo: PageInfo!
	}
	# An edge object for a character's friends
	type FriendsEdge {
		# A cursor used for pagination
		cursor: ID!
		# The character represented by this friendship edge
		node: Character
	}
	# Information for paginating this connection
	type PageInfo {
		startCursor: ID
		endCursor: ID
		hasNextPage: Boolean!
	}
	# Represents a review for a movie
	type Review {
		# The number of stars this review gave, 1-5
		stars: Int!
		# Comment about the movie
		commentary: String
	}
	# The input object sent when someone is creating a new review
	input ReviewInput {
		# 0-5 stars
		stars: Int!
		# Comment about the movie, optional
		commentary: String
	}
	type Starship {
		# The ID of the starship
		id: ID!
		# The name of the starship
		name: String!
		# Length of the starship, along the longest axis
		length(unit: LengthUnit = METER): Float!
	}
	union SearchResult = Human | Droid | Starship
`

type human struct {
	ID        graphql.ID
	Name      string
	Friends   []graphql.ID
	AppearsIn []string
	Height    float64
	Mass      int
	Starships []graphql.ID
}

var humans = []*human{
	{
		ID:        "1000",
		Name:      "Luke Skywalker",
		Friends:   []graphql.ID{"1002", "1003", "2000", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.72,
		Mass:      77,
		Starships: []graphql.ID{"3001", "3003"},
	},
	{
		ID:        "1001",
		Name:      "Darth Vader",
		Friends:   []graphql.ID{"1004"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    2.02,
		Mass:      136,
		Starships: []graphql.ID{"3002"},
	},
	{
		ID:        "1002",
		Name:      "Han Solo",
		Friends:   []graphql.ID{"1000", "1003", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.8,
		Mass:      80,
		Starships: []graphql.ID{"3000", "3003"},
	},
	{
		ID:        "1003",
		Name:      "Leia Organa",
		Friends:   []graphql.ID{"1000", "1002", "2000", "2001"},
		AppearsIn: []string{"NEWHOPE", "EMPIRE", "JEDI"},
		Height:    1.5,
		Mass:      49,
	},
	{
		ID:        "1004",
		Name:      "Wilhuff Tarkin",
		Friends:   []graphql.ID{"1001"},
		AppearsIn: []string{"NEWHOPE"},
		Height:    1.8,
		Mass:      0,
	},
}

var humanData = make(map[graphql.ID]*human)

func init() {
	for _, h := range humans {
		humanData[h.ID] = h
	}
}

type droid struct {
	ID              graphql.ID
	Name            string
	Friends         []graphql.ID
	AppearsIn       []string
	PrimaryFunction string
}

var droids = []*droid{
	{
		ID:              "2000",
		Name:            "C-3PO",
		Friends:         []graphql.ID{"1000", "1002", "1003", "2001"},
		AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
		PrimaryFunction: "Protocol",
	},
	{
		ID:              "2001",
		Name:            "R2-D2",
		Friends:         []graphql.ID{"1000", "1002", "1003"},
		AppearsIn:       []string{"NEWHOPE", "EMPIRE", "JEDI"},
		PrimaryFunction: "Astromech",
	},
}

var droidData = make(map[graphql.ID]*droid)

func init() {
	for _, d := range droids {
		droidData[d.ID] = d
	}
}

type starship struct {
	ID     graphql.ID
	Name   string
	Length float64
}

var starships = []*starship{
	{
		ID:     "3000",
		Name:   "Millennium Falcon",
		Length: 34.37,
	},
	{
		ID:     "3001",
		Name:   "X-Wing",
		Length: 12.5,
	},
	{
		ID:     "3002",
		Name:   "TIE Advanced x1",
		Length: 9.2,
	},
	{
		ID:     "3003",
		Name:   "Imperial shuttle",
		Length: 20,
	},
}

var starshipData = make(map[graphql.ID]*starship)

func init() {
	for _, s := range starships {
		starshipData[s.ID] = s
	}
}

type review struct {
	Stars      int32
	Commentary *string
}

var reviews = make(map[string][]*review)

type root struct{}

func Schema() *graphql.Schema {
	b := graphql.ParseSchema(schemaIDL)

	b.Resolvers("Query", (*root)(nil), map[string]interface{}{
		"hero": func(r *root, args struct{ Episode string }) interface{} {
			if args.Episode == "EMPIRE" {
				return humanData["1000"]
			}
			return droidData["2001"]
		},

		"reviews": func(r *root, args struct{ Episode string }) []*review {
			return reviews[args.Episode]
		},

		"search": func(r *root, args struct{ Text string }) []interface{} {
			var l []interface{}
			for _, h := range humans {
				if strings.Contains(h.Name, args.Text) {
					l = append(l, h)
				}
			}
			for _, d := range droids {
				if strings.Contains(d.Name, args.Text) {
					l = append(l, d)
				}
			}
			for _, s := range starships {
				if strings.Contains(s.Name, args.Text) {
					l = append(l, s)
				}
			}
			return l
		},

		"character": func(r *root, args struct{ ID graphql.ID }) interface{} {
			if h := humanData[args.ID]; h != nil {
				return h
			}
			if d := droidData[args.ID]; d != nil {
				return d
			}
			return nil
		},

		"human": func(r *root, args struct{ ID graphql.ID }) *human {
			return humanData[args.ID]
		},

		"droid": func(r *root, args struct{ ID graphql.ID }) *droid {
			return droidData[args.ID]
		},

		"starship": func(r *root, args struct{ ID graphql.ID }) *starship {
			return starshipData[args.ID]
		},
	})

	b.Resolvers("Mutation", (*root)(nil), map[string]interface{}{
		"createReview": func(r *root, args *struct {
			Episode string
			Review  *reviewInput
		}) *review {
			review := &review{
				Stars:      args.Review.Stars,
				Commentary: args.Review.Commentary,
			}
			reviews[args.Episode] = append(reviews[args.Episode], review)
			return review
		},
	})

	b.Resolvers("Human", (*human)(nil), map[string]interface{}{
		"id":        "ID",
		"name":      "Name",
		"appearsIn": "AppearsIn",

		"height": func(h *human, args struct{ Unit string }) float64 {
			return convertLength(h.Height, args.Unit)
		},

		"mass": func(h *human) *float64 {
			if h.Mass == 0 {
				return nil
			}
			f := float64(h.Mass)
			return &f
		},

		"friends": func(h *human) *[]interface{} {
			return resolveCharacters(h.Friends)
		},

		"friendsConnection": func(h *human, args *friendsConnectionArgs) (*friendsConnection, error) {
			return newFriendsConnection(h.Friends, args)
		},

		"starships": func(h *human) *[]*starship {
			l := make([]*starship, len(h.Starships))
			for i, id := range h.Starships {
				l[i] = starshipData[id]
			}
			return &l
		},
	})

	b.Resolvers("Droid", (*droid)(nil), map[string]interface{}{
		"id":        "ID",
		"name":      "Name",
		"appearsIn": "AppearsIn",

		"friends": func(d *droid) *[]interface{} {
			return resolveCharacters(d.Friends)
		},

		"friendsConnection": func(d *droid, args *friendsConnectionArgs) (*friendsConnection, error) {
			return newFriendsConnection(d.Friends, args)
		},

		"primaryFunction": func(d *droid) *string {
			if d.PrimaryFunction == "" {
				return nil
			}
			return &d.PrimaryFunction
		},
	})

	b.Resolvers("Starship", (*starship)(nil), map[string]interface{}{
		"id":   "ID",
		"name": "Name",

		"length": func(s *starship, args struct{ Unit string }) float64 {
			return convertLength(s.Length, args.Unit)
		},
	})

	b.Resolvers("Review", (*review)(nil), map[string]interface{}{
		"stars":      "Stars",
		"commentary": "Commentary",
	})

	b.Resolvers("FriendsConnection", (*friendsConnection)(nil), map[string]interface{}{
		"totalCount": func(c *friendsConnection) int32 {
			return int32(len(c.IDs))
		},

		"edges": func(c *friendsConnection) *[]*friendsEdge {
			l := make([]*friendsEdge, c.To-c.From)
			for i := range l {
				l[i] = &friendsEdge{
					Cursor: encodeCursor(c.From + i),
					ID:     c.IDs[c.From+i],
				}
			}
			return &l
		},

		"friends": func(c *friendsConnection) *[]interface{} {
			return resolveCharacters(c.IDs[c.From:c.To])
		},

		"pageInfo": func(c *friendsConnection) *pageInfo {
			start := encodeCursor(c.From)
			end := encodeCursor(c.To - 1)
			return &pageInfo{
				StartCursor: &start,
				EndCursor:   &end,
				HasNextPage: c.To < len(c.IDs),
			}
		},
	})

	b.Resolvers("FriendsEdge", (*friendsEdge)(nil), map[string]interface{}{
		"cursor": "Cursor",

		"node": func(e *friendsEdge) interface{} {
			return resolveCharacter(e.ID)
		},
	})

	b.Resolvers("PageInfo", (*pageInfo)(nil), map[string]interface{}{
		"startCursor": "StartCursor",
		"endCursor":   "EndCursor",
		"hasNextPage": "HasNextPage",
	})

	return b.Build(&root{})
}

func convertLength(meters float64, unit string) float64 {
	switch unit {
	case "METER":
		return meters
	case "FOOT":
		return meters * 3.28084
	default:
		panic("invalid unit")
	}
}

func resolveCharacters(ids []graphql.ID) *[]interface{} {
	var characters []interface{}
	for _, id := range ids {
		if c := resolveCharacter(id); c != nil {
			characters = append(characters, c)
		}
	}
	return &characters
}

func resolveCharacter(id graphql.ID) interface{} {
	if h, ok := humanData[id]; ok {
		return h
	}
	if d, ok := droidData[id]; ok {
		return d
	}
	return nil
}

type friendsConnectionArgs struct {
	First *int32
	After *graphql.ID
}

type friendsConnection struct {
	IDs  []graphql.ID
	From int
	To   int
}

type friendsEdge struct {
	Cursor graphql.ID
	ID     graphql.ID
}

func newFriendsConnection(ids []graphql.ID, args *friendsConnectionArgs) (*friendsConnection, error) {
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
		from = i
	}

	to := len(ids)
	if args.First != nil {
		to = from + int(*args.First)
		if to > len(ids) {
			to = len(ids)
		}
	}

	return &friendsConnection{
		IDs:  ids,
		From: from,
		To:   to,
	}, nil
}

func encodeCursor(i int) graphql.ID {
	return graphql.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", i+1))))
}

type pageInfo struct {
	StartCursor *graphql.ID
	EndCursor   *graphql.ID
	HasNextPage bool
}

type reviewInput struct {
	Stars      int32
	Commentary *string
}
