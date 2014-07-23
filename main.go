package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	batch       = flag.Bool("b", false, "batch (non-interactive) mode")
	optimized   = flag.Bool("opt", true, "add some optimization passes")
	printTokens = flag.Bool("tok", false, "print tokens")
	printAst    = flag.Bool("ast", false, "print abstract syntax tree")
	printLLVMIR = flag.Bool("llvm", false, "print LLVM generated code")
)

func main() {
	flag.Parse()
	if *optimized {
		Optimize()
	}

	lex := Lex()
	tokens := lex.Tokens()
	if *printTokens {
		tokens = DumpTokens(lex.Tokens())
	}

	// add files for the lexer to lex
	go func() {
		// command line filenames
		for _, fn := range flag.Args() {
			f, err := os.Open(fn)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(-1)
			}
			lex.Add(f)
		}

		// stdin
		if !*batch {
			lex.Add(os.Stdin)
		}
		lex.Done()
	}()

	nodes := Parse(tokens)
	nodesForExec := nodes
	if *printAst {
		nodesForExec = DumpTree(nodes)
	}

	Exec(nodesForExec, *printLLVMIR)
}
