package tests

import "testing"

// Chapter 14, Program.

func TestS14_Program(t *testing.T) {
	runRows(t, []row{
		// 14 Program step 1: "Process SourceElements for function declarations." before step 2 "Evaluate SourceElements."
		{"14 a function can be called before its declaration", output(`print(f()); function f() { return "hoisted" }`, "hoisted\n")},
		// 14: function declarations are processed even if a later statement fails
		{"14 functions are declared before any statement runs", runtimeError(`print(f()); undeclared; function f() { return "hoisted" }`, "hoisted\n")},
		// 14: SourceElements are processed "for function declarations" in order, so the last of two declarations wins
		{"14 the later of two declarations of the same name wins", output(`print(f()); function f() { return 1 } print(f()); function f() { return 2 }`, "2\n2\n")},
		// 14: SourceElements evaluated in order
		{"14 statements run in source order", output(`print(1); function f() {} print(2); print(3)`, "1\n2\n3\n")},
		// 14 SourceElements step 3: "If Result(2) is a value completion, return Result(2)."
		{"14 a program has the value of its last value statement", output(`print(eval("1; 2; 3"))`, "3\n")},
		// 14 SourceElement : FunctionDeclaration "is evaluated as follows: Return normal completion." and step 4 "Return Result(1)."
		{"14 a function declaration does not change the value of a program", output(`print(eval("1; function f() { return 2 }"))`, "1\n")},
		// 14: a program of function declarations only has no value
		{"14 a program of declarations only has the value undefined", output(`print(eval("function f() {}"))`, "undefined\n")},
		// 14: "SourceElement : Statement | FunctionDeclaration", in any order
		{"14 a program is a sequence of statements and function declarations", output(`var a = 1; function f() { return a } a = 2; print(f())`, "2\n")},
	})
}
