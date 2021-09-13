package types

import "fmt"

type Map map[string]interface {}

func (Map) ImplementsGraphQLType(name string) bool {
	return name == "Map" 
}

func (j *Map) UnmarshalGraphQL(input interface{}) error {
	json, ok := input.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type")
	}

	*j = json
	return nil
}
