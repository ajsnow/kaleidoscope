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
		l := lex(fn, str)
		for tok := range l.tokens {
			if tok.kind != tokSpace {
				fmt.Println(tok)
			}
		}
	}
}
