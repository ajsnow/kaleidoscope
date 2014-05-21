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
		mainLoop()
	}

	input := bufio.NewReader(os.Stdin)
	for die := false; !die; {
		fmt.Print("> ")
		l, _, _ := input.ReadLine()
		s.Init(bytes.NewBuffer(l))
		die = mainLoop()
	}
}

// Driver

func mainLoop() bool {
	for token = s.Scan(); token != scanner.EOF; {
		switch token {
		case ';':
			token = s.Scan()
		case scanner.EOF:
			return true
		case scanner.Ident:
			switch s.TokenText() {
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
	if F := parseDefinition(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			LF.Dump()
		}
	} else {
		s.Scan()
	}
}

func handleExtern() {
	if F := parseExtern(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			LF.Dump()
		}
	} else {
		s.Scan()
	}
}

func handleTopLevelExpression() {
	if F := parseTopLevelExpr(); F != nil {
		if LF := F.codegen(); !LF.IsNil() {
			LF.Dump()
			returnval := executionEngine.RunFunction(LF, []llvm.GenericValue{})
			fmt.Println(returnval.Float(llvm.DoubleType()))
		}
	} else {
		s.Scan()
	}
}
