package tests

import "testing"

// Chapter 13, Function Definition.

func TestS13_FunctionDefinition(t *testing.T) {
	runRows(t, []row{
		// 13: "Defines a property of the global object whose name is the Identifier and whose value is a function object"
		{"13 a function declaration defines a global property", output(`function f() { return 1 } print(typeof this.f, this.f())`, "function 1\n")},
		// 13: FunctionDeclaration : function Identifier ( ) Block — "Create a new Function object ... with no parameters"
		{"13 a function without parameters", output(`function f() { return arguments.length } print(f(), f.length)`, "0 0\n")},
		// 13: "Create a new Function object (15.3.2.1) with the parameters specified by the FormalParameterList"
		{"13 a function with parameters binds its arguments to them", output(`function f(a, b, c) { return a + b + c } print(f(1, 2, 3), f.length)`, "6 3\n")},
		// 13: "the Block as the body"
		{"13 the body of a function runs when it is called", output(`function f() { print("body") } print("before"); f(); f()`, "before\nbody\nbody\n")},
		// 13: "If the function definition is supplied text to the eval function and the calling context has an activation object, then the declared function is added to the activation object"
		{"13 a function declared by eval in a function is local to it",
			output(`function f() { eval("function g() { return 'g' }"); return g() } print(f(), typeof g)`, "g undefined\n")},
		// 13: eval in global code declares a global function
		{"13 a function declared by eval in global code is global", output(`eval("function g() { return 'g' }"); print(this.g())`, "g\n")},
		// 13: FormalParameterList : Identifier | FormalParameterList , Identifier
		{"13 parameters must be identifiers", syntaxError(`function f(1) {}`)},
		// 13: the body of a function is a Block
		{"13 a function body must be a block", syntaxError(`function f() return 1;`)},
		// 13: a FunctionDeclaration needs an Identifier
		{"13 a function declaration needs a name", syntaxError(`function () {}`)},
		// 13: each declaration creates its own function object, each with its own prototype property (§15.3.5.2)
		{"13 each declared function is a distinct object", output(`function f() {} function g() {} print(f == g, f.prototype == g.prototype)`, "false false\n")},
	})
}
