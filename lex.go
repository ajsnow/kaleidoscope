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
// func (t *token) String() string {
// 	switch {
// 	case t.kind == tokError:
// 		return t.val
// 	case t.kind == tokEOF:
// 		return "EOF"
// 	case t.kind > tokKeyword:
// 		return fmt.Sprintf("<%s>", t.val)
// 	case len(t.val) > 10:
// 		return fmt.Sprintf("%.10q...", t.val) // Limit the max width for long tokens
// 	case t.kind == tokSpace:
// 		return "_"
// 	default:
// 		return t.val
// 	}
// }

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
	tokBinary
	tokUnary
	tokVariable

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
	"def":    tokDefine,
	"extern": tokExtern,
	"if":     tokIf,
	"then":   tokThen,
	"else":   tokElse,
	"for":    tokFor,
	"in":     tokIn,
	"binary": tokBinary,
	"unary":  tokUnary,
	"var":    tokVariable,
}

var op = map[rune]tokenType{
	'=': tokEqual,
	'+': tokPlus,
	'-': tokMinus,
	'*': tokStar,
	'/': tokSlash,
	'<': tokLessThan,
}

// uopType differentiates a user-defined unary, binary or not found operator.
type uopType int

const (
	uopNOP uopType = iota
	uopUnaryOp
	uopBinaryOp
)

// uop maps user defined operators number of operands
var uop = map[rune]uopType{}

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name  string  // name of input file; used in error reports
	input string  // input being scanned
	state stateFn // next lexing function to be called
	pos   Pos     // current position in input
	start Pos     // beginning position of the current token
	width Pos     // width of last rune read from input
	// lastPos    Pos          // position of most recent token returned from lexNext
	tokens     chan<- token // channel of lexed items
	parenDepth int          // nested layers of paren expressions
}

// Lex creates and runs a new lexer from the input string.
func Lex(name, input string) <-chan token {
	ch := make(chan token, 10)
	l := &lexer{
		name:   name,
		input:  input,
		tokens: ch,
	}
	go l.run()
	return ch
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
		l.emit(tokEOF)
		return nil
	case isSpace(r):
		l.backup() // could be a single space
		return lexSpace
	case isEOL(r):
		l.start = l.pos
		return lexTopLevel
	case r == ';':
		l.emit(tokSemicolon)
		return lexTopLevel
	case r == ',':
		l.emit(tokComma)
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
	case op[r] > tokUserBinaryOp:
		l.emit(op[r])
		return lexTopLevel
	case uop[r] == uopBinaryOp:
		l.emit(tokUserBinaryOp)
		return lexTopLevel
	case uop[r] == uopUnaryOp:
		l.emit(tokUserUnaryOp)
		return lexTopLevel
	default:
		return l.errorf("unrecognized character: %#U", r)
	}
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
	numberish := "0123456789.-+xabcdefABCDEF" // let the parser check for errors.
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
		case isAlphaNumeric(r):
			// absorb
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if key[word] > tokKeyword {
				l.emit(key[word])
				switch word {
				case "binary":
					r = l.peek()
					uop[r] = uopBinaryOp
				case "unary":
					r = l.peek()
					uop[r] = uopBinaryOp
				}
			} else {
				l.emit(tokIdentifier)
			}
			break Loop
		}
	}
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
