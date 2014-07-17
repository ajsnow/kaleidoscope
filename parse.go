package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ajsnow/llvm"
)

type tree struct {
	name               string
	tokens             <-chan token
	token              token
	root               *listNode
	binaryOpPrecedence map[string]int
}

func NewTree(name string, tokens <-chan token) *tree {
	return &tree{
		name:   name,
		tokens: tokens,
		root:   &listNode{nodeList, 0, []node{}},
		binaryOpPrecedence: map[string]int{
			"=": 2,
			"<": 10,
			"+": 20,
			"-": 20,
			"*": 40,
			"/": 40,
		},
	}
}

func (t *tree) Parse() bool {
	for t.next(); t.token.kind != tokEOF && t.token.kind != tokError; { //t.next() { // may want/need to switch this back once i introduce statement delineation
		node := t.parseTopLevelStmt()
		if node != nil {
			t.root.nodes = append(t.root.nodes, node)
		} else {
			fmt.Println("Nil top level node near:", t.token.pos)
		}
	}
	return true
}

func (t *tree) next() token {
	t.token = <-t.tokens
	for t.token.kind == tokSpace || t.token.kind == tokComment || t.token.kind == tokSemicolon {
		t.token = <-t.tokens
	}
	return t.token
}

func (t *tree) parseTopLevelStmt() node {
	switch t.token.kind {
	case tokDefine:
		return t.parseDefinition()
	case tokExtern:
		return t.parseExtern()
	default:
		return t.parseTopLevelExpr()
	}
}

func (t *tree) parseDefinition() node {
	pos := t.token.pos
	t.next()
	p := t.parsePrototype()
	if p == nil {
		return nil
	}

	e := t.parseExpression()
	if e == nil {
		return nil
	}
	return &functionNode{nodeFunction, pos, p, e}
}

func (t *tree) parseExtern() node {
	t.next()
	return t.parsePrototype()
}

func (t *tree) parseTopLevelExpr() node {
	pos := t.token.pos
	e := t.parseExpression()
	if e == nil {
		return nil
	}
	p := &fnPrototypeNode{nodeFnPrototype, pos, "", nil, false, 0} // fnName, ArgNames, kind != idef, precedence}
	f := &functionNode{nodeFunction, pos, p, e}
	return f
}

func (t *tree) parsePrototype() node {
	pos := t.token.pos
	if t.token.kind != tokIdentifier &&
		t.token.kind != tokBinary &&
		t.token.kind != tokUnary {
		return Error(t.token, "expected function name in prototype")
	}

	fnName := t.token.val
	t.next()

	precedence := 30
	const (
		idef = iota
		unary
		binary
	)
	kind := idef

	switch fnName {
	case "unary":
		fnName += t.token.val // unary^
		kind = unary
		t.next()
	case "binary":
		fnName += t.token.val // binary^
		op := t.token.val
		kind = binary
		t.next()

		if t.token.kind == tokNumber {
			var err error
			precedence, err = strconv.Atoi(t.token.val)
			if err != nil {
				return Error(t.token, "\ninvalid precedence")
			}
			t.next()
		}
		t.binaryOpPrecedence[op] = precedence // make sure to take this out of codegen later if we're going to keep it here.
	}

	if t.token.kind != tokLeftParen {
		return Error(t.token, "expected '(' in prototype")
	}

	ArgNames := []string{}
	for t.next(); t.token.kind == tokIdentifier || t.token.kind == tokComma; t.next() {
		if t.token.kind != tokComma {
			ArgNames = append(ArgNames, t.token.val)
		}
	}
	if t.token.kind != tokRightParen {
		return Error(t.token, "expected ')' in prototype")
	}

	t.next()
	if kind != idef && len(ArgNames) != kind {
		return Error(t.token, "invalid number of operands for operator")
	}
	return &fnPrototypeNode{nodeFnPrototype, pos, fnName, ArgNames, kind != idef, precedence}
}

func (t *tree) parseExpression() node {
	// pos := t.token.pos
	lhs := t.parseUnarty()
	if lhs == nil {
		return nil
	}

	return t.parseBinaryOpRHS(1, lhs) //  !!! check on this value wrt our : = and 0 val for not found instead of tut's -1
} /// also this way of hacking on left to right preference on top of opperator precidence can fail if we have more expressions than the difference in the op pref, right?

func (t *tree) parseUnarty() node {
	pos := t.token.pos
	// If we're not an operator, parse as primary {this is correct.}
	if t.token.kind < tokUserUnaryOp {
		return t.parsePrimary()
	}

	name := t.token.val
	t.next()
	operand := t.parseUnarty()
	if operand != nil {
		return &unaryNode{nodeUnary, pos, name, operand}
	}
	return nil
}

func (t *tree) parseBinaryOpRHS(exprPrec int, lhs node) node {
	pos := t.token.pos
	for {
		if t.token.kind < tokUserUnaryOp {
			return lhs
		}
		tokenPrec := t.getTokenPrecedence(t.token.val)
		if tokenPrec < exprPrec {
			return lhs
		}
		binOp := t.token.val
		t.next()

		rhs := t.parseUnarty()
		if rhs == nil {
			return nil
		}

		nextPrec := t.getTokenPrecedence(t.token.val)
		if tokenPrec < nextPrec {
			rhs = t.parseBinaryOpRHS(tokenPrec+1, rhs)
			if rhs == nil {
				return nil
			}
		}

		lhs = &binaryNode{nodeBinary, pos, binOp, lhs, rhs}
	}
}

func (t *tree) getTokenPrecedence(token string) int {
	return t.binaryOpPrecedence[token]
}

