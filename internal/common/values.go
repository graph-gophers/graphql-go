package common

import (
	"github.com/graph-gophers/graphql-go/ast"
)

func ParseInputValue(l *Lexer) *ast.InputValueDefinition {
	p := &ast.InputValueDefinition{}
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
	p.Directives = ParseDirectives(l)
	return p
}

func ParseArgumentList(l *Lexer) ast.ArgumentList {
	var args ast.ArgumentList
	l.ConsumeToken('(')
	for l.Peek() != ')' {
		name := l.ConsumeIdentWithLoc()
		l.ConsumeToken(':')
		value := ParseLiteral(l, false)
		directives := ParseDirectives(l)
		args = append(args, &ast.Argument{
			Name:       name,
			Value:      value,
			Directives: directives,
		})
	}
	l.ConsumeToken(')')
	return args
}
