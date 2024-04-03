package query

import (
	"fmt"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

const (
	Query        ast.OperationType = "QUERY"
	Mutation     ast.OperationType = "MUTATION"
	Subscription ast.OperationType = "SUBSCRIPTION"
)

func Parse(queryString string) (*ast.ExecutableDefinition, *errors.QueryError) {
	l := common.NewLexer(queryString, false)

	var execDef *ast.ExecutableDefinition
	err := l.CatchSyntaxError(func() { execDef = parseExecutableDefinition(l) })
	if err != nil {
		return nil, err
	}

	return execDef, nil
}

func parseExecutableDefinition(l *common.Lexer) *ast.ExecutableDefinition {
	ed := &ast.ExecutableDefinition{}
	l.ConsumeWhitespace()
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			op := &ast.OperationDefinition{Type: Query, Loc: l.Location()}
			op.Selections = parseSelectionSet(l)
			ed.Operations = append(ed.Operations, op)
			continue
		}

		loc := l.Location()
		switch x := l.ConsumeIdent(); x {
		case "query":
			op := parseOperation(l, Query)
			op.Loc = loc
			ed.Operations = append(ed.Operations, op)

		case "mutation":
			ed.Operations = append(ed.Operations, parseOperation(l, Mutation))

		case "subscription":
			ed.Operations = append(ed.Operations, parseOperation(l, Subscription))

		case "fragment":
			frag := parseFragment(l)
			frag.Loc = loc
			ed.Fragments = append(ed.Fragments, frag)

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "fragment"`, x))
		}
	}
	return ed
}

func parseOperation(l *common.Lexer, opType ast.OperationType) *ast.OperationDefinition {
	op := &ast.OperationDefinition{Type: opType}
	op.Name.Loc = l.Location()
	if l.Peek() == scanner.Ident {
		op.Name = l.ConsumeIdentWithLoc()
	}
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			loc := l.Location()
			l.ConsumeToken('$')
			iv := common.ParseInputValue(l)
			iv.Loc = loc
			op.Vars = append(op.Vars, iv)
		}
		l.ConsumeToken(')')
	}
	op.Directives = common.ParseDirectives(l)
	op.Selections = parseSelectionSet(l)
	return op
}

func parseFragment(l *common.Lexer) *ast.FragmentDefinition {
	f := &ast.FragmentDefinition{}
	f.Name = l.ConsumeIdentWithLoc()
	l.ConsumeKeyword("on")
	f.On = ast.TypeName{Ident: l.ConsumeIdentWithLoc()}
	f.Directives = common.ParseDirectives(l)
	f.Selections = parseSelectionSet(l)
	return f
}

func parseSelectionSet(l *common.Lexer) []ast.Selection {
	var sels []ast.Selection
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		sels = append(sels, parseSelection(l))
	}
	l.ConsumeToken('}')
	return sels
}

func parseSelection(l *common.Lexer) ast.Selection {
	if l.Peek() == '.' {
		return parseSpread(l)
	}
	return parseFieldDef(l)
}

func parseFieldDef(l *common.Lexer) *ast.Field {
	f := &ast.Field{}
	f.Alias = l.ConsumeIdentWithLoc()
	f.Name = f.Alias
	if l.Peek() == ':' {
		l.ConsumeToken(':')
		f.Name = l.ConsumeIdentWithLoc()
	}
	if l.Peek() == '(' {
		f.Arguments = common.ParseArgumentList(l)
	}
	f.Directives = common.ParseDirectives(l)
	if l.Peek() == '{' {
		f.SelectionSetLoc = l.Location()
		f.SelectionSet = parseSelectionSet(l)
	}
	return f
}

func parseSpread(l *common.Lexer) ast.Selection {
	loc := l.Location()
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	l.ConsumeToken('.')

	f := &ast.InlineFragment{Loc: loc}
	if l.Peek() == scanner.Ident {
		ident := l.ConsumeIdentWithLoc()
		if ident.Name != "on" {
			fs := &ast.FragmentSpread{
				Name: ident,
				Loc:  loc,
			}
			fs.Directives = common.ParseDirectives(l)
			return fs
		}
		f.On = ast.TypeName{Ident: l.ConsumeIdentWithLoc()}
	}
	f.Directives = common.ParseDirectives(l)
	f.Selections = parseSelectionSet(l)
	return f
}
