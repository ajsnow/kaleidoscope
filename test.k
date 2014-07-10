// Kaleidoscope Test Routines

2 + 2                           // Int-like scanning
3.14 * 13.37                    // Float-like scanning
2 - 1 * (2 - (5 + 5) * 2) / 0.5 // Order of operations

def foo(a) a                    // Chaining functions
def double(b) foo(b)*foo(2)
def quad(c) double(c) + double(c)
quad(5)

extern cos(a); extern sin(a)    // External functions
def pi() 3.14159265358979323846
cos(0)
cos(pi()/2)
cos(pi())
double(cos(3.14))
cos(sin(5))

def x0(d) 1/2 * (1 + d/1)         // Manual Newton's method, for 
def x1(d) 1/2 * (x0(d) + d/x0(d)) // before we impliment branches.
def x2(d) 1/2 * (x1(d) + d/x1(d))
def x3(d) 1/2 * (x2(d) + d/x2(d))
def x4(d) 1/2 * (x3(d) + d/x3(d))
def x5(d) 1/2 * (x4(d) + d/x4(d))
def sqrt_naive(d) 1/2 * (x5(d) + d/x5(d))
def square(e) e*e
def abc1_naive(a, b, c) (b+sqrt_naive(4*a*c-square(b)))/(2*a)
def abc2_naive(a, b, c) (b-sqrt_naive(4*a*c-square(b)))/(2*a)

extern sqrt(arg); extern pow(base, exp)
def abc1_c(a, b, c) (b+sqrt(4*a*c-pow(b, 2)))/(2*a)
def abc2_c(a, b, c) (b-sqrt(4*a*c-pow(b, 2)))/(2*a)

abc1_naive(1, 4, 4)  // Let's see how ours compares to the 
abc1_c(1, 4, 4)      // easy version that uses math.h.



// Correct output:
// 4
// 41.9818
// 38
// 20
// 1
// 6.123233995736766e-17
// -1
// -1.999997463455079
// 0.574400879193934
// 2.00390625           // Didn't bother confirming this.
// 2