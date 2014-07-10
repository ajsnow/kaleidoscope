// Kaleidoscope Test Routines


2 + 2                           // Int-like scanning
3.14 * 13.37                    // Float-like scanning
2 - 1 * (2 - (5 + 5) * 2) / 0.5 // Order of operations

def foo(a) a                    // Chaining functions
def double(b) foo(b)*foo(2)
def quad(c) double(c) + double(c) 

def pi() 3.1415927
extern cos(a); extern sin(a)    // External functions
cos(0)                          // (from the c std lib which is compiled into the binary)
//cos(pi()/2)                   // Extern func call on the result of an internal func triggers
//cos(pi())                     // "LLVM ERROR: Tried to execute an unknown external function"
double(cos(3.14))               // Reverse is not the case. Can call internal func on result from external
cos(sin(5))                     // Also works fine
//def myCos(a) cos(a)           // But custom func calling an extern internally triggers same message


def x0(d) 1/2 * (1 + d/1)         // manual Newton's method, since
def x1(d) 1/2 * (x0(d) + d/x0(d)) // we don't have control flow yet :)
def x2(d) 1/2 * (x1(d) + d/x1(d))
def x3(d) 1/2 * (x2(d) + d/x2(d))
def x4(d) 1/2 * (x3(d) + d/x3(d))
def x5(d) 1/2 * (x4(d) + d/x4(d))
def sqrt_naive(d) 1/2 * (x5(d) + d/x5(d))
def square(e) e*e
def abc1_naive(a, b, c) (b+sqrt_naive(4*a*c-square(b)))/(2*a)
def abc2_naive(a, b, c) (b-sqrt_naive(4*a*c-square(b)))/(2*a)


extern sqrt(arg)
extern pow(base, exp)
//def abc1(a, b, c) (b+sqrt(4*a*c-pow(b, 2)))/(2*a) // Fails as above 
//def abc2(a, b, c) (b-sqrt(4*a*c-pow(b, 2)))/(2*a)

abc1_naive(1, 4, 4)
(4+sqrt(4*1*4-pow(4, 2)))/(2*1)