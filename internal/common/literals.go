package common

import (
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
)

func ParseLiteral(l *Lexer, constOnly bool) ast.Value {
	loc := l.Location()
	switch l.Peek() {
	case '$':
		if constOnly {
			l.SyntaxError("variable not allowed")
			panic("unreachable")
		}
		l.ConsumeToken('$')
		return &ast.Variable{Name: l.ConsumeIdent(), Loc: loc}

	case scanner.Int, scanner.Float, scanner.String, scanner.Ident:
		lit := l.ConsumeLiteral()
		if lit.Type == scanner.Ident && lit.Text == "null" {
			return &ast.NullValue{Loc: loc}
		}
		lit.Loc = loc
		return lit
	case '-':
		l.ConsumeToken('-')
		lit := l.ConsumeLiteral()
		lit.Text = "-" + lit.Text
		lit.Loc = loc
		return lit
	case '[':
		l.ConsumeToken('[')
		var list []ast.Value
		for l.Peek() != ']' {
			list = append(list, ParseLiteral(l, constOnly))
		}
		l.ConsumeToken(']')
		return &ast.ListValue{Values: list, Loc: loc}

	case '{':
		l.ConsumeToken('{')
		var fields []*ast.ObjectField
		for l.Peek() != '}' {
			name := l.ConsumeIdentWithLoc()
			l.ConsumeToken(':')
			value := ParseLiteral(l, constOnly)
			fields = append(fields, &ast.ObjectField{Name: name, Value: value})
		}
		l.ConsumeToken('}')
		return &ast.ObjectValue{Fields: fields, Loc: loc}

	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
