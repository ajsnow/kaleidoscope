package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ajsnow/llvm"
	"github.com/davecgh/go-spew/spew"
)

var (
	optimized   = flag.Bool("opt", true, "add some optimization passes")
	printLLVMIR = flag.Bool("llvm", false, "print LLVM generated code")
	printAst    = flag.Bool("ast", false, "print abstract syntax tree")
)

func main() {
	flag.Parse()
	if *optimized {
		optimize()
	}

	if fn := flag.Arg(0); fn != "" {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(-1)
		}
		str := string(b)
		ch := Lex(fn, str)
		ast := NewTree(fn, ch)
		if ast.Parse() && *printAst {
			spew.Dump(ast)
		}
		for _, n := range ast.root.nodes {
			llvmIR := n.codegen()
			if *printLLVMIR {
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
}
