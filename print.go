package main

import (
	"flag"
	"fmt"
	"io/ioutil"
)

func main() {
	flag.Parse()
	if fn := flag.Arg(0); fn != "" {
		bytes, _ := ioutil.ReadFile(fn)
		str := string(bytes)
		ch := Lex(fn, str)
		for tok := range ch {
			if tok.kind != tokSpace {
				fmt.Println(tok)
			}
		}
	}
}
