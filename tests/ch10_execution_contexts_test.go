package tests

import "testing"

// Chapter 10, Execution Contexts. Contexts, variable objects and scope
// chains are internal; they are observed through variable lookup, this,
// delete, for-in and the arguments object.

func TestS10_1_Definitions(t *testing.T) {
	runRows(t, []row{
		// 10.1.1: "Declared functions are defined in source text by a FunctionDeclaration."
		{"10.1.1 a declared function is a function object", output(`function f() { return "declared" } print(typeof f, f())`, "function declared\n")},
		// 10.1.1: "Anonymous functions are created dynamically by using the built-in Function object as a constructor"
		{"10.1.1 an anonymous function is created by the Function constructor", output(`var g = new Function("return 'anonymous'"); print(typeof g, g())`, "function anonymous\n")},
		// 10.1.1: "Internal functions are built-in objects of the language, such as parseInt and Math.exp."
		{"10.1.1 an internal function is a function object", output(`print(typeof parseInt, typeof Math.exp, parseInt("12"))`, "function function 12\n")},

		// 10.1.2: "Global code is source text that is outside all function declarations."
		{"10.1.2 global code runs, function code runs only when called", output(`print("global"); function f() { print("function") } print("end")`, "global\nend\n")},
		// 10.1.2: "if the parameter to the built-in eval function is a string, it is treated as an ECMAScript Program"
		{"10.1.2 eval code is a program", output(`print(eval("function sq(x) { return x * x } sq(5)"))`, "25\n")},
		// 10.1.2: "the last parameter provided in an instantiation of Function is converted to a string and treated as the StatementList"
		{"10.1.2 the last argument of Function is the body", output(`var f = new Function("a", "b", "return a + b"); print(f(1, 2))`, "3\n")},
		// 10.1.2: "all parameters except the last one are converted to strings and concatenated together, separated by commas"
		{"10.1.2 the other arguments of Function are joined into the parameter list", output(`var f = new Function("a, b", "c", "return a * b * c"); print(f(2, 3, 4))`, "24\n")},
		// 10.1.2: "Function code is source text that is inside a function declaration ... the Block"
		{"10.1.2 function code is the block of a declaration", output(`function f() { print("in f"); return 1 } print(f() + 1)`, "in f\n2\n")},

		// 10.1.3: "For each FunctionDeclaration in the code, in source text order ... If the variable object already has a property with this name, replace its value"
		{"10.1.3 the last declaration of a function name wins", output(`print(f()); function f() { return 1 } function f() { return 2 }`, "2\n")},
		// 10.1.3: "If the caller supplies fewer parameter values than there are formal parameters, the extra formal parameters have value undefined."
		{"10.1.3 missing arguments leave parameters undefined", output(`function f(a, b) { return typeof b } print(f(1))`, "undefined\n")},
		// 10.1.3: "If two or more formal parameters share the same name ... the corresponding property is given the value that was supplied for the last parameter with this name."
		{"10.1.3 a repeated parameter name takes the last value", output(`function f(a, a) { return a } print(f(1, 2))`, "2\n")},
		// 10.1.3: "If the value of this last parameter was not supplied by the caller, the value of the corresponding property is undefined."
		{"10.1.3 a repeated parameter name not supplied last is undefined", output(`function f(a, a) { return a } print(f(1))`, "undefined\n")},
		// 10.1.3: "If there is already a property of the variable object with the name of a declared variable, the value of the property and its attributes are not changed."
		{"10.1.3 a variable declaration does not reset a parameter", output(`function f(a) { var a; return a } print(f(5))`, "5\n")},
		// 10.1.3: "if a declared variable has the same name as a declared function ... the variable declaration does not disturb the existing property"
		{"10.1.3 a variable declaration does not reset a function", output(`print(typeof f); var f; function f() {}`, "function\n")},
		// 10.1.3: the variable's Initializer still assigns when it runs (§12.2)
		{"10.1.3 a variable initializer replaces a function when it runs", output(`print(typeof f); var f = 1; print(typeof f); function f() {}`, "function\nnumber\n")},
		// 10.1.3: "For each VariableDeclaration in the code, create a property of the variable object ... whose value is undefined"
		{"10.1.3 every variable of a function exists from the start", output(`function f() { var r = typeof v; if (false) { var v = 1 } return r + " " + v } print(f())`, "undefined undefined\n")},

		// 10.1.4 step 3: the first object of the scope chain that has the property gives the reference
		{"10.1.4 a local variable is found before a global one", output(`var x = "global"; function f() { var x = "local"; return x } print(f())`, "local\n")},
		// 10.1.4 steps 1-4: a name not in the activation object is looked up in the global object
		{"10.1.4 a global variable is visible in a function", output(`var x = "global"; function f() { return x } print(f())`, "global\n")},
		// 10.1.4 step 5: "Return a value of type Reference whose base object is null" (GetValue then fails)
		{"10.1.4 an unresolvable name is a runtime error when read", runtimeError(`function f() { return nowhere } print("before"); f()`, "before\n")},
		// 10.1.4: "When execution enters a with block, the object specified in the with statement is added to the front of the scope chain."
		{"10.1.4 a with object is searched first", output(`var x = "global"; var o = new Object(); o.x = "o"; with (o) print(x)`, "o\n")},
		// 10.1.4: "the scope chain of the execution context is affected only by WithStatement" so a called function does not see the caller's with
		{"10.1.4 a function called inside with does not see the with object",
			output(`function f() { return x } var x = "global"; var o = new Object(); o.x = "o"; with (o) print(f())`, "global\n")},
		// 10.1.4: a function does not see the local variables of its caller
		{"10.1.4 a function does not see the variables of its caller",
			output(`function g() { return typeof secret } function f() { var secret = 1; return g() } print(f())`, "undefined\n")},

		// 10.1.5: "Built-in objects such as Math, String, Date, parseInt, etc. These have attributes { DontEnum }."
		{"10.1.5 the built-in properties of the global object are not enumerated",
			output(`var found = ""; for (var p in this) if (p == "Math" || p == "String" || p == "Date" || p == "parseInt" || p == "eval") found += p; print("[" + found + "]")`, "[]\n")},
		// 10.1.5: "the initial properties may be changed"
		{"10.1.5 a built-in property of the global object can be replaced", output(`String = 5; parseInt = "p"; print(String, parseInt)`, "5 p\n")},
		// 10.1.5: the global object holds the built-ins; global code's this is the global object (§10.2.1)
		{"10.1.5 the built-ins are properties of the global object", output(`print(this.Math == Math, this.parseInt == parseInt, typeof this.Object)`, "true true function\n")},

		// 10.1.6: "The activation object is initialized with a property with name arguments and property attributes { DontDelete }."
		{"10.1.6 arguments is a DontDelete property of the activation object", output(`function f() { var r = delete arguments; return r + " " + typeof arguments } print(f())`, "false object\n")},
		// 10.1.6: "the arguments object ... is used as the new value of the arguments property of the function object"
		{"10.1.6 f.arguments is the arguments object while f runs", output(`function f(a) { return f.arguments[0] + " " + f.arguments.length } print(f("x", "y"))`, "x 2\n")},
		// 10.1.6: "the value of the [[OldArguments]] property of the activation object is stored into the arguments property"
		{"10.1.6 f.arguments is restored when a recursive call returns", output(`function f(n) { if (n > 0) f(n - 1); return f.arguments[0] } print(f(2))`, "2\n")},
		// 10.1.6: "if the activation object has no [[OldArguments]] property, then the arguments property of the function object is deleted"
		{"10.1.6 f.arguments holds no arguments object after f returns", output(`function f() {} f(1); print(f.arguments == null)`, "true\n")},

		// 10.1.7: "The this value associated with an execution context is immutable." this is not a reference.
		{"10.1.7 this cannot be assigned", earlyOrRuntimeError(`this = 1`)},
		// 10.1.7: "The this value depends on the caller"
		{"10.1.7 this depends on how the function is called",
			output(`function f() { return this.tag } var a = new Object(), b = new Object(); a.tag = "a"; b.tag = "b"; a.f = f; b.f = f; print(a.f(), b.f())`, "a b\n")},

		// 10.1.8: "The [[Prototype]] of the arguments object is to the original Object prototype object"
		{"10.1.8 the arguments object inherits from Object.prototype",
			output(`function f() { return arguments.toString == Object.prototype.toString } print(f())`, "true\n")},
		// 10.1.8: an arguments object is an ordinary object, so its class is Object
		{"10.1.8 the arguments object is of class Object",
			output(`function f() { arguments.ts = Object.prototype.toString; return arguments.ts() } print(f())`, "[object Object]\n")},
		// 10.1.8: "A property is created with name callee ... The initial value of this property is the function object being executed."
		{"10.1.8 arguments.callee is the running function", output(`function f() { return arguments.callee == f } print(f())`, "true\n")},
		// 10.1.8: "A property is created with name length ... the number of actual parameter values supplied by the caller."
		{"10.1.8 arguments.length is the number of arguments supplied", output(`function f(a, b) { return arguments.length } print(f(), f(1), f(1, 2, 3))`, "0 1 3\n")},
		// 10.1.8: "a property is created with name ToString(iarg) ... the value of the corresponding actual parameter"
		{"10.1.8 the arguments are indexed from 0", output(`function f() { return arguments[0] + arguments["1"] + arguments[2] } print(f("a", "b", "c"))`, "abc\n")},
		// 10.1.8: callee, length and the index properties have the attribute DontEnum
		{"10.1.8 the properties of the arguments object are not enumerated",
			output(`function f() { var n = 0; for (var p in arguments) n++; return n } print(f(1, 2, 3))`, "0\n")},
		// 10.1.8: "changing this property changes the corresponding property of the activation object"
		{"10.1.8 writing an argument changes the parameter", output(`function f(a) { arguments[0] = "changed"; return a } print(f("orig"))`, "changed\n")},
		// 10.1.8: "and vice versa"
		{"10.1.8 writing a parameter changes the argument", output(`function f(a) { a = "changed"; return arguments[0] } print(f("orig"))`, "changed\n")},
		// 10.1.8: "For each non-negative integer, iarg, less than the value of the length property" only; a missing argument has no property
		{"10.1.8 a parameter beyond the supplied arguments is not shared", output(`function f(a, b) { b = 2; return arguments[1] + " " + arguments.length } print(f(1))`, "undefined 1\n")},
	})
}

