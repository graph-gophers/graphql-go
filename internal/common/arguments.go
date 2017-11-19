package common

type Argument struct {
	Name  Ident
	Value Literal
}

type ArgumentList []Argument

func (l ArgumentList) Get(name string) (Literal, bool) {
	for _, arg := range l {
		if arg.Name.Name == name {
			return arg.Value, true
		}
	}
	return nil, false
}

func (l ArgumentList) MustGet(name string) Literal {
	value, ok := l.Get(name)
	if !ok {
		panic("argument not found")
	}
	return value
}

func ParseArguments(l *Lexer) ArgumentList {
	var args ArgumentList
	l.ConsumeToken('(')
	for l.Peek() != ')' {
		name := l.ConsumeIdentWithLoc()
		l.ConsumeToken(':')
		value := ParseLiteral(l, false)
		args = append(args, Argument{Name: name, Value: value})
	}
	l.ConsumeToken(')')
	return args
}
