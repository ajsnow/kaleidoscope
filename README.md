Kaleidoscope
============

Go port of [LLVM's Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html) using the [go-llvm/llvm](http://github.com/go-llvm/llvm) <sup>[doc](godoc.org/github.com/go-llvm/llvm)</sup> bindings.

This currently is a fully functional clone of the completed tutorial with slightly modified syntax (optional end of statement semicolons). The syntax will be conformant in the future—I'm currently cleaning up the parser and will add mandatory semicolons during this process.

After I'm done cleaning up the current code, the plan is to break it back up into chapters and port the text of the tutorial as well.

Other Resources
===============

* [LLVM's Official C++ Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html)

* [Rob Pike's *Lexical Scanning in Go*](http://www.youtube.com/watch?v=HxaD_trXwRE) — our lexer is based on the design outlined in this talk.