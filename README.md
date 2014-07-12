Kaleidoscope
============

Go port of [LLVM's Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html) using the `github.com/go-llvm/llvm` bindings.

This currently is a fully functional clone of the completed tutorial with slightly modified syntax (`//` instead of `#` delimited comments, optional EOL semicolons). The syntax will be conformant in the future--I'm going to rip out the current lexer soon so I'll take care of that then.

After I'm done cleaning up the current code, I'll break it back up into chapters and port the text of the tutorial as well.

Other Resources
===============

* [LLVM's Official C++ Kaleidoscope Tutorial](http://llvm.org/docs/tutorial/LangImpl1.html)

* If you wanted to impliment your own lexer, [Rob Pike's *Lexical Scanning in Go*](http://www.youtube.com/watch?v=HxaD_trXwRE) is a good starting point for building an interesting, idiomatic Go lexer/parser pair.