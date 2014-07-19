package main

import (
	"fmt"
	"os"

	"github.com/ajsnow/llvm"
	"github.com/davecgh/go-spew/spew"
)

// Exec JIT-compiles the top level statements in the roots chan and,
// if they are expressions, executes them.
func Exec(roots <-chan node, printAst, printLLVMIR bool) {
	for n := range roots {
		if printAst {
			spew.Dump(n)
		}
		llvmIR := n.codegen()
		if llvmIR.IsNil() {
			fmt.Fprintln(os.Stderr, "Aborting")
			return
		}
		if printLLVMIR {
			llvmIR.Dump()
		}
		if n.Kind() == nodeFunction {
			p := n.(*functionNode).proto.(*fnPrototypeNode)
			if p.name == "" {
				returnval := executionEngine.RunFunction(llvmIR, []llvm.GenericValue{})
				fmt.Println(returnval.Float(llvm.DoubleType()))
			}
		} else {
			// prototype nodes for externs
		}
	}
}