func (t *tree) parsePrimary() node {
	// pos := t.token.pos
	switch t.token.kind {
	case tokIdentifier:
		return t.parseIdentifierExpr()
	case tokIf:
		return t.parseIfExpr()
	case tokFor:
		return t.parseForExpr()
	case tokVariable:
		return t.parseVarExpr()
	case tokNumber:
		return t.parseNumericExpr()
	case tokLeftParen:
		return t.parseParenExpr()
	case tokEOF:
		return nil
	default:
		oldToken := t.token
		t.next()
		return Error(t.token, fmt.Sprint("unknown token when expecting expression: ", oldToken))
	}
}

func (t *tree) parseIdentifierExpr() node {
	pos := t.token.pos
	name := t.token.val
	t.next()
	// are we a variable? else function call
	if t.token.kind != tokLeftParen {
		return &variableNode{nodeVariable, pos, name}
	}
	args := []node{}
	for t.next(); t.token.kind != tokRightParen; {
		switch t.token.kind {
		case tokComma:
			t.next()
		default:
			arg := t.parseExpression()
			if arg == nil {
				return nil
			}
			args = append(args, arg)
		}
	}
	t.next()
	return &fnCallNode{nodeFnCall, pos, name, args}
}

func (t *tree) parseIfExpr() node {
	pos := t.token.pos
	// if
	t.next()
	ifE := t.parseExpression()
	if ifE == nil {
		return Error(t.token, "expected condition after 'if'")
	}

	if t.token.kind != tokThen {
		return Error(t.token, "expected 'then' after if condition")
	}
	t.next()
	thenE := t.parseExpression()
	if thenE == nil {
		return Error(t.token, "expected expression after 'then'")
	}

	if t.token.kind != tokElse {
		return Error(t.token, "expected 'else' after then expr")
	}
	t.next()
	elseE := t.parseExpression()
	if elseE == nil {
		return Error(t.token, "expected expression after 'else'")
	}

	return &ifNode{nodeIf, pos, ifE, thenE, elseE}
}

func (t *tree) parseForExpr() node {
	pos := t.token.pos
	t.next()
	if t.token.kind != tokIdentifier {
		return Error(t.token, "expected identifier after 'for'")
	}
	counter := t.token.val

	t.next()
	if t.token.kind != tokEqual {
		return Error(t.token, "expected '=' after 'for "+counter+"'")
	}

	t.next()
	start := t.parseExpression()
	if start == nil {
		return Error(t.token, "expected expression after 'for "+counter+" ='")
	}
	if t.token.kind != tokComma {
		return Error(t.token, "expected ',' after 'for' start expression")
	}

	t.next()
	end := t.parseExpression()
	if end == nil {
		return Error(t.token, "expected end expression after 'for' start expression")
	}

	// optional step
	var step node
	if t.token.kind == tokComma {
		t.next()
		if step = t.parseExpression(); step == nil {
			return Error(t.token, "invalid step expression after 'for'")
		}
	}

	if t.token.kind != tokIn {
		return Error(t.token, "expected 'in' after 'for' sub-expression")
	}

	t.next()
	body := t.parseExpression()
	if body == nil {
		return Error(t.token, "expected body expression after 'for ... in'")
	}

	return &forNode{nodeFor, pos, counter, start, end, step, body}
}

func (t *tree) parseVarExpr() node {
	pos := t.token.pos
	t.next()
	var v = variableExprNode{
		nodeType: nodeVariableExpr,
		Pos:      pos,
		vars: []struct {
			name string
			node node
		}{},
		body: nil,
	}
	var val node

	// this forloop can be simplified greatly.
	if t.token.kind != tokIdentifier {
		return Error(t.token, "expected identifier after var")
	}
	for {
		name := t.token.val
		t.next()

		// are we initialized?
		val = nil
		if t.token.kind == tokEqual {
			t.next()
			val = t.parseExpression()
			if val == nil {
				return Error(t.token, "initialization failed")
			}
		}
		v.vars = append(v.vars, struct {
			name string
			node node
		}{name, val})

		if t.token.kind != tokComma {
			break
		}
		t.next()

		if t.token.kind != tokIdentifier {
			return Error(t.token, "expected identifier after var")
		}
	}

	// 'in'
	if t.token.kind != tokIn {
		return Error(t.token, "expected 'in' after 'var'")
	}
	t.next()

	v.body = t.parseExpression()
	if v.body == nil {
		return Error(t.token, "empty body in var expression")
	}
	return &v
}

func (t *tree) parseParenExpr() node {
	// pos := t.token.pos
	t.next()
	v := t.parseExpression()
	if v == nil {
		return nil
	}
	if t.token.kind != tokRightParen {
		return Error(t.token, "expected ')'")
	}
	t.next()
	return v
}

func (t *tree) parseNumericExpr() node {
	pos := t.token.pos
	val, err := strconv.ParseFloat(t.token.val, 64)
	t.next()
	if err != nil {
		return Error(t.token, "invalid number")
	}
	return &numberNode{nodeNumber, pos, val}
}

// Helpers:
// error* prints error message and returns 0-values
func Error(t token, str string) node {
	fmt.Fprintf(os.Stderr, "Error at %v: %v\n\tkind:  %v\n\tvalue: %v\n", t.pos, str, t.kind, t.val)
	// log.Fatalf("Error at %v: %v\n\tkind:  %v\n\tvalue: %v\n", t.pos, str, t.kind, t.val)
	return nil
}

func ErrorV(str string) llvm.Value {
	fmt.Fprintf(os.Stderr, "Error: %v\n", str)
	return llvm.Value{nil} // I don't think this is correct.
}
