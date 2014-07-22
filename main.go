package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
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

	lex := Lex(*printTokens)
	nodes := Parse(lex.Tokens(), *printAst)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		Exec(nodes, *printLLVMIR)
		wg.Done()
	}()

	// handle files
	for _, fn := range flag.Args() {
		f, err := os.Open(fn)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(-1)
		}
		lex.Add(f)
	}

	// handle stdin
	if !*batch {
		lex.Add(os.Stdin)
	}

	lex.Done()
	wg.Wait()
}
