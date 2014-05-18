package main

import (
	"fmt"
	"os"

	"github.com/ajsnow/llvm"
)

// Parsing Functions
func ParseNumericExpr() ExprAST {
	result := &NumberExprAST{numberValue}
	getNextToken()
	return result
}

func ParseParenExpr() ExprAST {
	getNextToken()
	v := ParseExpression()
	if v == nil {
		return nil
	}

	if curToken != ')' {
		return Error("expected ')'")
	}
	getNextToken()
	return v
}

func ParseIdentifierExpr() ExprAST {
	name := identifierName

	getNextToken()
	if curToken != '(' { // variable reference
		return &VariableExprAST{name}
	}

	// function call
	getNextToken()
	args := []ExprAST{}
	if curToken != ')' {
		for {
			arg := ParseExpression()
			if arg == nil {
				return nil
			}
			args = append(args, arg)

			if curToken == ')' {
				break
			}

			if curToken != ',' {
				return Error("expected ')' or ',' in argument list")
			}
			getNextToken()
		}
	}
	getNextToken()
	return &CallExprAST{name, args}
}

func ParsePrimary() ExprAST {
	switch curToken {
	case tIdentifier:
		return ParseIdentifierExpr()
	case tNumber:
		return ParseNumericExpr()
	case '(':
		return ParseParenExpr()
	default:
		return Error(fmt.Sprintln("unknown token when expecting expression:\n\t", curToken, ":", identifierName))
	}
}

var BinaryOpPrecedence = map[rune]int{
	'<': 10,
	'+': 20,
	'-': 20,
	'*': 40,
}

func getTokenPrecedence() int {
	return BinaryOpPrecedence[curToken]
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

		binOp := curToken
		getNextToken()

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
	if curToken != tIdentifier {
		return ErrorP("expected function name in prototype")
	}

	fnName := identifierName
	getNextToken()

	if curToken != '(' {
		return ErrorP("expected '(' in prototype")
	}

	ArgNames := []string{}
	for getNextToken() == tIdentifier {
		ArgNames = append(ArgNames, identifierName)
	}
	if curToken != ')' {
		return ErrorP("expected ')' in prototype")
	}

	getNextToken()
	return &PrototypeAST{fnName, ArgNames}
}

func ParseDefinition() *FunctionAST {
	getNextToken()
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
	getNextToken()
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
// getNextToken impliments simple 1 token buffer
// error* prints error message and returns 0-values
var curToken rune

func getNextToken() rune {
	curToken = getToken()
	return curToken
}

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
