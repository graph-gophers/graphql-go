package schema

import (
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go/internal/common"
)

func validateEntryPointName(s *Schema, l *common.Lexer) {
	if s == Meta {
		return
	}

	switch name := l.PeekIdent(); name {
	case "query", "mutation", "subscription":
		if prev, ok := s.entryPointNames[name]; ok {
			l.SyntaxError(fmt.Sprintf(`%q provided more than once %s`, name, prev.Loc))
		}
	default:
		l.SyntaxError(fmt.Sprintf(`unexpected %q, expected "query", "mutation" or "subscription"`, name))
	}
}

func validateTypeName(s *Schema, l *common.Lexer) {
	if s == Meta {
		return
	}
	name := l.PeekIdent()
	validatePrefix(name, l)
	if _, ok := Meta.Types[name]; ok {
		l.SyntaxError(fmt.Sprintf(`built-in type %q redefined`, name))
	}
	if prev, ok := s.Types[name]; ok {
		l.SyntaxError(fmt.Sprintf(`%q defined more than once %s`, name, prev.Location()))
	}
}

func validateDirectiveName(s *Schema, l *common.Lexer) {
	if s == Meta {
		return
	}
	name := l.PeekIdent()
	validatePrefix(name, l)
	if _, ok := Meta.Directives[name]; ok {
		l.SyntaxError(fmt.Sprintf(`built-in directive %q redefined`, name))
	}
	if prev, ok := s.Directives[name]; ok {
		l.SyntaxError(fmt.Sprintf(`%q defined more than once %s`, name, prev.Loc))
	}
}

func validatePrefix(name string, l *common.Lexer) {
	if strings.HasPrefix(name, "__") {
		l.SyntaxError(fmt.Sprintf(`%q must not begin with "__", reserved for introspection types`, name))
	}
}
