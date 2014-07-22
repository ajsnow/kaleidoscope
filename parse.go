package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ajsnow/llvm"
	"github.com/davecgh/go-spew/spew"
)

// A parser holds the internal state of the AST being constructed. Instead of
// composing top-level statements into banches under the AST root, they are
// send along a node chan that can be codegen'd and executed. This allows us
// to begin code generation and execution before we have finished parsing
// input (and/or allows us to use one parser during interactive mode instead
// of creating a new one for each line).
type parser struct {
	name               string         // name of current file whose tokens are being recieved; used in error reporting
	tokens             <-chan token   // channel of tokens from the lexer
	token              token          // current token, most reciently recieved
	topLevelNodes      chan node      // channel of parsed top-level statements
	binaryOpPrecedence map[string]int // maps binary operators to the precidence determining the order of operations
	printAst           bool           // prints top-level statements before sending
}

// Parse creates and runs a new parser, returning a channel of
// top-level AST sub-trees for further processing.
func Parse(tokens <-chan token, printAst bool) <-chan node {
	p := &parser{
		tokens:        tokens,
		topLevelNodes: make(chan node, 100),
		binaryOpPrecedence: map[string]int{
			"=": 2,
			"<": 10,
			"+": 20,
			"-": 20,
			"*": 40,
			"/": 40,
		},
		printAst: printAst,
	}
	go p.parse()
	return p.topLevelNodes
}

// parse is the parsing mainloop. It receives tokens and begins
// the recursive decent until a nil or top-level sub-tree is
// returned. Non-nils are sent to the topLevelNode channel;
// nils are discarded (they indicate either errors, semicolons
// or file boundaries). Once the tokens channel is empty & closed,
// it closes its own topLevelNodes channel.
func (p *parser) parse() {
	for p.next(); p.token.kind != tokError && p.token.kind != tokDONE; { //p.next() { // may want/need to switch this back once i introduce statement delineation
		topLevelNode := p.parseTopLevelStmt()
		if topLevelNode != nil {
			if p.printAst {
				spew.Dump(topLevelNode)
			}
			p.topLevelNodes <- topLevelNode
		}
	}

	if p.token.kind == tokError {
		spew.Dump(p.token)
	}
	close(p.topLevelNodes)
}

// next advances to the next useful token, discarding tokens
// that the parser doesn't need to handle like whitespace and
// comments.
// --
// TODO: check for closed channel instead of getting a default value'd tokDONE
func (p *parser) next() token {
	for p.token = <-p.tokens; p.token.kind == tokSpace ||
		p.token.kind == tokComment; p.token = <-p.tokens {
	}
	return p.token
}

// parseTopLevelStmt determines if the current token is the
// beginning of a function definition, external declaration or
// a top level expression. Top level (lone) semicolons are ignored;
// file transitions change the parser's file name variable.
// --
// TODO: roll error and tokDONE detection into this function
// TODO: don't return nil for non-error, non-done conditions
func (p *parser) parseTopLevelStmt() node {
	switch p.token.kind {
	case tokNewFile:
		p.name = p.token.val
		p.next()
		return nil
	case tokSemicolon:
		p.next()
		return nil
	case tokDefine:
		return p.parseDefinition()
	case tokExtern:
		return p.parseExtern()
	default:
		return p.parseTopLevelExpr()
	}
}

// parseDefinition parses top level function definitions.
func (p *parser) parseDefinition() node {
	pos := p.token.pos
	p.next()
	proto := p.parsePrototype()
	if p == nil {
		return nil
	}

	e := p.parseExpression()
	if e == nil {
		return nil
	}
	return &functionNode{nodeFunction, pos, proto, e}
}

func (p *parser) parseExtern() node {
	p.next()
	return p.parsePrototype()
}

// parseTopLevelExpr parses top level expressions by wrapping them
// into unnamed functions. The name "" signals that this statement
// is to be executed directly.
func (p *parser) parseTopLevelExpr() node {
	pos := p.token.pos
	e := p.parseExpression()
	if e == nil {
		return nil
	}
	proto := &fnPrototypeNode{nodeFnPrototype, pos, "", nil, false, 0} // fnName, ArgNames, kind != idef, precedence}
	f := &functionNode{nodeFunction, pos, proto, e}
	return f
}

