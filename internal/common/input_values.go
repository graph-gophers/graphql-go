package common

import (
	"fmt"

	"github.com/neelance/graphql-go/errors"
)

type InputValue struct {
	Name    Ident
	Type    Type
	Default Literal
	Desc    string
	Loc     errors.Location
	TypeLoc errors.Location
}

type InputValueList []*InputValue

func (l InputValueList) Get(name string) *InputValue {
	for _, v := range l {
		if v.Name.Name == name {
			return v
		}
	}
	return nil
}

func ParseInputValue(l *Lexer) *InputValue {
	p := &InputValue{}
	p.Loc = l.Location()
	p.Desc = l.DescComment()
	p.Name = l.ConsumeIdentWithLoc()
	l.ConsumeToken(':')
	p.TypeLoc = l.Location()
	p.Type = ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = ParseLiteral(l, true)
	}
	return p
}

func ParseArgumentDeclList(l *Lexer) InputValueList {
	var args InputValueList
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			args = append(args, ParseInputValue(l))
		}
		l.ConsumeToken(')')
	}
	return args
}

func ParseInputFieldList(typeName string, l *Lexer) InputValueList {
	l.ConsumeToken('{')
	var list InputValueList
	for l.Peek() != '}' {
		list = append(list, ParseInputValue(l))
	}
	if len(list) == 0 {
		l.SyntaxError(fmt.Sprintf(`input type %q must define one or more fields`, typeName))
	}
	l.ConsumeToken('}')
	return list
}
