package main

import (
	"fmt"

	"github.com/ajsnow/llvm"
	"github.com/davecgh/go-spew/spew"
)

func Exec(roots <-chan node, printAst, printLLVMIR bool) {
	for n := range roots {
		if printAst {
			spew.Dump(n)
		}
		llvmIR := n.codegen()
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
