;;; jscheme.ss -- an interpreter for ECMA-262 1st edition (June 1997).
(import (chezscheme))

(include "core.ss")
(include "numconv.ss")
(include "lexer.ss")
(include "parser.ss")
(include "compiler.ss")
(include "builtins.ss")
(include "date.ss")
(include "main.ss")

(suppress-greeting #t)
(scheme-start jscheme-main)