// parsePrototype parses function prototypes. First it determines if
// the function is named. If the name is "unary" or "binary", then
// the prototype is for a user-defined operator. Binary ops may have
// an optional precedence specified to determine the order of
// operations.
// e.g. name(arg1, arg2, arg3)
// e.g. binary ∆ 50 (lhs rhs)
func (p *parser) parsePrototype() node {
	pos := p.token.pos
	if p.token.kind != tokIdentifier &&
		p.token.kind != tokBinary &&
		p.token.kind != tokUnary {
		return Error(p.token, "expected function name in prototype")
	}

	fnName := p.token.val
	p.next()

	precedence := 30
	const (
		idef = iota
		unary
		binary
	)
	kind := idef

	switch fnName {
	case "unary":
		fnName += p.token.val // unary^
		kind = unary
		p.next()
	case "binary":
		fnName += p.token.val // binary^
		op := p.token.val
		kind = binary
		p.next()

		if p.token.kind == tokNumber {
			var err error
			precedence, err = strconv.Atoi(p.token.val)
			if err != nil {
				return Error(p.token, "\ninvalid precedence")
			}
			p.next()
		}
		p.binaryOpPrecedence[op] = precedence // make sure to take this out of codegen later if we're going to keep it here.
	}

	if p.token.kind != tokLeftParen {
		return Error(p.token, "expected '(' in prototype")
	}

	ArgNames := []string{}
	for p.next(); p.token.kind == tokIdentifier || p.token.kind == tokComma; p.next() {
		if p.token.kind != tokComma {
			ArgNames = append(ArgNames, p.token.val)
		}
	}
	if p.token.kind != tokRightParen {
		return Error(p.token, "expected ')' in prototype")
	}

	p.next()
	if kind != idef && len(ArgNames) != kind {
		return Error(p.token, "invalid number of operands for operator")
	}
	return &fnPrototypeNode{nodeFnPrototype, pos, fnName, ArgNames, kind != idef, precedence}
}

// parseExpression parses expressions. First, it tries to parse
// the current token as the beginning of a unary expression. If
// the result is non-null, it will parse the rest as the right-
// hand side of a binary expression.
// e.g. !!5 + sin(2 * 4) - 2 -> {!!5} {+ sin(2 * 4) - 2}
func (p *parser) parseExpression() node {
	lhs := p.parseUnarty()
	if lhs == nil {
		return nil
	}

	return p.parseBinaryOpRHS(1, lhs) // TODO: check on this value wrt our : = and 0 val for not found instead of tut's -1
} /// also this way of hacking on left to right preference on top of operator precedence can fail if we have more expressions than the difference in the op pref, right?

// parseUnarty parses unary expressions. If the current token is
// not a unary operator, parse it as a primary expression; otherwise,
// return a unaryNode, parsing the operand of the unary operator as
// another unary expression (so as to allow chaining of unary ops).
func (p *parser) parseUnarty() node {
	pos := p.token.pos
	// If we're not an operator, parse as primary {this is correcp.}
	if p.token.kind < tokUserUnaryOp {
		return p.parsePrimary()
	}

	name := p.token.val
	p.next()
	operand := p.parseUnarty()
	if operand != nil {
		return &unaryNode{nodeUnary, pos, name, operand}
	}
	return nil
}

// parseBinaryOpRHS parses the operator and right-hand side of a
// binary operator expression. <TODO: describe algo after it's been cleaned up a bit>
func (p *parser) parseBinaryOpRHS(exprPrec int, lhs node) node {
	pos := p.token.pos
	for {
		if p.token.kind < tokUserUnaryOp {
			return lhs
		}
		tokenPrec := p.getTokenPrecedence(p.token.val)
		if tokenPrec < exprPrec {
			return lhs
		}
		binOp := p.token.val
		p.next()

		rhs := p.parseUnarty()
		if rhs == nil {
			return nil
		}

		nextPrec := p.getTokenPrecedence(p.token.val)
		if tokenPrec < nextPrec {
			rhs = p.parseBinaryOpRHS(tokenPrec+1, rhs)
			if rhs == nil {
				return nil
			}
		}

		lhs = &binaryNode{nodeBinary, pos, binOp, lhs, rhs}
	}
}

// getTokenPrecedence returns a binary operator's precedence
func (p *parser) getTokenPrecedence(token string) int {
	return p.binaryOpPrecedence[token]
}

// parsePrimary parses primary expressions. The parser arrives
// here when operator expressions are gathering their operands.
// (Or when there are no operators at the top level of a given
// sub-expression.)
func (p *parser) parsePrimary() node {
	switch p.token.kind {
	case tokIdentifier:
		return p.parseIdentifierExpr()
	case tokIf:
		return p.parseIfExpr()
	case tokFor:
		return p.parseForExpr()
	case tokVariable:
		return p.parseVarExpr()
	case tokNumber:
		return p.parseNumericExpr()
	case tokLeftParen:
		return p.parseParenExpr()
	case tokDONE:
		return nil // this token should not be skipped
	default:
		oldToken := p.token
		p.next()
		return Error(oldToken, "unknown token encountered when expecting expression")
	}
}

// parseIdentifierExpr parses user defined identifiers (i.e. variable
// and function names). If it is a function name, parse any arguments
// it may take and emit a function call node. Otherwise, emit the variable.
func (p *parser) parseIdentifierExpr() node {
	pos := p.token.pos
	name := p.token.val
	p.next()
	// are we a variable? else function call
	if p.token.kind != tokLeftParen {
		return &variableNode{nodeVariable, pos, name}
	}
	args := []node{}
	for p.next(); p.token.kind != tokRightParen; {
		switch p.token.kind {
		case tokComma:
			p.next()
		default:
			arg := p.parseExpression()
			if arg == nil {
				return nil
			}
			args = append(args, arg)
		}
	}
	p.next()
	return &fnCallNode{nodeFnCall, pos, name, args}
}

