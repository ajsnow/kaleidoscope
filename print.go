package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	flag.Parse()
	if fn := flag.Arg(0); fn != "" {
		bytes, _ := ioutil.ReadFile(fn)
		str := string(bytes)
		ch := Lex(fn, str)
		// var ch2 = make(chan token, 0)
		// go func() {
		// 	for t := range ch {
		// 		spew.Fdump(os.Stderr, t)
		// 		ch2 <- t
		// 	}
		// }()
		ast := NewTree(fn, ch)
		if ast.Parse() {
			spew.Fdump(os.Stderr, ast)
			spew.Println("Done")
		}
	}
}
