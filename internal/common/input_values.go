package common

import (
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
