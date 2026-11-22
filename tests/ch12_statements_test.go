package tests

import "testing"

// Chapter 12, Statements. Completion values are observed as the value of
// eval (§15.1.2.1), which is the value of the last value completion of the
// evaluated Program.

func TestS12_1_Block(t *testing.T) {
	runRows(t, []row{
		// 12.1: StatementList : StatementList Statement evaluates its statements in order
		{"12.1 the statements of a block run in order", output(`{ print(1); print(2); } print(3)`, "1\n2\n3\n")},
		// 12.1: "The production Block : { } ... Return "normal completion"." It carries no value.
		{"12.1 an empty block has no value", output(`print(eval("1; {}"))`, "1\n")},
		// 12.1 StatementList step 4: "If Result(3) is a value completion, return Result(3)."
		{"12.1 a block has the value of its last value statement", output(`print(eval("{ 1; 2 }"), eval("{ 'a'; { 'b'; } }"))`, "2 b\n")},
		// 12.1 StatementList steps 5-9: a statement without a value keeps the value before it
		{"12.1 a statement without a value keeps the previous value", output(`print(eval("{ 1; var x = 2; ; }"))`, "1\n")},
		// 12.1 StatementList step 2: "If Result(1) is an abrupt completion, return Result(1)."
		{"12.1 an abrupt completion skips the rest of the block", output(`while (true) { print(1); break; print(2); } print(3)`, "1\n3\n")},
		// 12.1 StatementList step 7: "return "abrupt completion after value V because of break""
		{"12.1 a break keeps the value computed before it in the block", output(`print(eval("while (true) { 'v'; break; }"))`, "v\n")},
		// 12.1 StatementList step 8: "return "abrupt completion after value V because of continue""
		{"12.1 a continue keeps the value computed before it in the block",
			output(`print(eval("var i = 0; while (i < 2) { i++; 'v' + i; continue; }"))`, "v2\n")},
	})
}

func TestS12_2_VariableStatement(t *testing.T) {
	runRows(t, []row{
		// 12.2: "Variables are created when the execution scope is entered." "initialized to the undefined value when created"
		{"12.2 a variable exists, as undefined, before its declaration runs", output(`print(x); var x = 1; print(x)`, "undefined\n1\n")},
		// 12.2: "A variable with an Initializer is assigned the value of its AssignmentExpression when the VariableStatement is executed"
		{"12.2 the initializer is assigned when the statement runs", output(`var x = 1; print(x); var x = 2; print(x)`, "1\n2\n")},
		// 12.2: VariableDeclaration : Identifier "is evaluated by taking no action", so a redeclaration keeps the value
		{"12.2 a declaration without an initializer keeps the value", output(`var x = 1; var x; print(x)`, "1\n")},
		// 12.2: VariableDeclarationList : VariableDeclarationList , VariableDeclaration
		{"12.2 one statement may declare several variables", output(`var a = 1, b, c = a + 1; print(a, b, c)`, "1 undefined 2\n")},
		// 12.2: "A Block does not define a new execution scope."
		{"12.2 a variable declared in a block is visible after it", output(`{ var x = "in block"; } print(x)`, "in block\n")},
		// 12.2: "If the variable statement occurs inside a FunctionDeclaration, the variables are defined with function-local scope"
		{"12.2 a variable declared in a function is local to it",
			output(`var x = "global"; function f() { var x = "local"; return x } print(f(), x)`, "local global\n")},
		// 12.2: function-local variables exist from the start of the function, shadowing a global of the same name
		{"12.2 a local variable shadows the global before its declaration runs",
			output(`var x = "global"; function f() { var before = x; var x = "local"; return before } print(f())`, "undefined\n")},
		// 12.2: "Otherwise, they are defined with global scope, that is, they are created as members of the global object"
		{"12.2 a global variable is a property of the global object", output(`var g = "value"; print(this.g)`, "value\n")},
		// 12.2 VariableDeclaration steps 1 and 4: "Evaluate Identifier." "Call PutValue(Result(1), Result(3))." The identifier is resolved through with.
		{"12.2 an initializer inside with assigns through the scope chain",
			output(`var o = new Object(); o.p = 1; with (o) { var p = 2; } print(o.p, p)`, "2 undefined\n")},
		// 12.2: "VariableStatement ... Return "normal completion"." It has no value.
		{"12.2 a variable statement has no value", output(`print(eval("1; var x = 2;"))`, "1\n")},
		// 12.2: an Initializer is an AssignmentExpression, so its GetValue can fail
		{"12.2 an initializer reading an undeclared variable is a runtime error", runtimeError(`print("before"); var x = undeclared;`, "before\n")},
	})
}

