package main

import (
	"fmt"
	"os"
	"strconv"
	"text/scanner"

	"github.com/ajsnow/llvm"
)

// Parsing Functions
func ParseNumericExpr() ExprAST {
	val, err := strconv.ParseFloat(s.TokenText(), 64)
	if err != nil {
		return Error("invalid number: " + s.TokenText())
	}
	result := &NumberExprAST{val}
	token = s.Scan()
	return result
}

func ParseParenExpr() ExprAST {
	token = s.Scan()
	v := ParseExpression()
	if v == nil {
		return nil
	}

	if token != ')' {
		return Error("expected ')'")
	}
	token = s.Scan()
	return v
}

func ParseIdentifierExpr() ExprAST {
	name := s.TokenText()

	token = s.Scan()
	if token != '(' { // variable reference
		return &VariableExprAST{name}
	}

	// function call
	args := []ExprAST{}
	for token = s.Scan(); token != ')'; { //token = s.Scan() {
		switch token {
		case ',':
		default:
			arg := ParseExpression()
			if arg == nil {
				return nil
			}
			args = append(args, arg)
		}
	}
	token = s.Scan()
	return &CallExprAST{name, args}
}

func ParsePrimary() ExprAST {
	switch token {
	case scanner.Ident:
		return ParseIdentifierExpr()
	case scanner.Int:
		fallthrough
	case scanner.Float:
		return ParseNumericExpr()
	case '(':
		return ParseParenExpr()
	case scanner.EOF:
		return nil
	default:
		return Error(fmt.Sprint("unknown token when expecting expression: ", token, ":", s.TokenText()))
	}
}

var BinaryOpPrecedence = map[rune]int{
	'<': 10,
	'+': 20,
	'-': 20,
	'*': 40,
	'/': 40,
}

func getTokenPrecedence() int {
	return BinaryOpPrecedence[token]
}

func ParseExpression() ExprAST {
	lhs := ParsePrimary()
	if lhs == nil {
		return nil
	}

	return ParseBinaryOpRHS(1, lhs)
}

func ParseBinaryOpRHS(exprPrec int, lhs ExprAST) ExprAST {
	for { // wtf?
		tokenPrec := getTokenPrecedence()

		if tokenPrec < exprPrec {
			return lhs
		}

		binOp := token
		token = s.Scan()

		rhs := ParsePrimary()
		if rhs == nil {
			return nil
		}

		nextPrec := getTokenPrecedence()
		if tokenPrec < nextPrec {
			rhs = ParseBinaryOpRHS(tokenPrec+1, rhs)
			if rhs == nil {
				return nil
			}
		}

		lhs = &BinaryExprAST{binOp, lhs, rhs}
	}
}

func ParsePrototype() *PrototypeAST {
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
	return &PrototypeAST{fnName, ArgNames}
}

func ParseDefinition() *FunctionAST {
	token = s.Scan()
	proto := ParsePrototype()
	if proto == nil {
		return nil
	}

	e := ParseExpression()
	if e == nil {
		return nil
	}
	return &FunctionAST{proto, e}
}

func ParseExtern() *PrototypeAST {
	token = s.Scan()
	return ParsePrototype()
}

func ParseTopLevelExpr() *FunctionAST {
	e := ParseExpression()
	if e == nil {
		return nil
	}

	// Make anon proto
	proto := &PrototypeAST{"", []string{}}
	return &FunctionAST{proto, e}
}

// AST Nodes

type ExprAST interface {
	codegen() llvm.Value
}

type NumberExprAST struct {
	val float64
}

type VariableExprAST struct {
	name string
}

type BinaryExprAST struct {
	op    rune
	left  ExprAST
	right ExprAST
}

type CallExprAST struct {
	callee string
	args   [](ExprAST)
}

type PrototypeAST struct {
	name string
	args []string
}

type FunctionAST struct {
	proto *PrototypeAST
	body  ExprAST
}

// Helpers:
// error* prints error message and returns 0-values
func Error(s string) ExprAST {
	fmt.Fprintf(os.Stderr, "Error: %v\n", s)
	return nil
}

func ErrorP(s string) *PrototypeAST {
	Error(s)
	return nil
}

func ErrorF(s string) *FunctionAST {
	Error(s)
	return nil
}

func ErrorV(s string) llvm.Value {
	Error(s)
	return llvm.Value{nil}
}
