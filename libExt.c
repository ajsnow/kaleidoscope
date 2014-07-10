// +build ignore

// Not currently in use; if you'd like to add external
// functions here, just follow the instructions below
// to make them visible to the kaleidoscope JIT.
//
// 1. Run:
//     clang -dynamiclib libExt.c
// 2. Add to main package:
//     err := llvm.LoadLibraryPermanently("./a.out")
//     check(err)
// 3. Now kaleidoscope can see the a.out dynamic
// library that contains this function:
//     > extern putchard(x); putchard(120)
#include <stdio.h>

extern double putchard(double X) {
    char a = (char)X;
    putchar(a);
    fflush(stdout);
    return 0;
}