func TestS12_3_EmptyStatement(t *testing.T) {
	runRows(t, []row{
		// 12.3: EmptyStatement : ; "Return "normal completion"."
		{"12.3 an empty statement does nothing", output(`;;; print(1); ;`, "1\n")},
		// 12.3: an empty statement can be the body of an if or a loop
		{"12.3 an empty statement can be a body", output(`var i = 0; while (i++ < 3); if (true); else print("no"); print(i)`, "4\n")},
		// 12.3: "normal completion" carries no value
		{"12.3 an empty statement has no value", output(`print(eval("1;;"))`, "1\n")},
	})
}

func TestS12_4_ExpressionStatement(t *testing.T) {
	runRows(t, []row{
		// 12.4: "Return "normal completion after value V", where the value V is Result(2)."
		{"12.4 an expression statement has the value of its expression", output(`print(eval("1 + 2"), eval("'a'; 'b'"))`, "3 b\n")},
		// 12.4 step 1: "Evaluate Expression." Its side effects happen.
		{"12.4 an expression statement is evaluated for its effects", output(`var x = 1; x = x + 1; x++; print(x)`, "3\n")},
		// 12.4 step 2: "Call GetValue(Result(1))."
		{"12.4 an expression statement reading an undeclared variable is a runtime error", runtimeError(`print("before"); undeclared;`, "before\n")},
	})
}

func TestS12_5_IfStatement(t *testing.T) {
	runRows(t, []row{
		// 12.5 if-else steps 4-6: a true condition evaluates the first Statement
		{"12.5 a true condition runs the first statement", output(`if (true) print("then"); else print("else");`, "then\n")},
		// 12.5 if-else steps 7-8: a false condition evaluates the second Statement
		{"12.5 a false condition runs the else statement", output(`if (false) print("then"); else print("else");`, "else\n")},
		// 12.5 step 3: "Call ToBoolean(Result(2))."
		{"12.5 the condition is converted with ToBoolean",
			output(`if ("") print(1); else print(2); if ("0") print(3); if (new Boolean(false)) print(4); if (NaN) print(5); else print(6)`, "2\n3\n4\n6\n")},
		// 12.5 if step 4: "If Result(3) is false, return "normal completion"."
		{"12.5 a false condition without else runs nothing", output(`if (false) print("no"); print("after")`, "after\n")},
		// 12.5 grammar: an else belongs to the nearest if
		{"12.5 else belongs to the nearest if", output(`if (true) if (false) print(1); else print(2)`, "2\n")},
		// 12.5 if-else step 6 and step 8: the if statement has the value of the branch that ran
		{"12.5 an if statement has the value of the branch that ran", output(`print(eval("if (true) 'a'; else 'b'"), eval("if (0) 'a'; else 'b'"))`, "a b\n")},
		// 12.5 if step 4: "normal completion" carries no value
		{"12.5 an if that runs nothing has no value", output(`print(eval("1; if (false) 2;"))`, "1\n")},
	})
}

