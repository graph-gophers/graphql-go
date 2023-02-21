package main

import (
	"fmt"
)

// State represents a type-safe enum in Go which corresponds to the State enum in the GraphQL schema.
type State int

const (
	Backlog State = iota // default value
	TODO
	InProg
	Done
)

// the items in this array must have the exact same order as the corresponding constants above
var states = [...]string{"BACKLOG", "TODO", "INPROG", "DONE"}

func (s State) String() string { return states[s] }

func (s *State) Deserialize(str string) {
	var found bool
	for i, st := range states {
		if st == str {
			found = true
			(*s) = State(i)
		}
	}
	if !found {
		panic("invalid value for enum State: " + str)
	}
}

// ImplementsGraphQLType instructs the GraphQL server that the State enum can be represented by the State type in Golang.
// If this method is missing we would get a runtime error that the State type can't be assigned a string. However, since
// this type implements the State enum, the server will try to call its [State.UnmarshalGraphQL] method to set a value.
func (State) ImplementsGraphQLType(name string) bool {
	return name == "State"
}

// UnmarshalGraphQL tries to unmarshal a type from a given GraphQL value.
func (s *State) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		s.Deserialize(input)
	default:
		err = fmt.Errorf("wrong type for State: %T", input)
	}
	return err
}
