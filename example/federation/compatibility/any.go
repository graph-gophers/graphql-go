package main

import (
	"encoding/json"
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

type key interface {
	isKey()
}

type ProdKey struct {
	ID        *graphql.ID `json:"id"`
	SKU       *string     `json:"sku"`
	Package   *string     `json:"package"`
	Variation *struct {
		ID graphql.ID `json:"id"`
	} `json:"variation"`
}

type DepProdKey struct {
	SKU     string `json:"sku"`
	Package string `json:"package"`
}

type ProdResKey struct {
	Study struct {
		ID graphql.ID `json:"caseNumber"`
	} `json:"study"`
}

type UserKey struct {
	Email graphql.ID `json:"email"`
}

type InvKey struct {
	ID graphql.ID `json:"id"`
}

func (ProdKey) isKey()    {}
func (DepProdKey) isKey() {}
func (ProdResKey) isKey() {}
func (UserKey) isKey()    {}
func (InvKey) isKey()     {}

type Any struct {
	TypeName string `json:"__typename"`
	Key      key
}

func (a *Any) UnmarshalJSON(d []byte) error {
	var temp struct {
		T string `json:"__typename"`
	}
	err := json.Unmarshal(d, &temp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal typename: %w", err)
	}
	a.TypeName = temp.T

	switch a.TypeName {
	case "DeprecatedProduct":
		var p DepProdKey
		err := json.Unmarshal(d, &p)
		if err != nil {
			return fmt.Errorf("failed to unmarshal deprecated product key: %w", err)
		}
		(*a).Key = p
	case "Inventory":
		var i InvKey
		err := json.Unmarshal(d, &i)
		if err != nil {
			return fmt.Errorf("failed to unmarshal inventory key: %w", err)
		}
		(*a).Key = i
	case "Product":
		var p ProdKey
		err := json.Unmarshal(d, &p)
		if err != nil {
			return fmt.Errorf("failed to unmarshal product key: %w", err)
		}
		(*a).Key = p
	case "ProductResearch":
		var r ProdResKey
		err := json.Unmarshal(d, &r)
		if err != nil {
			return fmt.Errorf("failed to unmarshal product research key: %w", err)
		}
		(*a).Key = r
	case "User":
		var u UserKey
		err := json.Unmarshal(d, &u)
		if err != nil {
			return fmt.Errorf("failed to unmarshal product key: %w", err)
		}
		(*a).Key = u
	default:
		return fmt.Errorf("invalid typename %q", a.TypeName)
	}
	return nil
}

func (Any) ImplementsGraphQLType(name string) bool {
	return name == "_Any"
}

func (a *Any) UnmarshalGraphQL(input interface{}) error {
	var data []byte
	switch val := input.(type) {
	case string: // json wrapped in quotes
		v := fmt.Sprint(val) // turns all the `\"` into `"`
		data = []byte(v)
	case map[string]interface{}:
		var err error
		data, err = json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal to json: %w", err)
		}
	default:
		return fmt.Errorf("invalid input type %T", input)
	}

	return json.Unmarshal(data, a)
}
