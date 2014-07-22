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
		if n.Kind() == nodeFunction {
			p := n.(*functionNode).proto.(*fnPrototypeNode)
			if p.name == "" {
				returnval := execEngine.RunFunction(llvmIR, []llvm.GenericValue{})
				fmt.Println(returnval.Float(llvm.DoubleType()))
			}
		} else {
			// prototype nodes for externs
		}
	}
}
