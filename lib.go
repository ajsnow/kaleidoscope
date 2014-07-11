package main

// Want to call a Go function from kaleidoscope?
// Good news! Here's how:
// <detailed instructions>

// We've separated external Go and C funcs into lib.go/lib.c because of
// a Cgo limitation. Quoting the Go Blog:
// "[I]f your program uses any //export directives, then the C code in
// the comment may only include declarations (extern int f();), not
// definitions (int f() { return 1; }). You can use //export directives
// to make Go functions accessible to C code."[^1](http://blog.golang.org/c-go-cgo)

// #include <stdio.h>
import "C"
import "fmt"

//export cgoputchard
func cgoputchard(x C.double) C.double {
	C.putchar(C.int(x))
	C.fflush(C.stdout)
	return 0
}

//export goputchard
func goputchard(x float64) float64 {
	fmt.Printf("%c", rune(x))
	return 0
}
