package main

import "github.com/ajsnow/llvm"

var (
	TheModule            = llvm.NewModule("root")
	FPM                  = llvm.NewFunctionPassManagerForModule(TheModule)
	executionEngine, err = llvm.NewExecutionEngine(TheModule)
	Builder              = llvm.NewBuilder()
	NamedValues          = map[string]llvm.Value{}
)

func init() {
	FPM.AddInstructionCombiningPass()
	FPM.AddReassociatePass()
	FPM.AddGVNPass()
	FPM.AddCFGSimplificationPass()
	FPM.InitializeFunc()
}

func (n *numAST) codegen() llvm.Value {
	return llvm.ConstFloat(llvm.DoubleType(), n.val)
}

func (n *varAST) codegen() llvm.Value {
	v := NamedValues[n.name]
	if v.IsNil() {
		return ErrorV("unknown variable name")
	}
	return v
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
		return ErrorV("code generation failid for then expression")
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
		return Builder.CreateUIToFP(l, llvm.FloatType(), "booltmp")
	default:
		return ErrorV("invalid binary operator")
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

	FPM.RunFunc(theFunction)
	return theFunction
}
