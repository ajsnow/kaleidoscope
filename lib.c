
// Want to call a custom C function from kaleidoscope?
// Good news! Here's how:
// <detailed instructions>

#include <stdio.h>

double putchard(double x) {
    putchar((char)x);
    fflush(stdout);
    return 0;
}