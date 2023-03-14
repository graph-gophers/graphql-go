package common

import (
	"github.com/graph-gophers/graphql-go/ast"
	"github.com/graph-gophers/graphql-go/errors"
)

func ParseType(l *Lexer) ast.Type {
	t := parseNullType(l)
	if l.Peek() == '!' {
		l.ConsumeToken('!')
		return &ast.NonNull{OfType: t}
	}
	return t
}

func parseNullType(l *Lexer) ast.Type {
	if l.Peek() == '[' {
		l.ConsumeToken('[')
		ofType := ParseType(l)
		l.ConsumeToken(']')
		return &ast.List{OfType: ofType}
	}

	return &ast.TypeName{Ident: l.ConsumeIdentWithLoc()}
}

type Resolver func(name string) ast.Type

// ResolveType attempts to resolve a type's name against a resolving function.
// This function is used when one needs to check if a TypeName exists in the resolver (typically a Schema).
//
// In the example below, ResolveType would be used to check if the resolving function
// returns a valid type for Dimension:
//
//	type Profile {
//	   picture(dimensions: Dimension): Url
//	}
//
// ResolveType recursively unwraps List and NonNull types until a NamedType is reached.
func ResolveType(t ast.Type, resolver Resolver) (ast.Type, *errors.QueryError) {
	switch t := t.(type) {
	case *ast.List:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &ast.List{OfType: ofType}, nil
	case *ast.NonNull:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &ast.NonNull{OfType: ofType}, nil
	case *ast.TypeName:
		refT := resolver(t.Name)
		if refT == nil {
			err := errors.Errorf("Unknown type %q.", t.Name)
			err.Rule = "KnownTypeNames"
			err.Locations = []errors.Location{t.Loc}
			return nil, err
		}
		return refT, nil
	default:
		return t, nil
	}
}
