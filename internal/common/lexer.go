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
	sc                       *scanner.Scanner
	next                     rune
	descComment              string
	noCommentsAsDescriptions bool
}

type Ident struct {
	Name string
	Loc  errors.Location
}

func NewLexer(s string, noCommentsAsDescriptions bool) *Lexer {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(s))

	return &Lexer{sc: sc, noCommentsAsDescriptions: noCommentsAsDescriptions}
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

// Consume whitespace and tokens equivalent to whitespace (e.g. commas and comments).
//
// Consumed comment characters will build the description for the next type or field encountered.
// The description is available from `DescComment()`, and will be reset every time `Consume()` is
// executed.
func (l *Lexer) Consume(allowNewStyleDescription bool) {
	l.descComment = ""
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

		if l.next == scanner.String && allowNewStyleDescription {
			// Instead of comments, strings are used to encode descriptions in the June 2018 graphql spec.
			// We can handle both, but there's an option to disable the old comment based descriptions and treat comments
			// as comments.
			// Single quote strings are also single line. Triple quote strings can be multi-line. Triple quote strings
			// whitespace trimmed on both ends.
			//
			// http://facebook.github.io/graphql/June2018/#sec-Descriptions

			// a triple quote string is an empty "string" followed by an open quote due to the way the parser treats strings as one token
			tokenText := l.sc.TokenText()
			if l.sc.Peek() == '"' {
				// Consume the third quote
				l.next = l.sc.Next()
				l.consumeTripleQuoteComment()
				continue
			}
			l.consumeStringComment(tokenText)
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
	l.Consume(true)
}

func (l *Lexer) ConsumeLiteral() *BasicLit {
	lit := &BasicLit{Type: l.next, Text: l.sc.TokenText()}
	l.Consume(false)
	return lit
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.Consume(false)
}

func (l *Lexer) DescComment() string {
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
		next := l.sc.Next()
		if next == '"' {
			numQuotes++
		} else {
			numQuotes = 0
		}
		comment += string(next)
		if numQuotes == 3 || next == scanner.EOF {
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

	if l.descComment != "" && !l.noCommentsAsDescriptions {
		// TODO: use a bytes.Buffer or strings.Builder instead of this.
		l.descComment += "\n"
	}

	for {
		next := l.sc.Next()
		if next == '\r' || next == '\n' || next == scanner.EOF {
			break
		}

		if !l.noCommentsAsDescriptions {
			// TODO: use a bytes.Buffer or strings.Build instead of this.
			l.descComment += string(next)
		}
	}
}
