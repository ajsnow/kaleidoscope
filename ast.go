package main

import (
	"fmt"
	"os"
	"strconv"
	"text/scanner"

	"github.com/ajsnow/llvm"
)

// Parsing Functions
func parseNumericExpr() exprAST {
	val, err := strconv.ParseFloat(s.TokenText(), 64)
	if err != nil {
		return Error("invalid number: " + s.TokenText())
	}
	result := &numAST{val}
	token = s.Scan()
	return result
}

func parseParenExpr() exprAST {
	token = s.Scan()
	v := parseExpression()
	if v == nil {
		return nil
	}

	if token != ')' {
		return Error("expected ')'")
	}
	token = s.Scan()
	return v
}

func parseIdentifierExpr() exprAST {
	name := s.TokenText()

	token = s.Scan()
	if token != '(' { // variable reference
		return &varAST{name}
	}

	// function call
	args := []exprAST{}
	for token = s.Scan(); token != ')'; { //token = s.Scan() {
		switch token {
		case ',':
			token = s.Scan()
		default:
			arg := parseExpression()
			if arg == nil {
				return nil
			}
			args = append(args, arg)
		}
	}
	token = s.Scan()
	return &callAST{name, args}
}

func parseIfExpr() exprAST {
	// if
	token = s.Scan()
	ifE := parseExpression()
	if ifE == nil {
		return Error("expected condition after 'if'")
	}

	// then
	if s.TokenText() != "then" {
		return Error("expected 'then' after if condition")
	}
	token = s.Scan()
	thenE := parseExpression()
	if thenE == nil {
		return Error("expected expression after 'then'")
	}

	// else
	if s.TokenText() != "else" {
		return Error("expected 'else' after then expr")
	}
	token = s.Scan()
	elseE := parseExpression()
	if elseE == nil {
		return Error("expected expression after 'else'")
	}

	return &ifAST{ifE, thenE, elseE}
}

func parseForExpr() exprAST {
	token = s.Scan()
	if token != scanner.Ident {
		return Error("expected identifier after 'for'")
	}
	counter := s.TokenText()

	token = s.Scan()
	if token != '=' {
		return Error("expected '=' after 'for " + counter + "'")
	}

	token = s.Scan()
	start := parseExpression()
	if start == nil {
		return Error("expected expression after 'for " + counter + " ='")
	}
	if token != ',' {
		return Error("expected ',' after 'for' start expression")
	}

	token = s.Scan()
	end := parseExpression()
	if end == nil {
		return Error("expected end expression after 'for' start expression")
	}

	// optional step
	var step exprAST
	if token == ',' {
		token = s.Scan()
		step = parseExpression()
		if step == nil {
			return Error("invalid step expression after 'for'")
		}
	}

	if s.TokenText() != "in" {
		return Error("expected 'in' after 'for' sub-expression")
	}

	token = s.Scan()
	body := parseExpression()
	if body == nil {
		return Error("expected body expression after 'for ... in'")
	}

	return &forAST{counter, start, end, step, body}
}

func parsePrimary() exprAST {
	switch token {
	case scanner.Ident:
		switch s.TokenText() {
		case "if":
			return parseIfExpr()
		case "for":
			return parseForExpr()
		default:
			return parseIdentifierExpr()
		}
	case scanner.Float, scanner.Int:
		return parseNumericExpr()
	case '(':
		return parseParenExpr()
	case scanner.EOF:
		return nil
	default:
		return Error(fmt.Sprint("unknown token when expecting expression: ", token, ":", s.TokenText()))
	}
}

var binaryOpPrecedence = map[rune]int{
	'<': 10,
	'+': 20,
	'-': 20,
	'*': 40,
	'/': 40,
}

func getTokenPrecedence() int {
	return binaryOpPrecedence[token]
}

func parseExpression() exprAST {
	lhs := parsePrimary()
	if lhs == nil {
		return nil
	}

	return parseBinaryOpRHS(1, lhs)
}

func parseBinaryOpRHS(exprPrec int, lhs exprAST) exprAST {
	for {
		tokenPrec := getTokenPrecedence()

		if tokenPrec < exprPrec {
			return lhs
		}

		binOp := token
		token = s.Scan()

		rhs := parsePrimary()
		if rhs == nil {
			return nil
		}

		nextPrec := getTokenPrecedence()
		if tokenPrec < nextPrec {
			rhs = parseBinaryOpRHS(tokenPrec+1, rhs)
			if rhs == nil {
				return nil
			}
		}

		lhs = &binAST{binOp, lhs, rhs}
	}
}

func parsePrototype() *protoAST {
	if token != scanner.Ident {
		return ErrorP("expected function name in prototype")
	}

	fnName := s.TokenText()
	token = s.Scan()

	if token != '(' {
		return ErrorP("expected '(' in prototype")
	}

	ArgNames := []string{}
	for token = s.Scan(); token == scanner.Ident || token == ','; token = s.Scan() {
		if token != ',' {
			ArgNames = append(ArgNames, s.TokenText())
		}
	}
	if token != ')' {
		return ErrorP("expected ')' in prototype")
	}

	token = s.Scan()
	return &protoAST{fnName, ArgNames}
}

func parseDefinition() *funcAST {
	token = s.Scan()
	proto := parsePrototype()
	if proto == nil {
		return nil
	}

	e := parseExpression()
	if e == nil {
		return nil
	}
	return &funcAST{proto, e}
}

func parseExtern() *protoAST {
	token = s.Scan()
	return parsePrototype()
}

func parseTopLevelExpr() *funcAST {
	e := parseExpression()
	if e == nil {
		return nil
	}

	// Make anon proto
	proto := &protoAST{"", []string{}}
	return &funcAST{proto, e}
}

// AST Nodes

type exprAST interface {
	codegen() llvm.Value
}

type numAST struct {
	val float64
}

type varAST struct {
	name string
}

type binAST struct {
	op    rune
	left  exprAST
	right exprAST
}

type callAST struct {
	callee string
	args   [](exprAST)
}

type protoAST struct {
	name string
	args []string
}

type funcAST struct {
	proto *protoAST
	body  exprAST
}

type ifAST struct {
	// psudeo-Hungarian notation as 'if' & 'else' are Go keywords
	ifE   exprAST
	thenE exprAST
	elseE exprAST
}

type forAST struct {
	counter string
	start   exprAST
	end     exprAST
	step    exprAST
	body    exprAST
}

// Helpers:
// error* prints error message and returns 0-values
func Error(str string) exprAST {
	fmt.Fprintf(os.Stderr, "Error at %v: %v\n\ttoken: %v\n\ttext:", s.Pos(), str, token, s.TokenText())
	return nil
}

func ErrorP(str string) *protoAST {
	Error(str)
	return nil
}

func ErrorF(str string) *funcAST {
	Error(str)
	return nil
}

func ErrorV(str string) llvm.Value {
	fmt.Fprintf(os.Stderr, "Error: %v\n", str)
	return llvm.Value{nil} // I don't think this is correct.
}
