# Mandelbrot Set Printer

def binary : 1 (x y) y;

def binary> 10 (LHS RHS) RHS < LHS;

def binary| 5 (LHS RHS)
  if LHS then
    1
  else if RHS then
    1
  else
    0;
def unary-(v) 0-v;

extern putchard(char)

def printdensity(d)
  if d > 8 then
    putchard(32)  # ' '
  else if d > 4 then
    putchard(46)  # '.'
  else if d > 2 then
    putchard(43)  # '+'
  else
    putchard(42); # '*'

# Determine whether the specific location diverges.
# Solve for z = z^2 + c in the complex plane.
def mandleconverger(real imag iters creal cimag)
  if iters > 255 | (real*real + imag*imag > 4) then
    iters
  else
    mandleconverger(real*real - imag*imag + creal,
                    2*real*imag + cimag,
                    iters+1, creal, cimag);

# Return the number of iterations required for the iteration to escape
def mandleconverge(real imag)
  mandleconverger(real, imag, 0, real, imag);

# Compute and plot the mandlebrot set with the specified 2 dimensional range
# info.
def mandelhelp(xmin xmax xstep   ymin ymax ystep)
  for y = ymin, y < ymax, ystep in (
    (for x = xmin, x < xmax, xstep in
       printdensity(mandleconverge(x,y)))
    : putchard(10)
  )

# mandel - This is a convenient helper function for plotting the mandelbrot set
# from the specified position with the specified Magnification.
def mandel(realstart imagstart realmag imagmag)
  mandelhelp(realstart, realstart+realmag*78, realmag,
             imagstart, imagstart+imagmag*40, imagmag);

mandel(-2.3, -1.3, 0.05, 0.07)
