package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"text/scanner"

	"github.com/ajsnow/llvm"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var s scanner.Scanner
var token rune

func main() {
	if len(os.Args) == 2 {
		f, _ := os.Open(os.Args[1])
		s.Init(f)
		theLoop()
	}

	input := bufio.NewReader(os.Stdin)
	for die := false; !die; {
		fmt.Print("> ")
		l, _, _ := input.ReadLine()
		s.Init(bytes.NewBuffer(l))
		die = theLoop()
	}
}

// Driver

func theLoop() bool {
	for token = s.Scan(); token != scanner.EOF; token = s.Scan() {
		switch token {
		case scanner.Ident:
			name := s.TokenText()
			switch name {
			case "def":
				handleDefinition()
			case "extern":
				handleExtern()
			default:
				handleTopLevelExpression()
			}
		default:
			handleTopLevelExpression()
		}
	}
	return false
}

func handleDefinition() {
	if F := ParseDefinition(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read function definition:")
			LF.Dump()
		}
	} else {
		s.Scan()
	}
}

func handleExtern() {
	if F := ParseExtern(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read extern:")
			LF.Dump()
		}
	} else {
		s.Scan()
	}
}

func handleTopLevelExpression() {
	if F := ParseTopLevelExpr(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			fmt.Fprint(os.Stderr, "Read top-level expression:")
			returnval := executionEngine.RunFunction(LF, []llvm.GenericValue{})
			fmt.Println("Evaluated to", returnval.Float(llvm.DoubleType()))
		}
	} else {
		s.Scan()
	}
}
