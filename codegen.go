package main

import (
	"unicode/utf8"

	"github.com/ajsnow/llvm"
)

var (
	TheModule                        = llvm.NewModule("root")
	FPM                              = llvm.NewFunctionPassManagerForModule(TheModule)
	unhandledError1                  = llvm.InitializeNativeTarget()
	executionEngine, unhandledError2 = llvm.NewJITCompiler(TheModule, 0)
	Builder                          = llvm.NewBuilder()
	NamedValues                      = map[string]llvm.Value{}
)

func optimize() {
	FPM.Add(executionEngine.TargetData())
	FPM.AddPromoteMemoryToRegisterPass()
	FPM.AddInstructionCombiningPass()
	FPM.AddReassociatePass()
	FPM.AddGVNPass()
	FPM.AddCFGSimplificationPass()
	FPM.InitializeFunc()
}

func createEntryBlockAlloca(f llvm.Value, name string) llvm.Value {
	var TmpB = llvm.NewBuilder()
	TmpB.SetInsertPoint(f.EntryBasicBlock(), f.EntryBasicBlock().FirstInstruction())
	return TmpB.CreateAlloca(llvm.DoubleType(), name)
}

func (n *protoAST) createArgAlloca(f llvm.Value) {
	args := f.Params()
	for i := range args {
		alloca := createEntryBlockAlloca(f, n.args[i])
		Builder.CreateStore(args[i], alloca)
		NamedValues[n.args[i]] = alloca
	}
}

func (n *numAST) codegen() llvm.Value {
	return llvm.ConstFloat(llvm.DoubleType(), n.val)
}

func (n *varAST) codegen() llvm.Value {
	v := NamedValues[n.name]
	if v.IsNil() {
		return ErrorV("unknown variable name")
	}
	return Builder.CreateLoad(v, n.name)
}

func (n *ifAST) codegen() llvm.Value {
	// psudeo-Hungarian notation as 'if' & 'else' are Go keywords
	// also aligns with llvm pkg's func arg names
	ifv := n.ifE.codegen()
	if ifv.IsNil() {
		return ErrorV("code generation failed for if expression")
	}
	ifv = Builder.CreateFCmp(llvm.FloatONE, ifv, llvm.ConstFloat(llvm.DoubleType(), 0), "ifcond")

	parentFunc := Builder.GetInsertBlock().Parent()
	thenBlk := llvm.AddBasicBlock(parentFunc, "then")
	elseBlk := llvm.AddBasicBlock(parentFunc, "else")
	mergeBlk := llvm.AddBasicBlock(parentFunc, "merge")
	Builder.CreateCondBr(ifv, thenBlk, elseBlk)

	// generate 'then' block
	Builder.SetInsertPointAtEnd(thenBlk)
	thenv := n.thenE.codegen()
	if thenv.IsNil() {
		return ErrorV("code generation failed for then expression")
	}
	Builder.CreateBr(mergeBlk)
	// Codegen of 'Then' can change the current block, update ThenBB for the PHI.
	thenBlk = Builder.GetInsertBlock()

	// generate 'else' block
	// C++ unknown eq: TheFunction->getBasicBlockList().push_back(ElseBB);
	Builder.SetInsertPointAtEnd(elseBlk)
	elsev := n.elseE.codegen()
	if elsev.IsNil() {
		return ErrorV("code generation failed for else expression")
	}
	Builder.CreateBr(mergeBlk)
	elseBlk = Builder.GetInsertBlock()

	Builder.SetInsertPointAtEnd(mergeBlk)
	PhiNode := Builder.CreatePHI(llvm.DoubleType(), "iftmp")
	PhiNode.AddIncoming([]llvm.Value{thenv}, []llvm.BasicBlock{thenBlk})
	PhiNode.AddIncoming([]llvm.Value{elsev}, []llvm.BasicBlock{elseBlk})
	return PhiNode
}

func (n *forAST) codegen() llvm.Value {
	startVal := n.start.codegen()
	if startVal.IsNil() {
		return ErrorV("code generation failed for start expression")
	}

	parentFunc := Builder.GetInsertBlock().Parent()
	alloca := createEntryBlockAlloca(parentFunc, n.counter)
	Builder.CreateStore(startVal, alloca)
	loopBlk := llvm.AddBasicBlock(parentFunc, "loop")

	Builder.CreateBr(loopBlk)

	Builder.SetInsertPointAtEnd(loopBlk)

	// save higher levels' variables if we have the same name
	oldVal := NamedValues[n.counter]
	NamedValues[n.counter] = alloca

	if n.body.codegen().IsNil() { // wtf?
		return ErrorV("code generation failed for body expression")
	}

	var stepVal llvm.Value
	if n.step != nil {
		stepVal = n.step.codegen()
		if stepVal.IsNil() {
			return llvm.ConstNull(llvm.DoubleType())
		}
	} else {
		stepVal = llvm.ConstFloat(llvm.DoubleType(), 1)
	}

	curVar := Builder.CreateLoad(alloca, n.counter)
	nextVar := Builder.CreateFAdd(curVar, stepVal, "nextvar")
	Builder.CreateStore(nextVar, alloca)

	endVal := n.end.codegen()
	if endVal.IsNil() {
		return endVal
	}

	endVal = Builder.CreateFCmp(llvm.FloatONE, endVal, llvm.ConstFloat(llvm.DoubleType(), 0), "loopcond")
	afterBlk := llvm.AddBasicBlock(parentFunc, "afterloop")

	Builder.CreateCondBr(endVal, loopBlk, afterBlk)

	Builder.SetInsertPointAtEnd(afterBlk)

	if !oldVal.IsNil() {
		NamedValues[n.counter] = oldVal
	} else {
		delete(NamedValues, n.counter)
	}

	return llvm.ConstFloat(llvm.DoubleType(), 0)
}

func (n *unaryAST) codegen() llvm.Value {
	operandValue := n.operand.codegen()
	if operandValue.IsNil() {
		return ErrorV("nil operand")
	}

	f := TheModule.NamedFunction("unary" + string(n.name))
	if f.IsNil() {
		return ErrorV("unknown unary operator")
	}
	return Builder.CreateCall(f, []llvm.Value{operandValue}, "unop")
}

func (n *callAST) codegen() llvm.Value {
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

func (n *binAST) codegen() llvm.Value {
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
	case '/':
		return Builder.CreateFDiv(l, r, "divtmp")
	case '<':
		l = Builder.CreateFCmp(llvm.FloatOLT, l, r, "cmptmp")
		return Builder.CreateUIToFP(l, llvm.DoubleType(), "booltmp")
	default:
		function := TheModule.NamedFunction("binary" + string(n.op))
		if function.IsNil() {
			return ErrorV("invalid binary operator")
		}
		return Builder.CreateCall(function, []llvm.Value{l, r}, "binop")
	}
}

func (n *protoAST) codegen() llvm.Value {
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

func (n *funcAST) codegen() llvm.Value {
	NamedValues = make(map[string]llvm.Value)

	theFunction := n.proto.codegen()
	if theFunction.IsNil() {
		return ErrorV("prototype")
	}

	if n.proto.isOperator && len(n.proto.args) == 2 {
		opChar, _ := utf8.DecodeLastRuneInString(n.proto.name)
		binaryOpPrecedence[opChar] = n.proto.precedence
	}

	block := llvm.AddBasicBlock(theFunction, "entry")
	Builder.SetInsertPointAtEnd(block)

	n.proto.createArgAlloca(theFunction)

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

	FPM.RunFunc(theFunction)
	return theFunction
}
