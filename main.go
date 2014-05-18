package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	FPM.AddInstructionCombiningPass()
	FPM.AddReassociatePass()
	FPM.AddGVNPass()
	FPM.AddCFGSimplificationPass()
	FPM.InitializeFunc()

	input := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		r, _, _ := input.ReadLine()
		scanner = bufio.NewScanner(bytes.NewBuffer(r))
		scanner.Split(bufio.ScanWords)
		getNextToken()
		mainLoop()
	}
}