func TestS10_2_EnteringAnExecutionContext(t *testing.T) {
	runRows(t, []row{
		// 10.2.1: "The scope chain is created and initialized to contain the global object and no others."
		{"10.2.1 global code resolves names in the global object", output(`this.viaThis = "set on this"; print(viaThis)`, "set on this\n")},
		// 10.2.1: "Variable instantiation is performed using the global object as the variable object"
		{"10.2.1 global variables and functions are properties of the global object", output(`var gv = 1; function gf() {} print(this.gv, typeof this.gf)`, "1 function\n")},
		// 10.2.1: "... and using empty property attributes": enumerable
		{"10.2.1 global variables are enumerable", output(`var gv = 1; var found = ""; for (var p in this) if (p == "gv") found += p; print(found)`, "gv\n")},
		// 10.2.1: "... and using empty property attributes": deletable
		{"10.2.1 global variables and functions can be deleted",
			output(`var gv = 1; function gf() {} print(delete gv, delete gf, typeof gv, typeof gf)`, "true true undefined undefined\n")},
		// 10.2.1: "The this value is the global object."
		{"10.2.1 this in global code is the global object", output(`var gv = "g"; print(this.gv, this.Math == Math)`, "g true\n")},

		// 10.2.2: "The scope chain is initialized to contain the same objects, in the same order, as the calling context's scope chain."
		{"10.2.2 eval code sees the local variables of its caller", output(`function f() { var local = "local"; return eval("local") } print(f())`, "local\n")},
		// 10.2.2: "This includes objects added to the calling context's scope chain by WithStatement."
		{"10.2.2 eval code sees the with objects of its caller", output(`var o = new Object(); o.x = "o"; var x = "global"; with (o) print(eval("x"))`, "o\n")},
		// 10.2.2: the calling context's scope chain is used whatever name eval is called through
		{"10.2.2 eval called through another name sees the local variables of its caller",
			output(`function f() { var x = 1; var e = eval; return e("typeof x") } print(f())`, "number\n")},
		// 10.2.2: "Variable instantiation is performed using the calling context's variable object"
		{"10.2.2 a variable declared by eval in a function is local to it",
			output(`function f() { eval("var v = 'from eval'"); return v } print(f(), typeof v)`, "from eval undefined\n")},
		// 10.2.2: "... and using empty property attributes": a variable declared by eval can be deleted
		{"10.2.2 a variable declared by eval can be deleted", output(`function f() { eval("var v = 1"); return delete v } print(f())`, "true\n")},
		// 10.2.2: "The this value is the same as the this value of the calling context."
		{"10.2.2 eval code has the this of its caller", output(`function m() { return eval("this.tag") } var o = new Object(); o.tag = "o"; o.m = m; print(o.m())`, "o\n")},
		// 10.2.2: in global code the calling context is global code, so eval declares global variables
		{"10.2.2 a variable declared by eval in global code is global", output(`eval("var gv = 'global'"); print(this.gv)`, "global\n")},

		// 10.2.3: "The scope chain is initialized to contain the activation object followed by the global object."
		{"10.2.3 function code sees its own variables and the globals",
			output(`var g = "g"; function f(p) { var l = "l"; return p + l + g } print(f("p"))`, "plg\n")},
		// 10.2.3: "Variable instantiation is performed using the activation object ... and using property attributes { DontDelete }."
		{"10.2.3 the variables of a function cannot be deleted", output(`function f(p) { var v = 1; return (delete v) + " " + (delete p) } print(f(1))`, "false false\n")},
		// 10.2.3: "If the this value provided by the caller is not an object (including the case where it is null), then the this value is the global object."
		{"10.2.3 a plain call gets the global object as this", output(`var gv = "global"; function f() { return this.gv } print(f())`, "global\n")},
		// 10.2.3: "The caller provides the this value."
		{"10.2.3 a method call gets its object as this", output(`function f() { return this.v } var o = new Object(); o.v = "o"; o.f = f; print(o.f())`, "o\n")},
		// 10.2.3: every call enters a new execution context, with no limit on the depth of recursion
		{"10.2.3 recursion has no fixed depth limit",
			output(`function depth(n) { return n == 0 ? 0 : 1 + depth(n - 1) } print(depth(100000))`, "100000\n")},
		// 10.2.3: each call has its own activation object
		{"10.2.3 each call has its own variables", output(`function f(n) { var mine = n; if (n > 0) f(n - 1); return mine } print(f(3))`, "3\n")},
	})
}