func TestS12_6_IterationStatements(t *testing.T) {
	runRows(t, []row{
		// 12.6.1 steps 2-11: the condition is evaluated before each iteration
		{"12.6.1 while repeats while the condition is true", output(`var i = 0; while (i < 3) { print(i); i++; }`, "0\n1\n2\n")},
		// 12.6.1 step 5: "If Result(3) is false, go to step 12."
		{"12.6.1 while with a false condition never runs its body", output(`while (false) print("no"); print("after")`, "after\n")},
		// 12.6.1 step 4: "Call ToBoolean(Result(2))."
		{"12.6.1 the while condition is converted with ToBoolean", output(`var s = "abc"; while (s) { print(s); s = s.substring(1); }`, "abc\nbc\nc\n")},
		// 12.6.1 step 8: "If Result(6) is a break completion, go to step 12."
		{"12.6.1 break leaves a while loop", output(`var i = 0; while (true) { if (i == 2) break; print(i); i++; } print("done")`, "0\n1\ndone\n")},
		// 12.6.1 step 9: "If Result(6) is a continue completion, go to step 2."
		{"12.6.1 continue goes back to the while condition", output(`var i = 0; while (i < 4) { i++; if (i == 2) continue; print(i); }`, "1\n3\n4\n")},
		// 12.6.1 step 10: "If Result(6) is a return completion, return Result(6)."
		{"12.6.1 return leaves a while loop and the function", output(`function f() { var i = 0; while (true) { if (i == 3) return i; i++; } } print(f())`, "3\n")},
		// 12.6.1 steps 7 and 12: the value is that of the last iteration with a value
		{"12.6.1 a while loop has the value of its last iteration", output(`print(eval("var i = 0; while (i < 3) i++;"))`, "2\n")},
		// 12.6.1 step 1: "Let C be "normal completion"." A loop that never runs has no value.
		{"12.6.1 a while loop that never runs has no value", output(`print(eval("1; while (false) 2;"))`, "1\n")},

		// 12.6.2 steps 2-3: the first Expression is evaluated once
		{"12.6.2 for evaluates its initializer once and its update after each iteration",
			output(logJS+`var i; for (log("init", i = 0); log("test", i < 2); log("update", i++)) print("body")`,
				"init\ntest\nbody\nupdate\ntest\nbody\nupdate\ntest\n")},
		// 12.6.2 steps 1, 5 and 15: all three expressions are optional; a missing condition is true
		{"12.6.2 a for loop with no expressions runs until break", output(`var i = 0; for (;;) { if (i == 3) break; i++; } print(i)`, "3\n")},
		// 12.6.2 step 9: "If Result(8) is false, go to step 19."
		{"12.6.2 a for loop with a false condition never runs its body", output(`for (var i = 0; i < 0; i++) print("no"); print(i)`, "0\n")},
		// 12.6.2 step 13: "If Result(10) is a continue completion, go to step 15." (the update still runs)
		{"12.6.2 continue in a for loop still runs the update",
			output(`var s = ""; for (var i = 0; i < 5; i++) { if (i % 2) continue; s += i; } print(s, i)`, "024 5\n")},
		// 12.6.2 step 12: "If Result(10) is a break completion, go to step 19." (the update does not run)
		{"12.6.2 break in a for loop skips the update", output(`for (var i = 0; i < 5; i++) { if (i == 2) break; } print(i)`, "2\n")},
		// 12.6.2 step 14: "If Result(10) is a return completion, return Result(10)."
		{"12.6.2 return leaves a for loop and the function", output(`function f() { for (var i = 0; ; i++) if (i == 4) return "at " + i; } print(f())`, "at 4\n")},
		// 12.6.2 for ( var VariableDeclarationList ; ... ): the declared variables are ordinary variables of the scope
		{"12.6.2 the for var declarations are visible after the loop",
			output(`for (var i = 0, j = 10; i < j; i++, j--); print(i, j)`, "5 5\n")},
		// 12.6.2 step 8: "Call ToBoolean(Result(7))."
		{"12.6.2 the for condition is converted with ToBoolean", output(`var n = 0; for (var s = "xx"; s; s = s.substring(1)) n++; print(n)`, "2\n")},
		// 12.6.2 steps 11 and 19: the value is that of the last iteration with a value
		{"12.6.2 a for loop has the value of its last iteration", output(`print(eval("for (var i = 0; i < 3; i++) i * 10;"))`, "20\n")},
		// 12.6.2 step 4: "Let C be "normal completion"." A loop that never runs has no value.
		{"12.6.2 a for loop that never runs has no value", output(`print(eval("1; for (; false; ) 2;"))`, "1\n")},

		// 12.6.3 step 5: "Get the name of the next property of Result(3) that doesn't have the DontEnum attribute."
		{"12.6.3 for-in visits each enumerable property",
			linesUnordered(`var o = new Object(); o.a = 1; o.b = 2; o.c = 3; for (var p in o) print(p, o[p])`, "a 1\nb 2\nc 3\n")},
		// 12.6.3 step 5: DontEnum properties are not visited
		{"12.6.3 for-in skips DontEnum properties", output(`var n = 0; for (var p in new Object()) n++; print(n)`, "0\n")},
		// 12.6.3: "Enumerating the properties of an object includes enumerating properties of its prototype"
		{"12.6.3 for-in visits inherited properties",
			linesUnordered(`function F() {} F.prototype.inherited = 1; var o = new F(); o.own = 2; for (var p in o) print(p)`, "inherited\nown\n")},
		// 12.6.3: "a property of a prototype is not enumerated if it is "shadowed""
		{"12.6.3 for-in visits a shadowed property once",
			linesUnordered(`function F() {} F.prototype.x = 1; var o = new F(); o.x = 2; o.y = 3; for (var p in o) print(p, o[p])`, "x 2\ny 3\n")},
		// 12.6.3: "If a property that has not yet been visited during enumeration is deleted, then it will not be visited."
		{"12.6.3 a property deleted before it is visited is not visited",
			output(`var o = new Object(); o.a = 1; o.b = 2; var n = 0; for (var p in o) { n++; delete o.a; delete o.b; } print(n)`, "1\n")},
		// 12.6.3 steps 6-7: "Evaluate the LeftHandSideExpression ( it may be evaluated repeatedly). Call PutValue(Result(6), Result(5))."
		{"12.6.3 the left side may be any left-hand-side expression, evaluated for each property",
			linesUnordered(`var o = new Object(); o.a = 1; o.b = 2; var k = new Object(); var i = 0; for (k[i++] in o); print(i); print(k[0]); print(k[1])`, "2\na\nb\n")},
		// 12.6.3 second form steps 1-5: "Evaluate the Initializer ... Call PutValue" before the Expression
		{"12.6.3 for var with an initializer assigns it before enumerating", output(`var o = new Object(); for (var p = "init" in o) ; print(p)`, "init\n")},
		// 12.6.3 step 3: "Call ToObject(Result(2))." A primitive value is converted to an object.
		{"12.6.3 for-in over a primitive value visits no own properties", output(`var n = 0; for (var p in 5) n++; for (var p in true) n++; print(n)`, "0\n")},
		// 12.6.3 step 3: ToObject of null or undefined generates a runtime error (§9.9)
		{"12.6.3 for-in over null is a runtime error", runtimeError(`print("before"); for (var p in null) ;`, "before\n")},
		// 12.6.3 step 3: ToObject of undefined generates a runtime error (§9.9)
		{"12.6.3 for-in over undefined is a runtime error", runtimeError(`var u; print("before"); for (var p in u) ;`, "before\n")},
		// 12.6.3 steps 10-12: break, continue and return
		{"12.6.3 break and continue in for-in",
			output(`var o = new Object(); o.a = 1; o.b = 2; o.c = 3; var n = 0, m = 0; for (var p in o) { n++; break; } for (var p in o) { continue; m++; } print(n, m)`, "1 0\n")},
		// 12.6.3 step 12: "If Result(8) is a return completion, return Result(8)."
		{"12.6.3 return leaves a for-in loop and the function",
			output(`function f(o) { for (var p in o) return "found " + p; return "none" } var o = new Object(); o.k = 1; print(f(o), f(new Object()))`, "found k none\n")},
		// 12.6.3 steps 9 and 14: the value is that of the last iteration with a value
		{"12.6.3 a for-in loop has the value of its last iteration", output(`print(eval("var o = new Object(); o.only = 1; for (var p in o) p + '!';"))`, "only!\n")},
		// 12.6.3 step 4: "Let C be "normal completion"." A loop over no properties has no value.
		{"12.6.3 a for-in loop that visits nothing has no value", output(`print(eval("1; for (var p in new Object()) 2;"))`, "1\n")},
	})
}

