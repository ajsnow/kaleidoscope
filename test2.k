# Kaleidoscope Test Routines

2 + 2                           # Int-like scanning
3.14 * 13.37                    # Float-like scanning
2 - 1 * (2 - (5 + 5) * 2) / 0.5 # Order of operations

def foo(a) a                    # Chaining functions
def double(b) foo(b)*foo(2)
def quad(c) double(c) + double(c)
quad(5)

extern cos(a); extern sin(a)    # External functions
def pi() 3.14159265358979323846
cos(0)
cos(pi()/2)
cos(pi())
double(cos(3.14))
cos(sin(5))

extern putchard(char)           # External func via Cgo C
putchard(67)
extern cgoputchard(char)        # External func via Cgo Go (via <stdio.h> calls)
cgoputchard(71)
extern goputchard(char)         # External func via Cgo Go (via fmt)
goputchard(79)

def x0(d) 1/2 * (1 + d/1)         # Manual Newton's method, for 
def x1(d) 1/2 * (x0(d) + d/x0(d)) # before we impliment branches.
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

abc1_naive(1, 4, 4)  # Let's see how ours compares to the 
abc1_c(1, 4, 4)      # easy version that uses math.h.

# If Expr
def fib(x) if x < 3 then 1 else fib(x-1)+fib(x-2)
fib(20)

# For Loop
def printstar(n) for i = 1, i < n, 1.0 in putchard(42)
printstar(5)

# User-defined Binary Operators
def binary!(l,r) l * 2 + r / 9
15 ! 18
def binary?50(l,r) r*r
sqrt(2 ? (15 ! 74 ? sqrt(18)))
def binary∆(l,r) if l < r then r else l
32 ∆ 2 ∆ 4 ∆ 16 ∆ 96 ∆ 1

# Mutable Variables
extern printd(x)
def binary:1(x,y) y
def mut(x) 
    printd(x) : 
    x = 4 :
    printd(x)

mut(123)

# Declared Variables
def fibi(x)
  var a = 0, b = 1, c in
  (for i = 2, i < x in
     c = a + b :
     a = b :
     b = c ) :
  b;
fibi(20)

# Expected output:
# 4
# 41.9818
# 38
# 20
# 1
# 6.123233995736766e-17
# -1
# -1.999997463455079
# 0.574400879193934
# C0                   # 'C' printed; 0 returned.
# G0                   # 'G' printed; 0 returned.
# O0                   # 'O' printed; 0 returned.
# 2.00390625           # Didn't bother confirming this.
# 2
# 6765
# *****0               # "*****" printed; 0 returned.
# 32
# 32
# 96
# 123
# 4
# 0
# 6765
