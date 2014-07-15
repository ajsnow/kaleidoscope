package main

import (
	"fmt"
	"os"
	"strconv"
	"text/scanner"

	"github.com/ajsnow/llvm"
)

type tree struct {
	name               string
	tokens             <-chan token
	token              token
	root               *listNode
	binaryOpPrecedence map[rune]int
}

func NewTree(name string, tokens <-chan token) *tree {
	return &tree{
		name:   name,
		tokens: tokens,
		binaryOpPrecedence: map[rune]int{
			'=': 2,
			'<': 10,
			'+': 20,
			'-': 20,
			'*': 40,
			'/': 40,
		},
	}
}

func (t *tree) Parse() {
	for token := t.next(); token != tokEOF && token != tokError; token = t.next() {
		t.root.nodes = append(t.root.nodes, t.parseTopLevel())
	}
}

func (t *tree) next() token {
	t.token <- t.tokens
	return t.token
}

func (t *tree) parseTopLevel() node {
	switch t.token.kind {
	case tokDefine:
		return parseDefinition()
	case tokExtern:
		return parseExtern()
	default:
		return parseTopLevelExpr()
	}
}

func (t *tree) parseTopLevelExpr() node {
	pos = t.token.pos
	e := parseExpression()
	if e == nil {
		return nil
	}
	p := &fnPrototypeNode{nodeFnPrototype, pos} // everything else zero valued
	f := &functionNode{nodeFunction, pos, p, e}
	return f
}

func (t *tree) parseExtern() node {
	t.next()
}

// automagically move ../ast.go into here and convert the functions.
////////

// Parsing Functions
func parseNumericExpr() exprAST {
	val, err := strconv.ParseFloat(s.TokenText(), 64)
	token = s.Scan()
	if err != nil {
		return Error("invalid number")
	}
	result := &numAST{val}
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

func parseUnarty() exprAST {
	// If we're not an operator, parse as primary
	if token < -1 || token == '(' || token == ',' {
		return parsePrimary()
	}

	name := token
	token = s.Scan()
	operand := parseUnarty()
	if operand != nil {
		return &unaryAST{name, operand}
	}
	return nil
}

func parseVarExpr() exprAST {
	token = s.Scan()
	var v = varExprAST{
		vars: []struct {
			name string
			node exprAST
		}{},
		body: nil,
	}
	var val exprAST

	if token != scanner.Ident {
		return Error("expected identifier after var")
	}

	for {
		name := s.TokenText()
		token = s.Scan()

		// are we initialized?
		val = nil
		if token == '=' {
			token = s.Scan()
			val = parseExpression()
			if val == nil {
				return Error("initialization failed")
			}
		}
		v.vars = append(v.vars, struct {
			name string
			node exprAST
		}{name, val})

		if token != ',' {
			break
		}
		token = s.Scan()

		if token != scanner.Ident {
			return Error("expected identifier after var")
		}

	}

	// 'in'
	if s.TokenText() != "in" {
		return Error("expected 'in' after 'var'")
	}
	token = s.Scan()

	v.body = parseExpression()
	if v.body == nil {
		return Error("empty body in var expression")
	}
	return &v
}

func parsePrimary() exprAST {
	switch token {
	case scanner.Ident:
		switch s.TokenText() {
		case "if":
			return parseIfExpr()
		case "for":
			return parseForExpr()
		case "var":
			return parseVarExpr()
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
		oldToken := token
		oldText := s.TokenText()
		token = s.Scan()
		return Error(fmt.Sprint("unknown token when expecting expression: ", oldToken, ":", oldText))
	}
}

var binaryOpPrecedence = map[rune]int{
	'=': 2,
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
	lhs := parseUnarty()
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

		rhs := parseUnarty()
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

	precedence := 30
	const (
		idef = iota
		unary
		binary
	)
	kind := idef

	switch fnName {
	case "unary":
		fnName += s.TokenText() // unary^
		kind = unary
		token = s.Scan()
	case "binary":
		fnName += s.TokenText() // binary^
		kind = binary
		token = s.Scan()

		if token == scanner.Int {
			var err error
			precedence, err = strconv.Atoi(s.TokenText())
			if err != nil {
				return ErrorP("\ninvalid precedence")
			}
			token = s.Scan()
		}
	}

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
	if kind != idef && len(ArgNames) != kind {
		return ErrorP("invalid number of operands for operator")
	}
	return &protoAST{fnName, ArgNames, kind != idef, precedence}
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
	proto := &protoAST{"", []string{}, false, 0}
	return &funcAST{proto, e}
}

// Helpers:
// error* prints error message and returns 0-values
func Error(str string) exprAST {
	fmt.Fprintf(os.Stderr, "Error at %v: %v\n\ttoken: %v\n\ttext:%v\n", s.Pos(), str, token, s.TokenText())
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
