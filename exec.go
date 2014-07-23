package main

import (
	"fmt"
	"os"

	"github.com/ajsnow/llvm"
)

// Exec JIT-compiles the top level statements in the roots chan and,
// if they are expressions, executes them.
func Exec(roots <-chan node, printLLVMIR bool) {
	for n := range roots {
		llvmIR := n.codegen()
		if llvmIR.IsNil() {
			fmt.Fprintln(os.Stderr, "Error: Codegen failed; skipping.")
			continue
		}
		if printLLVMIR {
			llvmIR.Dump()
		}
		if isTopLevelExpr(n) {
			returnval := execEngine.RunFunction(llvmIR, []llvm.GenericValue{})
			fmt.Println(returnval.Float(llvm.DoubleType()))
		}
	}
}

// isTopLevelExpr determines if the node is a top level expression.
// Top level expressions are function nodes with no name.
func isTopLevelExpr(n node) bool {
	return n.Kind() == nodeFunction && n.(*functionNode).proto.(*fnPrototypeNode).name == ""
}
