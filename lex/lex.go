package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// token
// A lexeme is the basic unit of lexicographical meaning.
// Aren't circular definitions fun?
type token struct {
	kind tokenType // The kind of token with which we're dealing.
	pos  Pos       // The byte offset of the beginning of the token with respect to the beginning of the input.
	val  string    // The token's value. Error message for lexError; otherwise, the token's constituent text.
}

// Defining the String function satisfies the Stinger interface.
// Satisfying Stringer allows package fmt to pretty-print our tokens.
func (t *token) String() string {
	switch {
	case t.kind == tokError:
		return t.val
	case t.kind == tokEOF:
		return "EOF"
	case t.kind > tokKeyword:
		return fmt.Sprintf("<%s>", t.val)
	case len(t.val) > 10:
		return fmt.Sprintf("%.10q...", t.val) // Limit the max width for long tokens
	case t.kind == tokSpace:
		return "space, the most useful token"
	default:
		return t.val
	}
}

// tokenType identifies the type of a token.
type tokenType int

const (
	// special
	tokError tokenType = iota // error occurred
	tokEOF
	tokComment

	// punctuation
	tokSpace
	tokSemicolon
	tokComma
	tokLeftParen
	tokRightParen

	// literals
	tokNumber

	// identifiers
	tokIdentifier

	// keywords
	tokKeyword // used to delimit keywords
	tokDefine
	tokExtern
	tokIf
	tokThen
	tokElse
	tokFor
	tokIn
	tokVar

	// operators
	tokUserUnaryOp
	tokUserBinaryOp
	tokEqual
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokLessThan
)

var key = map[string]tokenType{
	"define": tokDefine,
	"extern": tokExtern,
	"if":     tokIf,
	"then":   tokThen,
	"else":   tokElse,
	"for":    tokFor,
	"in":     tokIn,
	"var":    tokVar,
}

var op = map[rune]tokenType{
	'=': tokEqual,
	'+': tokPlus,
	'-': tokMinus,
	'*': tokStar,
	'/': tokSlash,
	'<': tokLessThan,
}

const eof = -1

// -- move to ast --
// Pos defines a byte offset from the beginning of the input text.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// In text/template/parse/node.go Rob adds an unexported() method to Pos
// I do know why he did that rather than make Pos -> pos
// -- end move to ast --

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string     // name of input file; used in error reports
	input      string     // input being scanned
	state      stateFn    // next lexing function to be called
	pos        Pos        // current position in input
	start      Pos        // beginning position of the current token
	width      Pos        // width of last rune read from input
	lastPos    Pos        // position of most recent token returned from lexNext
	tokens     chan token // channel of lexed items
	parenDepth int        // nested layers of paren expressions
}

// next returns the next rune from the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

// ignore skips the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// acceptRun consumes a run of runes from valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// errorf sending an error token and terminates the scan by passing nil as the next stateFn
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{
		kind: tokError,
		pos:  l.start,
		val:  fmt.Sprintf(format, args...)}
	return nil
}

// emit passes the current token.
func (l *lexer) emit(tt tokenType) {
	l.tokens <- token{
		kind: tt,
		pos:  l.start,
		val:  l.input[l.start:l.pos],
	}
	l.start = l.pos
}

// lex creates and runs a new lexer from the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:   name,
		input:  input,
		tokens: make(chan token),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexTopLevel; l.state != nil; {
		l.state = l.state(l)
		// Println(runtime.FuncForPC(reflect.ValueOf(l.state).Pointer()).Name())
	}
	close(l.tokens) // Tells the client no more tokens will be delivered.
}

// State Functions

// lexTopLevel
func lexTopLevel(l *lexer) stateFn {
	// Either whitespace, an empty line, a comment,
	// a number, a paren, identifier, or unary operator.
	r := l.next()
	switch {
	case r == eof:
		return nil
	case isSpace(r):
		l.backup() // could be a single space
		return lexSpace
	case isEOL(r):
		l.start = l.pos
		return lexTopLevel
	case r == '#':
		return lexComment
	case r == '(':
		l.parenDepth++
		l.emit(tokLeftParen)
		return lexTopLevel
	case r == ')':
		l.parenDepth--
		l.emit(tokRightParen)
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
		return lexTopLevel
	case '0' <= r && r <= '9':
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexIdentifer
	default:
		// since the user can define arbitrary single-rune operators, anything else could be valid
		// e.g. "∆ ¿¡ 7 !? <-- what" could be a perfectly valid expression
		l.backup()
		return lexOperator
		// return l.errorf("unrecognized character: %#U", r)
	}
	return l.errorf("This shouldn't happen!")
}

func lexSpace(l *lexer) stateFn {
	for isSpace(l.next()) {
	}
	l.backup()
	l.emit(tokSpace)
	return lexTopLevel
}

func lexComment(l *lexer) stateFn {
	for !isEOL(l.next()) { // this fails if # ---- EOF check others for similar fault !!!!!
	}
	l.backup()
	l.emit(tokComment)
	return lexTopLevel
}

func lexNumber(l *lexer) stateFn {
	numberish := "0123456789-+.xabcdefABCDEF" // let the parser check for errors.
	l.acceptRun(numberish)
	// if isAlphaNumeric(l.peek()) { // probably a mistyped identifier
	// 	l.next()
	// 	return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	// }
	l.emit(tokNumber)
	return lexTopLevel
}

func lexIdentifer(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r): // current fails for binary∆ / unary etc
			// absorb
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if key[word] > tokKeyword {
				l.emit(key[word])
			} else {
				l.emit(tokIdentifier)
			}
			break Loop
		}
	}
	return lexTopLevel
}

//BROKEN!
func lexOperator(l *lexer) stateFn {
	r := l.next()
	if op[r] > tokUserBinaryOp {
		l.emit(op[r])
		return lexTopLevel
	}
	l.emit(tokUserBinaryOp)
	return lexTopLevel
}

// isSpace reports whether r is a space character
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEOL reports whether r is an end-of-line character
func isEOL(r rune) bool {
	return r == '\n' || r == '\r'
}

// isValidIdefRune reports if r may be part of an identifier name
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
