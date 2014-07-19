package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
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

	// handle files
	for _, fn := range flag.Args() {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(-1)
		}
		str := string(b)
		l := NewLex(fn, str)
		ast := NewTree(fn, l.tokens)
		Exec(ast.roots, *printAst, *printLLVMIR)
	}

	// interactive mode
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		l := NewLex("stdin", s.Text()) // probably not the most efficient way to do this
		ast := NewTree("stdin", l.tokens)
		Exec(ast.roots, *printAst, *printLLVMIR)
	}
}
