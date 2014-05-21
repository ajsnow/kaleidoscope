// Kaleidoscope Test Routines


2 + 2                           // Int-like scanning
3.14 * 13.37                    // Float-like scanning
2 - 1 * (2 - (5 + 5) * 2) / 0.5 // PEMDAS

def pi() 3.1415927
//extern cos(a)
//cos(0)                          // from the c std lib which is compiled into the binary
//cos(pi()/2)                     // triggers error:
//cos(pi())                       // "Tried to execute an unknown external function"

def foo(a) a
def double(b) foo(b)*foo(2)
def quad(c) double(c) + double(c)

def x0(d) 1/2 * (1 + d/1)         // manual Newton's method, since
def x1(d) 1/2 * (x0(d) + d/x0(d)) // we don't have control flow yet :)
def x2(d) 1/2 * (x1(d) + d/x1(d))
def x3(d) 1/2 * (x2(d) + d/x2(d))
def x4(d) 1/2 * (x3(d) + d/x3(d))
def x5(d) 1/2 * (x4(d) + d/x4(d))
def sqrt(d) 1/2 * (x5(d) + d/x5(d))

def square(e) e*e

def id(num) square(sqrt(num))

def abc1(a, b, c) (b+sqrt(4*a*c-square(b)))/(2*a)
def abc2(a, b, c) (b-sqrt(4*a*c-square(b)))/(2*a)
abc1(1, 4, 4)