func TestS12_7_ContinueStatement(t *testing.T) {
	runRows(t, []row{
		// 12.7: "Return "abrupt completion because of continue"." Only the innermost loop continues.
		{"12.7 continue goes to the next iteration of the innermost loop",
			output(`var s = ""; for (var i = 0; i < 2; i++) { for (var j = 0; j < 3; j++) { if (j == 1) continue; s += i + "" + j + " "; } } print(s)`, "00 02 10 12 \n")},
		// 12.7: "syntactically incorrect ... if it contains a continue statement that is not within at least one while or for statement"
		{"12.7 continue outside a loop is a syntax error", syntaxError(`print(1); continue;`)},
		// 12.7: a continue nested in an if, but outside any loop, is still outside a loop
		{"12.7 continue in an if outside a loop is a syntax error", syntaxError(`if (true) continue;`)},
		// 12.7: a continue inside a function body that is not in a loop of that function is outside a loop
		{"12.7 continue in a function outside a loop is a syntax error", syntaxError(`function f() { continue; }`)},
	})
}

func TestS12_8_BreakStatement(t *testing.T) {
	runRows(t, []row{
		// 12.8: "Return "abrupt completion because of break"." Only the innermost loop is left.
		{"12.8 break leaves only the innermost loop",
			output(`var s = ""; for (var i = 0; i < 2; i++) { while (true) { s += i; break; } s += "!"; } print(s)`, "0!1!\n")},
		// 12.8: "syntactically incorrect ... if it contains a break statement that is not within at least one while or for statement"
		{"12.8 break outside a loop is a syntax error", syntaxError(`print(1); break;`)},
		// 12.8: a break in a block outside any loop is outside a loop
		{"12.8 break in a block outside a loop is a syntax error", syntaxError(`{ break; }`)},
		// 12.8: a break inside a function body that is not in a loop of that function is outside a loop
		{"12.8 break in a function outside a loop is a syntax error", syntaxError(`function f() { break; }`)},
	})
}