// parseIfExpr, as the name suggest, parses each part of an if expression
// and emits the result.
func (p *parser) parseIfExpr() node {
	pos := p.token.pos
	// if
	p.next()
	ifE := p.parseExpression()
	if ifE == nil {
		return Error(p.token, "expected condition after 'if'")
	}

	if p.token.kind != tokThen {
		return Error(p.token, "expected 'then' after if condition")
	}
	p.next()
	thenE := p.parseExpression()
	if thenE == nil {
		return Error(p.token, "expected expression after 'then'")
	}

	if p.token.kind != tokElse {
		return Error(p.token, "expected 'else' after then expr")
	}
	p.next()
	elseE := p.parseExpression()
	if elseE == nil {
		return Error(p.token, "expected expression after 'else'")
	}

	return &ifNode{nodeIf, pos, ifE, thenE, elseE}
}

// parseIfExpr parses each part of a for expression. The increment
// step is optional and defaults to += 1 if unspecified.
func (p *parser) parseForExpr() node {
	pos := p.token.pos
	p.next()
	if p.token.kind != tokIdentifier {
		return Error(p.token, "expected identifier after 'for'")
	}
	counter := p.token.val

	p.next()
	if p.token.kind != tokEqual {
		return Error(p.token, "expected '=' after 'for "+counter+"'")
	}

	p.next()
	start := p.parseExpression()
	if start == nil {
		return Error(p.token, "expected expression after 'for "+counter+" ='")
	}
	if p.token.kind != tokComma {
		return Error(p.token, "expected ',' after 'for' start expression")
	}

	p.next()
	end := p.parseExpression()
	if end == nil {
		return Error(p.token, "expected end expression after 'for' start expression")
	}

	// optional step
	var step node
	if p.token.kind == tokComma {
		p.next()
		if step = p.parseExpression(); step == nil {
			return Error(p.token, "invalid step expression after 'for'")
		}
	}

	if p.token.kind != tokIn {
		return Error(p.token, "expected 'in' after 'for' sub-expression")
	}

	p.next()
	body := p.parseExpression()
	if body == nil {
		return Error(p.token, "expected body expression after 'for ... in'")
	}

	return &forNode{nodeFor, pos, counter, start, end, step, body}
}

// parseVarExpr parses an expression declaring (and using) mutable
// variables.
func (p *parser) parseVarExpr() node {
	pos := p.token.pos
	p.next()
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
	if p.token.kind != tokIdentifier {
		return Error(p.token, "expected identifier after var")
	}
	for {
		name := p.token.val
		p.next()

		// are we initialized?
		val = nil
		if p.token.kind == tokEqual {
			p.next()
			val = p.parseExpression()
			if val == nil {
				return Error(p.token, "initialization failed")
			}
		}
		v.vars = append(v.vars, struct {
			name string
			node node
		}{name, val})

		if p.token.kind != tokComma {
			break
		}
		p.next()

		if p.token.kind != tokIdentifier {
			return Error(p.token, "expected identifier after var")
		}
	}

	// 'in'
	if p.token.kind != tokIn {
		return Error(p.token, "expected 'in' after 'var'")
	}
	p.next()

	v.body = p.parseExpression()
	if v.body == nil {
		return Error(p.token, "empty body in var expression")
	}
	return &v
}

// parseParenExpr parses expressions offset by parens.
func (p *parser) parseParenExpr() node {
	p.next()
	v := p.parseExpression()
	if v == nil {
		return nil
	}
	if p.token.kind != tokRightParen {
		return Error(p.token, "expected ')'")
	}
	p.next()
	return v
}

// parseNumericExpr parses number literals.
func (p *parser) parseNumericExpr() node {
	pos := p.token.pos
	val, err := strconv.ParseFloat(p.token.val, 64)
	p.next()
	if err != nil {
		return Error(p.token, "invalid number")
	}
	return &numberNode{nodeNumber, pos, val}
}

// Helpers:
// Error prints error message and returns a nil node
func Error(t token, str string) node {
	fmt.Fprintf(os.Stderr, "Error at %v: %v\n\tkind:  %v\n\tvalue: %v\n", t.pos, str, t.kind, t.val)
	// log.Fatalf("Error at %v: %v\n\tkind:  %v\n\tvalue: %v\n", p.pos, str, p.kind, p.val)
	return nil
}

// ErrorV prints the error message and returns a nil llvm.Value
func ErrorV(str string) llvm.Value {
	fmt.Fprintf(os.Stderr, "Error: %v\n", str)
	return llvm.Value{nil} // TODO: this is wrong; fix it.
}
