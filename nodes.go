package main

import "github.com/ajsnow/llvm"

// Node Nodes

type node interface {
	Kind() nodeType
	// String() string
	Position() Pos
	codegen() llvm.Value
}

type nodeType int

// Pos defines a byte offset from the beginning of the input text.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// In text/template/parse/node.go Rob adds an unexported() method to Pos
// I do know why he did that rather than make Pos -> pos

// Type returns itself, embedding into Nodes
func (t nodeType) Kind() nodeType {
	return t
}

const (
	// literals
	nodeNumber nodeType = iota

	// expressions
	nodeIf
	nodeFor
	nodeUnary
	nodeBinary
	nodeFnCall
	nodeVariable
	nodeVariableExpr

	// non-expression statements
	nodeFnPrototype
	nodeFunction

	// other
	nodeList
)

type numberNode struct {
	nodeType
	Pos

	val float64
}

// func NewNumberNode(t token, val float64) *numberNode {
// 	return &numberNode{
// 		nodeType: nodeNumber,
// 		Pos:      t.pos,
// 		val:      val,
// 	}
// }

type ifNode struct {
	nodeType
	Pos

	// psudeo-Hungarian notation as 'if' & 'else' are Go keywords
	ifN   node
	thenN node
	elseN node
}

// func NewIfNode(t token, ifN, thenN, elseN node) *ifNode {
// 	return &ifNode{
// 		nodeType: nodeIf,
// 		Pos:      t.pos,
// 		ifN:      ifN,
// 		thenN:    thenN,
// 		elseN:    elseN,
// 	}
// }

type forNode struct {
	nodeType
	Pos

	counter string
	start   node
	test    node
	step    node
	body    node
}

// func NewForNode(t token, counter string, start, test, step, body node) *forNode {
// 	return &forNode{nodeFor, t.pos, counter, start, test, step, body}
// }

type unaryNode struct {
	nodeType
	Pos

	name    string
	operand node
}

type binaryNode struct {
	nodeType
	Pos

	op    string
	left  node
	right node
}

type fnCallNode struct {
	nodeType
	Pos

	callee string
	args   [](node)
}

type variableNode struct {
	nodeType
	Pos

	name string
}

type variableExprNode struct {
	nodeType
	Pos

	vars []struct {
		name string
		node node
	}
	body node
}

type fnPrototypeNode struct {
	nodeType
	Pos

	name       string
	args       []string
	isOperator bool
	precedence int
}

type functionNode struct {
	nodeType
	Pos

	proto node
	body  node
}

type listNode struct {
	nodeType
	Pos

	nodes []node
}