func TestS12_9_ReturnStatement(t *testing.T) {
	runRows(t, []row{
		// 12.9: "Otherwise, the return value is the value of Expression."
		{"12.9 return gives the value of its expression", output(`function f() { return 1 + 2 } print(f())`, "3\n")},
		// 12.9: "If Expression is omitted, the return value is the undefined value."
		{"12.9 return without an expression gives undefined", output(`function f() { return; } print(f())`, "undefined\n")},
		// 12.9: "It causes a function to cease execution and return a value to the caller."
		{"12.9 return stops the function", output(`function f() { print("a"); return; print("b") } f(); print("c")`, "a\nc\n")},
		// 12.9: a function that ends without return gives undefined (§13.2.1)
		{"12.9 a function without return gives undefined", output(`function f() { 1 } print(f())`, "undefined\n")},
		// 12.9: "syntactically incorrect ... if it contains a return statement that is not within the Block of a FunctionDeclaration"
		{"12.9 return outside a function is a syntax error", syntaxError(`print(1); return 1;`)},
		// 12.9 step 3: "Call GetValue(Result(2))."
		{"12.9 returning an undeclared variable is a runtime error", runtimeError(`function f() { return undeclared } print("before"); f()`, "before\n")},
	})
}

func TestS12_10_WithStatement(t *testing.T) {
	runRows(t, []row{
		// 12.10 step 4: "Add Result(3) to the front of the scope chain."
		{"12.10 with makes the properties of an object visible as variables",
			output(`var o = new Object(); o.a = "from o"; var a = "global"; with (o) print(a); print(a)`, "from o\nglobal\n")},
		// 12.10 step 5: names the object does not have are resolved further down the scope chain
		{"12.10 names missing from the with object are resolved outside it", output(`var g = "global"; with (new Object()) print(g)`, "global\n")},
		// 12.10: assignment to a name the object has changes the object's property
		{"12.10 assignment inside with changes the object's property",
			output(`var o = new Object(); o.a = 1; var a = "global"; with (o) a = 2; print(o.a, a)`, "2 global\n")},
		// 12.10: assignment to a name the object lacks goes to the scope chain below it (here, the global object)
		{"12.10 assignment inside with to a name the object lacks does not add it",
			output(`var o = new Object(); with (o) b = 3; print(o.b, b)`, "undefined 3\n")},
		// 12.10 step 3: "Call ToObject(Result(2))." A primitive becomes an object.
		{"12.10 with converts a primitive to an object", output(`with ("abc") print(length, charAt(1))`, "3 b\n")},
		// 12.10 step 3: ToObject of null generates a runtime error (§9.9)
		{"12.10 with null is a runtime error", runtimeError(`print("before"); with (null) print("inside")`, "before\n")},
		// 12.10 step 6: "Remove Result(3) from the front of the scope chain." after a normal completion
		{"12.10 the scope chain is restored after with", output(`var o = new Object(); o.x = "o"; var x = "global"; with (o) { x; } print(x)`, "global\n")},
		// 12.10: "no matter how control leaves the embedded Statement ... the scope chain is always restored"
		{"12.10 the scope chain is restored after a break out of with",
			output(`var o = new Object(); o.x = "o"; var x = "global"; while (true) { with (o) { break; } } print(x)`, "global\n")},
		// 12.10: the scope chain is restored after a continue out of with
		{"12.10 the scope chain is restored after a continue out of with",
			output(`var o = new Object(); o.x = "o"; var x = "global"; var i = 0; while (i++ < 2) { print(x); with (o) { continue; } } print(x)`, "global\nglobal\nglobal\n")},
		// 12.10: the scope chain is restored after a return out of with
		{"12.10 the scope chain is restored after a return out of with",
			output(`var o = new Object(); o.x = "o"; var x = "global"; function f() { with (o) return x } print(f(), x)`, "o global\n")},
		// 12.10: with statements nest; the innermost object comes first
		{"12.10 nested with statements search the innermost object first",
			output(`var a = new Object(), b = new Object(); a.x = "a"; a.y = "a"; b.x = "b"; with (a) with (b) print(x, y)`, "b a\n")},
		// 12.10 step 7: "Return Result(5)."
		{"12.10 a with statement has the value of its statement", output(`print(eval("var o = new Object(); o.v = 'w'; with (o) v;"))`, "w\n")},
	})
}
