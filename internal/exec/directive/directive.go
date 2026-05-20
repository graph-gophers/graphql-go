package directive

import (
	"fmt"
	"reflect"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
)

func ShouldSkipSelection(vars map[string]any, directives ast.DirectiveList) (bool, error) {
	if d := directives.Get("skip"); d != nil {
		ok, err := decodeBoolArg(vars, d)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	if d := directives.Get("include"); d != nil {
		ok, err := decodeBoolArg(vars, d)
		if err != nil {
			return false, err
		}
		if !ok {
			return true, nil
		}
	}

	return false, nil
}

func decodeBoolArg(vars map[string]any, d *ast.Directive) (bool, error) {
	p := packer.ValuePacker{ValueType: reflect.TypeFor[bool]()}
	v, err := p.Pack(d.Arguments.MustGet("if").Deserialize(vars))
	if err != nil {
		return false, fmt.Errorf("%s", err)
	}
	return v.Bool(), nil
}
