package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
)

// token represents the basic lexicographical units of the language.
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

// The list of tokenTypes.
const (
	// special
	tokEndOfTokens tokenType = iota // finished tokenizing all input
	tokError                        // error occurred
	tokNewFile
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
	tokKeyword // used to delineate keywords
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
	tokUserUnaryOp // additionally used to delineate operators
	tokUserBinaryOp
	tokEqual
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokLessThan
)

// key maps keywords strings to their tokenType.
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

// op maps built-in operators to tokenTypes
// As this should never be written to, it is safe to share between lexer goroutines.
var op = map[rune]tokenType{
	'=': tokEqual,
	'+': tokPlus,
	'-': tokMinus,
	'*': tokStar,
	'/': tokSlash,
	'<': tokLessThan,
}

// userOpType differentiates a user-defined unary, binary or not found operator.
type userOpType int

const (
	uopNOP userOpType = iota // Signals that the rune is *not* a user operator.
	uopUnaryOp
	uopBinaryOp
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	files         chan *os.File       // files to be lexed
	scanner       *bufio.Scanner      // scanner is a buffered interface to the current file
	name          string              // name of current input file; used in error reports
	line          string              // current line being scanned
	state         stateFn             // next lexing function to be called
	pos           Pos                 // current position in input
	start         Pos                 // beginning position of the current token
	width         Pos                 // width of last rune read from input
	lineCount     int                 // number of lines seen in the current file
	parenDepth    int                 // nested layers of paren expressions
	tokens        chan token          // channel of lexed items
	userOperators map[rune]userOpType // userOperators maps user defined operators to number of operands

	printTokens bool // print tokens before sending
}

// Lex creates and runs a new lexer.
func Lex(printTokens bool) *lexer {
	l := &lexer{
		files:         make(chan *os.File, 10),
		tokens:        make(chan token, 10),
		userOperators: map[rune]userOpType{},
		printTokens:   printTokens,
	}
	go l.run()
	return l
}

// Add adds the given file to the lexer's file queue
func (l *lexer) Add(f *os.File) {
	l.files <- f
}

// Done signals that the user is finished Add()ing files
// and that the lexer goroutine should stop once it has
// finished processing all files currently in its queue.
func (l *lexer) Done() {
	close(l.files)
}

// Tokens returns a read-only channel of tokens that can
// be printed or parsed.
func (l *lexer) Tokens() <-chan token {
	return l.tokens
}

// l.next() returns eof to signal end of file to a stateFn.
const eof = -1

// word returns the value of the token that would be emitted if
// l.emit() were to be called.
func (l *lexer) word() string {
	return l.line[l.start:l.pos]
}

// next returns the next rune from the input and advances the scan.
// It returns the eof constant (-1) if the scanner is at the end of
// the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.line) {
		if l.scanner.Scan() {
			l.line = l.scanner.Text() + "\n"
			l.pos = 0
			l.start = 0
			l.width = 0
		} else {
			l.width = 0
			return eof
		}
	}
	r, w := utf8.DecodeRuneInString(l.line[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	// spew.Printf("Rune: %q", r)
	return r
}

// peek returns the next rune without moving the scan forward.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup moves the scan back one rune.
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
	t := token{
		kind: tokError,
		pos:  l.start,
		val:  fmt.Sprintf(format, args...)}
	if l.printTokens {
		spew.Dump(t)
	}
	l.tokens <- t
	return nil
}

// emit passes the current token.
func (l *lexer) emit(tt tokenType) {
	t := token{
		kind: tt,
		pos:  l.start,
		val:  l.word(),
	}
	if l.printTokens {
		spew.Dump(t)
	}
	l.tokens <- t
	l.start = l.pos
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for {
		f, ok := <-l.files
		if !ok {
			close(l.tokens) // tokDONE is the zero value of a token, so we don't need to send it.
			break
		}

		// reset Lexer for new file.
		l.name = f.Name()
		l.scanner = bufio.NewScanner(f)
		l.line = ""
		l.pos = 0
		l.start = 0
		l.width = 0
		l.parenDepth = 0

		// emit a new file token for the parser.
		t := token{
			kind: tokNewFile,
			val:  l.name,
		}
		if l.printTokens {
			spew.Dump(t)
		}
		l.tokens <- t

		// run state machine for the lexer.
		for l.state = lexTopLevel; l.state != nil; {
			l.state = l.state(l)
			// spew.Println("State:", runtime.FuncForPC(reflect.ValueOf(l.state).Pointer()).Name())
		}

		// close file handle
		f.Close()
	}
}

