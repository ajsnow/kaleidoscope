Kaleidoscope
============

Go port of [LLVM's Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html) using the [go-llvm/llvm](http://github.com/go-llvm/llvm) <sup>[doc](http://godoc.org/github.com/go-llvm/llvm)</sup> bindings.

This is a fully functional clone of the completed tutorial. Currently, I'm refactoring the finished code into ideomatic Go. The lexer and parser are now pretty good. The codegen code, error handling and maybe test integration are what's left. After the refactoring is complete, I will break it back up into chapters and port the text of the tutorial as well.

Other Resources
===============

* [LLVM's Official C++ Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html)

* [Rob Pike's *Lexical Scanning in Go*](http://www.youtube.com/watch?v=HxaD_trXwRE) — our lexer is based on the design outlined in this talk.