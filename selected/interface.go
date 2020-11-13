package selected

import (
	"fmt"
)

type Kind int

func (k Kind) String() string {
	switch k {
	case FieldKind:
		return "field"
	case TypeAssertionKind:
		return "type_assertion"
	case TypenameFieldKind:
		return "typename_field"
	default:
		panic(fmt.Errorf("invalid kind %d received", k))
	}
}

const (
	FieldKind Kind = iota
	TypeAssertionKind
	TypenameFieldKind
)

type Selection interface {
	Kind() Kind
}

type Field interface {
	Selection
	Identifier() string
	Aliased() string
	Children() []Selection
}

type TypeAssertion interface {
	Selection
	Type() string
	Children() []Selection
}

type TypenameField interface {
	Selection
	Type() string
	Aliased() string
}

func Dump(selection Selection) {
	if selection == nil {
		fmt.Println("Selection <nil>")
		return
	}

	var print func(string, Selection)
	print = func(indent string, sel Selection) {
		switch v := sel.(type) {
		case Field:
			fmt.Printf(indent+"Field %s (%s)\n", v.Identifier(), v.Aliased())
			for _, subSel := range v.Children() {
				print(indent+"  ", subSel)
			}
		case TypeAssertion:
			fmt.Printf(indent+"TypeAssertion %s\n", v.Type())
			for _, subSel := range v.Children() {
				print(indent+"  ", subSel)
			}
		case TypenameField:
			fmt.Printf(indent+"TypenameField %s (%s)\n", v.Type(), v.Aliased())
		default:
			panic(fmt.Errorf("invalid selection %T received", v))
		}
	}

	print("", selection)
}