// State Functions

// lexTopLevel lexes any top level statement. Because our language is simple,
// our lexer rarely needs to know its prior state and therefore this amounts
// to the giant-switch style of lexing. Nevertheless, the stateFn technique
// allows us to easy extend our lexer to more complex grammars.
func lexTopLevel(l *lexer) stateFn {
	// Either whitespace, an empty line, a comment,
	// a number, a paren, identifier, or unary operator.
	r := l.next()
	switch {
	case r == eof:
		return nil
	case isSpace(r):
		l.backup()
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
			return l.errorf("unexpected right paren")
		}
		return lexTopLevel
	case '0' <= r && r <= '9', r == '.':
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexIdentifer
	case op[r] > tokUserBinaryOp:
		l.emit(op[r])
		return lexTopLevel
	case l.userOperators[r] == uopBinaryOp:
		l.emit(tokUserBinaryOp)
		return lexTopLevel
	case l.userOperators[r] == uopUnaryOp:
		l.emit(tokUserUnaryOp)
		return lexTopLevel
	default:
		return l.errorf("unrecognized character: %#U", r)
	}
}

// lexSpace globs contiguous whitespace.
func lexSpace(l *lexer) stateFn {
	globWhitespace(l)
	return lexTopLevel
}

// globWhitespace globs contiguous whitespace. (Sometimes we
// don't want to return to lexTopLevel after doing this.)
func globWhitespace(l *lexer) {
	for isSpace(l.next()) {
	}
	l.backup()
	if l.start != l.pos {
		l.emit(tokSpace)
	}
}

// lexComment runs from '#' to the end of line or end of file.
func lexComment(l *lexer) stateFn {
	// for !isEOL(l.next()) {
	// }
	// l.backup()
	l.pos = Pos(len(l.line))
	l.emit(tokComment)
	return lexTopLevel
}

// lexNumber globs potential number-like strings. We let the parser
// verify that the token is actually a valid number.
// e.g. "3.A.8" could be emitted by this function.
func lexNumber(l *lexer) stateFn {
	l.acceptRun("0123456789.xabcdefABCDEF")
	// if isAlphaNumeric(l.peek()) { // probably a mistyped identifier
	// 	l.next()
	// 	return l.errorf("bad number syntax: %q", l.word())
	// }
	l.emit(tokNumber)
	return lexTopLevel
}

// lexIdentfier globs unicode alpha-numerics, determines if they
// represent a keyword or identifier, and output the appropriate
// token. For the "binary" & "unary" keywords, we need to add their
// associated user-defined operator to our map so that we can
// identify it later.
func lexIdentifer(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb
		default:
			l.backup()
			word := l.word()
			if key[word] > tokKeyword { // We already know it's not an operator.
				l.emit(key[word])
				switch word {
				case "binary":
					return lexUserBinaryOp
				case "unary":
					return lexUserUnaryOp
				}
			} else {
				l.emit(tokIdentifier)
			}
			return lexTopLevel
		}
	}
}

// lexUserBinaryOp checks for spaces and then identifies and maps.
// the newly defined user operator.
func lexUserBinaryOp(l *lexer) stateFn {
	globWhitespace(l)
	r := l.next()
	l.userOperators[r] = uopBinaryOp
	l.emit(tokUserBinaryOp)
	return lexTopLevel
}

// lexUserBinaryOp checks for spaces and then identifies and maps.
// the newly defined user operator.
func lexUserUnaryOp(l *lexer) stateFn {
	globWhitespace(l)
	r := l.next()
	l.userOperators[r] = uopUnaryOp
	l.emit(tokUserUnaryOp)
	return lexTopLevel
}

// Helper Functions

// isSpace reports whether r is whitespace.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEOL reports whether r is an end-of-line character or an EOF.
func isEOL(r rune) bool {
	return r == '\n' || r == '\r' || r == eof
}

// isValidIdefRune reports if r may be part of an identifier name.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
