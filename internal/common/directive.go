package common

import "github.com/graph-gophers/graphql-go/ast"

func ParseDirectives(l *Lexer) ast.DirectiveList {
	var directives ast.DirectiveList
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		d := &ast.Directive{}
		d.Name = l.ConsumeIdentWithLoc()
		d.Name.Loc.Column--
		if l.Peek() == '(' {
			d.Arguments = ParseArgumentList(l)
		}
		directives = append(directives, d)
	}
	return directives
}
