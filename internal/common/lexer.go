package common

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
)

type syntaxError string

type Lexer struct {
	sc                    *scanner.Scanner
	next                  rune
	descComment           string
	useStringDescriptions bool
}

type Ident struct {
	Name string
	Loc  errors.Location
}

func NewLexer(s string, useStringDescriptions bool) *Lexer {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(s))

	return &Lexer{sc: sc, useStringDescriptions: useStringDescriptions}
}

func (l *Lexer) CatchSyntaxError(f func()) (errRes *errors.QueryError) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(syntaxError); ok {
				errRes = errors.Errorf("syntax error: %s", err)
				errRes.Locations = []errors.Location{l.Location()}
				return
			}
			panic(err)
		}
	}()

	f()
	return
}

func (l *Lexer) Peek() rune {
	return l.next
}

// ConsumeWhitespace consumes whitespace and tokens equivalent to whitespace (e.g. commas and comments).
//
// Consumed comment characters will build the description for the next type or field encountered.
// The description is available from `DescComment()`, and will be reset every time `ConsumeWhitespace()` is
// executed unless l.useStringDescriptions is set.
func (l *Lexer) ConsumeWhitespace() {
	if !l.useStringDescriptions {
		l.descComment = ""
	}
	for {
		l.next = l.sc.Scan()

		if l.next == ',' {
			// Similar to white space and line terminators, commas (',') are used to improve the
			// legibility of source text and separate lexical tokens but are otherwise syntactically and
			// semantically insignificant within GraphQL documents.
			//
			// http://facebook.github.io/graphql/draft/#sec-Insignificant-Commas
			continue
		}

		if l.next == '#' {
			// GraphQL source documents may contain single-line comments, starting with the '#' marker.
			//
			// A comment can contain any Unicode code point except `LineTerminator` so a comment always
			// consists of all code points starting with the '#' character up to but not including the
			// line terminator.

			l.consumeComment()
			continue
		}

		break
	}
}

// consumeDescription optionally consumes a description based on the June 2018 graphql spec if any are present.
//
// Single quote strings are also single line. Triple quote strings can be multi-line. Triple quote strings
// whitespace trimmed on both ends.
// If a description is found, consume any following comments as well
//
// http://facebook.github.io/graphql/June2018/#sec-Descriptions
func (l *Lexer) consumeDescription() bool {
	// If the next token is not a string, we don't consume it
	if l.next == scanner.String {
		// a triple quote string is an empty "string" followed by an open quote due to the way the parser treats strings as one token
		l.descComment = ""
		tokenText := l.sc.TokenText()
		if l.sc.Peek() == '"' {
			// Consume the third quote
			l.next = l.sc.Next()
			l.consumeTripleQuoteComment()
		} else {
			l.consumeStringComment(tokenText)
		}
		return true
	}
	return false
}

func (l *Lexer) ConsumeIdent() string {
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return name
}

func (l *Lexer) ConsumeIdentWithLoc() Ident {
	loc := l.Location()
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return Ident{name, loc}
}

func (l *Lexer) ConsumeKeyword(keyword string) {
	if l.next != scanner.Ident || l.sc.TokenText() != keyword {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %q", l.sc.TokenText(), keyword))
	}
	l.ConsumeWhitespace()
}

func (l *Lexer) ConsumeLiteral() *BasicLit {
	lit := &BasicLit{Type: l.next, Text: l.sc.TokenText()}
	l.ConsumeWhitespace()
	return lit
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.ConsumeWhitespace()
}

func (l *Lexer) DescComment() string {
	if l.useStringDescriptions {
		if l.consumeDescription() {
			l.ConsumeWhitespace()
		}
	}
	return l.descComment
}

func (l *Lexer) SyntaxError(message string) {
	panic(syntaxError(message))
}

func (l *Lexer) Location() errors.Location {
	return errors.Location{
		Line:   l.sc.Line,
		Column: l.sc.Column,
	}
}

func (l *Lexer) consumeTripleQuoteComment() {
	if l.next != '"' {
		panic("consumeTripleQuoteComment used in wrong context: no third quote?")
	}

	if l.descComment != "" {
		l.descComment += "\n"
	}

	comment := ""
	numQuotes := 0
	for {
		l.next = l.sc.Next()
		if l.next == '"' {
			numQuotes++
		} else {
			numQuotes = 0
		}
		comment += string(l.next)
		if numQuotes == 3 || l.next == scanner.EOF {
			break
		}
	}
	l.descComment += strings.TrimSpace(comment[:len(comment)-numQuotes])
}

func (l *Lexer) consumeStringComment(str string) {
	if l.descComment != "" {
		l.descComment += "\n"
	}

	value, err := strconv.Unquote(str)
	if err != nil {
		panic(err)
	}
	l.descComment += value
}

// consumeComment consumes all characters from `#` to the first encountered line terminator.
// The characters are appended to `l.descComment`.
func (l *Lexer) consumeComment() {
	if l.next != '#' {
		panic("consumeComment used in wrong context")
	}

	// TODO: count and trim whitespace so we can dedent any following lines.
	if l.sc.Peek() == ' ' {
		l.sc.Next()
	}

	if l.descComment != "" && !l.useStringDescriptions {
		// TODO: use a bytes.Buffer or strings.Builder instead of this.
		l.descComment += "\n"
	}

	for {
		next := l.sc.Next()
		if next == '\r' || next == '\n' || next == scanner.EOF {
			break
		}

		if !l.useStringDescriptions {
			// TODO: use a bytes.Buffer or strings.Build instead of this.
			l.descComment += string(next)
		}
	}
}
