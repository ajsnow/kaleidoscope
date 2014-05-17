package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/ajsnow/llvm"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Lexer

// Lexer returns tokens, either [0-utf8.MaxRune] for
// unknown runes, or one of these for known things.
const (
	tEof = -1 - iota
	tDef
	tExtern
	tIdentifier
	tNumber
)

var (
	identifierName string  // tIdentifier's name if last call to getToken returned tIdentifier
	numberValue    float64 // the last tNumber's value
	scanner        *bufio.Scanner
	isWord         = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	isNumber       = regexp.MustCompile(`^[0-9.]+`)
)

func getToken() rune {
	var err error
	if !scanner.Scan() {
		return tEof
	}

	chars := scanner.Bytes()

	if isWord.Match(chars) {
		identifierName = scanner.Text()
		switch identifierName {
		case "def":
			return tDef
		case "extern":
			return tExtern
		default:
			return tIdentifier
		}
	}

	if isNumber.Match(chars) {
		numberValue, err = strconv.ParseFloat(scanner.Text(), 64)
		check(err)
		return tNumber
	}

	if scanner.Text() == "/*" {
		for scanner.Text() != "*/" {
			if !scanner.Scan() {
				return tEof
			}
		}
	}

	r, _ := utf8.DecodeRune(chars)
	return r
}

// Abstract Syntax Tree

type ExprAST interface {
	codegen() llvm.Value
}

type NumberExprAST struct {
	val float64
}

func (n *NumberExprAST) codegen() llvm.Value {
	return llvm.ConstFloat(llvm.DoubleType(), n.val)
}

type VariableExprAST struct {
	name string
}

func (n *VariableExprAST) codegen() llvm.Value {
	v := NamedValues[n.name]
	if v.IsNil() {
		return ErrorV("unknown variable name")
	}
	return v
}

type BinaryExprAST struct {
	op    rune
	left  ExprAST
	right ExprAST
}

func (n *BinaryExprAST) codegen() llvm.Value {
	l := n.left.codegen()
	r := n.right.codegen()
	if l.IsNil() || r.IsNil() {
		return ErrorV("operand was nil")
	}

	switch n.op {
	case '+':
		return Builder.CreateFAdd(l, r, "addtmp")
	case '-':
		return Builder.CreateFSub(l, r, "subtmp")
	case '*':
		return Builder.CreateFMul(l, r, "multmp")
	case '<':
		l = Builder.CreateFCmp(llvm.FloatUGT, l, r, "cmptmp")
		return Builder.CreateUIToFP(l, llvm.FloatType(), "booltmp")
	default:
		return ErrorV("invalid binary operator")
	}
}

type CallExprAST struct {
	callee string
	args   [](ExprAST)
}

func (n *CallExprAST) codegen() llvm.Value {
	callee := TheModule.NamedFunction(n.callee)
	if callee.IsNil() {
		return ErrorV("unknown function referenced")
	}

	if callee.ParamsCount() != len(n.args) {
		return ErrorV("incorrect number of arguments passed")
	}

	args := []llvm.Value{}
	for _, arg := range n.args {
		args = append(args, arg.codegen())
		if args[len(args)-1].IsNil() {
			return ErrorV("an argument was nil")
		}
	}

	return Builder.CreateCall(callee, args, "calltmp")
}

type PrototypeAST struct {
	name string
	args []string
}

func (n *PrototypeAST) codegen() llvm.Value {
	funcArgs := []llvm.Type{}
	for _ = range n.args {
		funcArgs = append(funcArgs, llvm.DoubleType())
	}
	funcType := llvm.FunctionType(llvm.DoubleType(), funcArgs, false)
	function := llvm.AddFunction(TheModule, n.name, funcType)

	if function.Name() != n.name {
		function.EraseFromParentAsFunction()
		function = TheModule.NamedFunction(n.name)
	}

	if function.BasicBlocksCount() != 0 {
		return ErrorV("redefinition of function")
	}

	if function.ParamsCount() != len(n.args) {
		return ErrorV("redefinition of function with different number of args")
	}

	for i, param := range function.Params() {
		param.SetName(n.args[i])
		NamedValues[n.args[i]] = param
	}

	return function
}

type FunctionAST struct {
	proto *PrototypeAST
	body  ExprAST
}

func (n *FunctionAST) codegen() llvm.Value {
	NamedValues = make(map[string]llvm.Value)

	theFunction := n.proto.codegen()
	if theFunction.IsNil() {
		return ErrorV("prototype")
	}

	block := llvm.AddBasicBlock(theFunction, "entry")
	Builder.SetInsertPointAtEnd(block)

	retVal := n.body.codegen()
	if retVal.IsNil() {
		theFunction.EraseFromParentAsFunction()
		return ErrorV("function body")
	}

	Builder.CreateRet(retVal)
	if llvm.VerifyFunction(theFunction, llvm.PrintMessageAction) != nil {
		theFunction.EraseFromParentAsFunction()
		return ErrorV("function verifiction failed")
	}

	//------------This code does have a bug, though. Since the PrototypeAST::Codegen can return a previously defined forward declaration, our code can actually delete a forward declaration. There are a number of ways to fix this bug, see what you can come up with! Here is a testcase

	FPM.RunFunc(theFunction)
	return theFunction
}

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

// Parser
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

// LLVM Stuff

var (
	TheModule            = llvm.NewModule("root")
	FPM                  = llvm.NewFunctionPassManagerForModule(TheModule)
	executionEngine, err = llvm.NewExecutionEngine(TheModule)
	Builder              = llvm.NewBuilder()
	NamedValues          = map[string]llvm.Value{}
)

// Driver

func handleDefinition() {
	if F := ParseDefinition(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read function definition:")
			LF.Dump()
		}
	} else {
		getNextToken()
	}
}

func handleExtern() {
	if F := ParseExtern(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read extern:")
			LF.Dump()
		}
	} else {
		getNextToken()
	}
}

func handleTopLevelExpression() {
	if F := ParseTopLevelExpr(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read top-level expression:")
			returnval := executionEngine.RunFunction(LF, []llvm.GenericValue{})
			fmt.Println("Evaluated to", returnval.Float(llvm.DoubleType()))
		}
	} else {
		getNextToken()
	}
}

func mainLoop() {
	for {
		switch curToken {
		case tEof:
			return
		case ';':
			getNextToken()
		case tDef:
			handleDefinition()
		case tExtern:
			handleExtern()
		default:
			handleTopLevelExpression()
		}
	}
}

func main() {
	FPM.AddInstructionCombiningPass()
	FPM.AddReassociatePass()
	FPM.AddGVNPass()
	FPM.AddCFGSimplificationPass()
	FPM.InitializeFunc()

	input := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		r, _, _ := input.ReadLine()
		scanner = bufio.NewScanner(bytes.NewBuffer(r))
		scanner.Split(bufio.ScanWords)
		getNextToken()
		mainLoop()
	}
}
