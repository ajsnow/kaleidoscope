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

func parsePrimary() exprAST {
	switch token {
	case scanner.Ident:
		return parseIdentifierExpr()
	case scanner.Int:
		fallthrough
	case scanner.Float:
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

// Helpers:
// error* prints error message and returns 0-values
func Error(s string) exprAST {
	fmt.Fprintf(os.Stderr, "Error: %v\n", s)
	return nil
}

func ErrorP(s string) *protoAST {
	Error(s)
	return nil
}

func ErrorF(s string) *funcAST {
	Error(s)
	return nil
}

func ErrorV(s string) llvm.Value {
	Error(s)
	return llvm.Value{nil}
}
