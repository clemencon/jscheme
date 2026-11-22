package tests

import "testing"

// Chapter 11, Expressions.

// Arrange helpers for evaluation-order rows. ES1 has no function
// expressions or object literals, so they are declared in the program.

// logJS defines log(s, v): print s, then return v.
const logJS = `function log(s, v) { print(s); return v } `

// objJS defines obj(name, v): an object whose valueOf prints name and
// returns v, so the moment it is converted to a primitive shows in stdout.
const objJS = `function loggedValueOf() { print(this.name); return this.v } ` +
	`function obj(name, v) { var o = new Object(); o.name = name; o.v = v; o.valueOf = loggedValueOf; return o } `

func TestS11_1_PrimaryExpressions(t *testing.T) {
	runRows(t, []row{
		// 11.1.1: "The this keyword evaluates to the this value of the execution context." Global code: the global object (§10.2.1).
		{"11.1.1 this in global code is the global object", output(`var g = "global var"; print(this.g)`, "global var\n")},
		// 11.1.1: in function code called as a method, this is the base object (§11.2.3)
		{"11.1.1 this in a method is the object it was called on",
			output(`function m() { return this.tag } var o = new Object(); o.tag = "o"; o.m = m; print(o.m())`, "o\n")},

		// 11.1.2: "The result of an Identifier is always a value of type Reference." So it can be assigned.
		{"11.1.2 an identifier is a reference that can be assigned", output(`var x = 1; x = 2; print(x)`, "2\n")},
		// 11.1.2: "An Identifier is evaluated using the scoping rules stated in section 10.1.4"
		{"11.1.2 an identifier is resolved when it is evaluated",
			output(`function f() { return later } var later = "found"; print(f())`, "found\n")},

		// 11.1.3: "A Literal is evaluated as described in section 7.7."
		{"11.1.3 a literal evaluates to its value", output(`print(12, "s", true, null)`, "12 s true null\n")},

		// 11.1.4: "Evaluate Expression." The grouping operator overrides precedence.
		{"11.1.4 parentheses group an expression", output(`print((1 + 2) * 3, 1 + 2 * 3)`, "9 7\n")},
		// 11.1.4: "Note that this algorithm does not apply GetValue to Result(1)." So the group is still assignable.
		{"11.1.4 a parenthesised reference can be assigned", output(`var o = new Object(); (o.p) = 2; var x; (x) = 3; print(o.p, x)`, "2 3\n")},
		// 11.1.4: "so that operators such as delete and typeof may be applied to parenthized expressions"
		{"11.1.4 delete and typeof see through parentheses",
			output(`var o = new Object(); o.p = 1; print(delete (o.p), o.p, typeof (undeclared))`, "true undefined undefined\n")},
		// 11.1.4: the reference is kept, so a parenthesised method call still gets its base as this
		{"11.1.4 a parenthesised method reference keeps its this",
			output(`function m() { return this.tag } var o = new Object(); o.tag = "o"; o.m = m; print((o.m)())`, "o\n")},
	})
}

func TestS11_2_LeftHandSideExpressions(t *testing.T) {
	runRows(t, []row{
		// 11.2.1: "MemberExpression . Identifier is identical in its behavior to MemberExpression [ <identifier-string> ]"
		{"11.2.1 dot notation is bracket notation with the identifier as a string",
			output(`var o = new Object(); o.abc = 1; o["def"] = 2; print(o["abc"], o.def)`, "1 2\n")},
		// 11.2.1 step 6: "Call ToString(Result(4))." The property name is a string.
		{"11.2.1 a bracket expression is converted to a string",
			output(`var o = new Object(); o[1] = "one"; o[1.5] = "x"; o[null] = "n"; o[true] = "t"; print(o["1"], o["1.5"], o["null"], o["true"])`, "one x n t\n")},
		// 11.2.1 step 6: ToString of an object key uses its toString (§9.8)
		{"11.2.1 an object used as a property name is converted with toString",
			output(`function k() { return "key" } var key = new Object(); key.toString = k; var o = new Object(); o[key] = 1; print(o.key)`, "1\n")},
		// 11.2.1 step 5: "Call ToObject(Result(2))." A primitive base is converted to an object.
		{"11.2.1 a property of a primitive value is read through its object", output(`print("abc"["length"], "abc".length, typeof (5)["toString"])`, "3 3 function\n")},
		// 11.2.1 steps 3-5: the Expression is evaluated before ToObject fails on a null base
		{"11.2.1 the bracket expression is evaluated before a null base fails",
			runtimeError(logJS+`var n = null; n[log("key", "x")]`, "key\n")},
		// 11.2.1 steps 5-6: ToObject comes before ToString, so a null base fails before the key is converted
		{"11.2.1 a null base fails before the key is converted to a string",
			runtimeError(`function k() { print("toString called"); return "x" } var key = new Object(); key.toString = k; var n = null; print("before"); n[key]`, "before\n")},
		// 11.2.1: "The production CallExpression : CallExpression [ Expression ] is evaluated in exactly the same manner"
		{"11.2.1 a property of a call result can be accessed",
			output(`function f() { var o = new Object(); o.x = "fx"; return o } print(f().x, f()["x"])`, "fx fx\n")},

		// 11.2.2: "new NewExpression ... providing no arguments (that is, an empty list of arguments)"
		{"11.2.2 new without arguments calls the constructor with none",
			output(`function F() { this.n = arguments.length } var a = new F; print(a.n)`, "0\n")},
		// 11.2.2: "new MemberExpression Arguments ... providing the list Result(3) as the argument values"
		{"11.2.2 new passes its arguments to the constructor",
			output(`function F(a, b) { this.sum = a + b } print(new F(2, 3).sum)`, "5\n")},
		// 11.2.2: new MemberExpression Arguments takes a member expression as the constructor
		{"11.2.2 new can construct through a property", output(`function F(v) { this.v = v } var o = new Object(); o.F = F; print(new o.F(7).v)`, "7\n")},
		// 11.2.2 step 3: "If Type(Result(2)) is not Object, generate a runtime error."
		{"11.2.2 new on a value that is not an object is a runtime error", runtimeError(`var x = 1; print("before"); new x`, "before\n")},
		// 11.2.2 step 4: "If Result(2) does not implement the internal [[Construct]] method, generate a runtime error."
		{"11.2.2 new on an object that is not a constructor is a runtime error", runtimeError(`print("before"); new Math`, "before\n")},
		// 11.2.2 steps 3-4 (MemberExpression Arguments): the arguments are evaluated before the type check
		{"11.2.2 new evaluates the arguments before checking the constructor",
			runtimeError(logJS+`var x = 1; new x(log("arg", 1))`, "arg\n")},

		// 11.2.3 steps 1-2: "Evaluate MemberExpression. Evaluate Arguments" before step 3 "Call GetValue(Result(1))"
		{"11.2.3 the arguments are evaluated before the function is looked up",
			runtimeError(logJS+`undeclaredFunction(log("arg", 1))`, "arg\n")},
		// 11.2.3 step 4: "If Type(Result(3)) is not Object, generate a runtime error."
		{"11.2.3 calling a value that is not an object is a runtime error", runtimeError(logJS+`var n = 1; n(log("arg", 1))`, "arg\n")},
		// 11.2.3 step 5: "If Result(3) does not implement the internal [[Call]] method, generate a runtime error."
		{"11.2.3 calling an object that is not a function is a runtime error",
			runtimeError(logJS+`var o = new Object(); o(log("arg", 1))`, "arg\n")},
		// 11.2.3 step 6: "If Type(Result(1)) is Reference, Result(6) is GetBase(Result(1))." The base becomes this.
		{"11.2.3 a call through a property passes its base object as this",
			output(`function who() { return this.name } var o = new Object(); o.name = "o"; o.who = who; print(o.who(), o["who"]())`, "o o\n")},
		// 11.2.3 step 6: "Otherwise, Result(6) is null." A null this becomes the global object (§10.2.3).
		{"11.2.3 a call of a value that is not a reference gets the global object as this",
			output(`function self() { return this } var o = new Object(); o.self = self; print((0, o.self)() == this, o.self() == o)`, "true true\n")},
		// 11.2.3 step 7: "If Result(6) is an activation object, Result(7) is null."
		{"11.2.3 a call through a local variable gets the global object as this",
			output(`function self() { return this } function f() { var h = self; return h() } print(f() == this)`, "true\n")},
		// 11.2.3 step 7: a function found on an object added by with is called with that object as this
		{"11.2.3 a call through a with object gets that object as this",
			output(`function who() { return this.name } var o = new Object(); o.name = "o"; o.who = who; with (o) print(who())`, "o\n")},
		// 11.2.3: "The production CallExpression : CallExpression Arguments is evaluated in exactly the same manner"
		{"11.2.3 the result of a call can be called", output(`function g() { return "g called" } function f() { return g } print(f()())`, "g called\n")},
		// 11.2.3 step 9: "Return Result(8)."
		{"11.2.3 a call evaluates to the returned value", output(`function f(x) { return x * 2 } print(f(21))`, "42\n")},

		// 11.2.4: "Arguments : ( ) ... Return an empty internal list of values."
		{"11.2.4 empty parentheses pass no arguments", output(`function f() { return arguments.length } print(f())`, "0\n")},
		// 11.2.4: "ArgumentList : ArgumentList , AssignmentExpression ... followed at the end by Result(3)": left to right
		{"11.2.4 arguments are evaluated from left to right",
			output(logJS+`function f(a, b, c) { return a + b + c } print(f(log("a", 1), log("b", 2), log("c", 3)))`, "a\nb\nc\n6\n")},
		// 11.2.4: "Call GetValue(Result(1))." Each argument is a value, not a reference.
		{"11.2.4 an argument is passed by value", output(`function f(a) { a = 2 } var x = 1; f(x); print(x)`, "1\n")},
		// 11.2.4: GetValue of an undeclared argument fails
		{"11.2.4 an undeclared variable as an argument is a runtime error", runtimeError(`function f(a) {} print("before"); f(undeclared)`, "before\n")},
		// 11.2.4: an argument is an AssignmentExpression, so a comma separates arguments
		{"11.2.4 an assignment can be an argument", output(`var x; print(x = 5, x)`, "5 5\n")},
	})
}

func TestS11_3_PostfixExpressions(t *testing.T) {
	runRows(t, []row{
		// 11.3.1 step 6: "Return Result(3)." (the old value converted with ToNumber)
		{"11.3.1 x++ evaluates to the old value as a number", output(`var s = "5"; var r = s++; print(r, typeof r, s, typeof s)`, "5 number 6 number\n")},
		// 11.3.1 step 5: "Call PutValue(Result(1), Result(4))."
		{"11.3.1 x++ stores the incremented value in a property", output(`var o = new Object(); o.p = 1; o.p++; print(o.p)`, "2\n")},
		// 11.3.1 steps 3-4: ToNumber, then add 1 "using the same rules as for the + operator"
		{"11.3.1 x++ on a value that is not a number", output(`var u, n = null, b = true; u++; n++; b++; print(u, n, b)`, "NaN 1 2\n")},
		// 11.3.1 step 2: GetValue of an undeclared variable fails
		{"11.3.1 x++ on an undeclared variable is a runtime error", runtimeError(`print("before"); undeclared++`, "before\n")},
		// 11.3.1 step 5: PutValue on a value that is not a reference is an error (§8.7.4)
		{"11.3.1 ++ after a value that is not a reference is an error", earlyOrRuntimeError(`1++`)},

		// 11.3.2 step 6: "Return Result(3)."
		{"11.3.2 x-- evaluates to the old value as a number", output(`var s = "5"; var r = s--; print(r, typeof r, s)`, "5 number 4\n")},
		// 11.3.2 step 4: "Subtract the value 1 from Result(3)"
		{"11.3.2 x-- stores the decremented value in a property", output(`var o = new Object(); o.p = 1; o.p--; print(o.p)`, "0\n")},
		// 11.3.2 step 5: PutValue on a value that is not a reference is an error
		{"11.3.2 -- after a value that is not a reference is an error", earlyOrRuntimeError(`1--`)},
	})
}

func TestS11_4_UnaryOperators(t *testing.T) {
	runRows(t, []row{
		// 11.4.1 step 4: "If Type(Result(2)) is not Object, return true." (an unresolvable reference has base null)
		{"11.4.1 deleting an undeclared variable returns true", output(`print(delete undeclared)`, "true\n")},
		// 11.4.1 steps 6-7: "Call the [[Delete]] method on Result(2) ... Return Result(6)."
		{"11.4.1 delete removes a property and returns true", output(`var o = new Object(); o.p = 1; print(delete o.p, o.p, delete o["p"])`, "true undefined true\n")},
		// 11.4.1 step 7 with [[Delete]] step 2: a DontDelete property is kept and false returned
		{"11.4.1 delete of a DontDelete property returns false", output(`print(delete Math.LN2, Math.LN2)`, "false 0.6931471805599453\n")},
		// 11.4.1 with §10.2.3: variables and parameters of function code are DontDelete
		{"11.4.1 delete of a local variable or parameter returns false",
			output(`function f(p) { var v = 1; return (delete v) + " " + (delete p) + " " + v + " " + p } print(f(2))`, "false false 1 2\n")},
		// 11.4.1 with §10.2.1: global variables have empty attributes, so delete removes them
		{"11.4.1 delete of a global variable removes it",
			output(`var x = 1; print(delete x, typeof x)`, "true undefined\n")},
		// 11.4.1 with §8.7.4 step 6: a variable created by assignment is a property of the global object with empty attributes
		{"11.4.1 delete of an implicitly created global removes it", output(`y = 2; print(delete y, typeof y)`, "true undefined\n")},
		// 11.4.1 step 2: GetBase of a value that is not a Reference generates a runtime error (§8.7.1)
		{"11.4.1 delete of a value that is not a reference is an error",
			earlyOrRuntimeError(`print(delete "s")`)},

		// 11.4.2 step 3: "Return undefined."
		{"11.4.2 void evaluates to undefined", output(`print(void 0, void "x", typeof void 1)`, "undefined undefined undefined\n")},
		// 11.4.2 step 1: "Evaluate UnaryExpression." Its side effects happen.
		{"11.4.2 void evaluates its operand", output(`var x = 1; void x++; print(x)`, "2\n")},
		// 11.4.2 step 2: "Call GetValue(Result(1))."
		{"11.4.2 void of an undeclared variable is a runtime error", runtimeError(`print("before"); void undeclared`, "before\n")},

		// 11.4.3 step 2: "If Type(Result(1)) is Reference and GetBase(Result(1)) is null, return "undefined"."
		{"11.4.3 typeof an undeclared variable is undefined", output(`print(typeof undeclared)`, "undefined\n")},
		// 11.4.3 table: Undefined "undefined", Null "object", Boolean "boolean", Number "number", String "string"
		{"11.4.3 typeof of each primitive type",
			output(`var u; print(typeof u, typeof null, typeof false, typeof 1.5, typeof "")`, "undefined object boolean number string\n")},
		// 11.4.3 table: "Object (native and doesn't implement [[Call]])" "object"
		{"11.4.3 typeof an object that is not a function is object",
			output(`print(typeof new Object(), typeof new Number(1), typeof new String(""), typeof Math)`, "object object object object\n")},
		// 11.4.3 table: "Object (native and implements [[Call]])" "function"
		{"11.4.3 typeof a function is function",
			output(`function f() {} print(typeof f, typeof Math.abs, typeof Object, typeof new Function("return 1"))`, "function function function function\n")},
		// 11.4.3 step 3: "Call GetValue(Result(1))." A missing property is undefined.
		{"11.4.3 typeof a missing property is undefined", output(`var o = new Object(); print(typeof o.missing)`, "undefined\n")},
		// 11.4.3 step 3: GetValue fails when the base of a property access cannot be resolved
		{"11.4.3 typeof a property of an undeclared variable is a runtime error", runtimeError(`print("before"); typeof undeclared.p`, "before\n")},

		// 11.4.4 step 6: "Return Result(4)." (the new value)
		{"11.4.4 ++x evaluates to the new value as a number", output(`var s = "5"; var r = ++s; print(r, typeof r, s)`, "6 number 6\n")},
		// 11.4.4 step 5: "Call PutValue(Result(1), Result(4))."
		{"11.4.4 ++x stores the incremented value", output(`var o = new Object(); o.p = 1.5; ++o.p; var u; ++u; print(o.p, u)`, "2.5 NaN\n")},
		// 11.4.4 step 5: PutValue on a value that is not a reference is an error
		{"11.4.4 ++ before a value that is not a reference is an error", earlyOrRuntimeError(`++1`)},

		// 11.4.5 step 6: "Return Result(4)."
		{"11.4.5 --x evaluates to the new value as a number", output(`var s = "5"; var r = --s; print(r, typeof r, s)`, "4 number 4\n")},
		// 11.4.5 step 5: PutValue on a value that is not a reference is an error
		{"11.4.5 -- before a value that is not a reference is an error", earlyOrRuntimeError(`--1`)},

		// 11.4.6: "The unary + operator converts its operand to Number type."
		{"11.4.6 unary + converts its operand to a number", output(`print(+"3" + 1, typeof +"3", +"x", +true, +-1)`, "4 number NaN 1 -1\n")},

		// 11.4.7 step 4: "If Result(3) is NaN, return NaN."
		{"11.4.7 negating NaN gives NaN", output(`print(-NaN, -"x")`, "NaN NaN\n")},
		// 11.4.7: "negating +0 produces -0, and negating -0 produces +0"
		{"11.4.7 negating a zero flips its sign", output(`print(1 / -0, 1 / -(-0))`, "-Infinity Infinity\n")},
		// 11.4.7 step 5: "compute a number with the same magnitude but opposite sign"
		{"11.4.7 unary - negates its operand converted to a number", output(`print(-"3", - -1, -Infinity, -true)`, "-3 1 -Infinity -1\n")},

		// 11.4.8 steps 3-4: "Call ToInt32(Result(2)). Apply bitwise complement to Result(3)."
		{"11.4.8 ~ complements the bits of a 32-bit integer", output(`print(~0, ~-1, ~5, ~2147483647)`, "-1 0 -6 -2147483648\n")},
		// 11.4.8 step 3: the operand is converted with ToInt32
		{"11.4.8 ~ converts its operand with ToInt32", output(`print(~4294967295, ~1.9, ~NaN, ~"7")`, "0 -2 -1 -8\n")},

		// 11.4.9 steps 3-5: "Call ToBoolean ... If Result(3) is true, return false. Return true."
		{"11.4.9 ! negates the boolean value of its operand", output(`print(!true, !false, !0, !1, !"", !"a", !null, !new Object())`, "false true true false true false true false\n")},
	})
}

func TestS11_5_MultiplicativeOperators(t *testing.T) {
	runRows(t, []row{
		// 11.5 steps 1-6: both operands are evaluated, then converted with ToNumber, left first
		{"11.5 both operands are evaluated before either is converted",
			output(logJS+objJS+`print(log("x", obj("a", 2)) * log("y", obj("b", 3)))`, "x\ny\na\nb\n6\n")},
		// 11.5 steps 5-6: "Call ToNumber"
		{"11.5 operands are converted to numbers", output(`print("3" * "4", true * 5, null * 5, "x" * 1)`, "12 5 0 NaN\n")},

		// 11.5.1: "If either operand is NaN, the result is NaN."
		{"11.5.1 a NaN operand gives NaN", output(`print(NaN * 1, 1 * NaN)`, "NaN NaN\n")},
		// 11.5.1: "The sign of the result is positive if both operands have the same sign, negative if the operands have different signs."
		{"11.5.1 the sign of a product", output(`print(-2 * 3, -2 * -3, 1 / (0 * -1), 1 / (-0 * -1))`, "-6 6 -Infinity Infinity\n")},
		// 11.5.1: "Multiplication of an infinity by a zero results in NaN."
		{"11.5.1 infinity times zero is NaN", output(`print(Infinity * 0, -0 * Infinity)`, "NaN NaN\n")},
		// 11.5.1: "Multiplication of an infinity by an infinity results in an infinity."
		{"11.5.1 infinity times infinity is a signed infinity", output(`print(Infinity * Infinity, Infinity * -Infinity)`, "Infinity -Infinity\n")},
		// 11.5.1: "Multiplication of an infinity by a finite non-zero value results in a signed infinity."
		{"11.5.1 infinity times a finite number is a signed infinity", output(`print(Infinity * 2, Infinity * -0.5)`, "Infinity -Infinity\n")},
		// 11.5.1: "the product is computed and rounded to the nearest representable value"
		{"11.5.1 a product is rounded to the nearest number", output(`print(1.5 * 4, 3 * 0.1 == 0.30000000000000004)`, "6 true\n")},
		// 11.5.1: "If the magnitude is too large to represent, the result is then an infinity of appropriate sign."
		{"11.5.1 an overflowing product is infinity", output(`print(1e308 * 10, -1e308 * 10)`, "Infinity -Infinity\n")},
		// 11.5.1: "If the magnitude is too small to represent, the result is then a zero of appropriate sign."
		{"11.5.1 an underflowing product is a signed zero", output(`print(1e-308 * 1e-308, 1 / (-1e-308 * 1e-308))`, "0 -Infinity\n")},
		// 11.5.1: "The ECMAScript language requires support of gradual underflow as defined by IEEE 754."
		{"11.5.1 products underflow gradually", output(`var tiny = 2.2250738585072014e-308 * 0.5; print(tiny > 0, tiny * 2 == 2.2250738585072014e-308)`, "true true\n")},

		// 11.5.2: "If either operand is NaN, the result is NaN."
		{"11.5.2 a NaN operand gives NaN", output(`print(NaN / 1, 1 / NaN)`, "NaN NaN\n")},
		// 11.5.2: "ECMAScript does not perform integer division."
		{"11.5.2 division is not integer division", output(`print(7 / 2, -7 / 2)`, "3.5 -3.5\n")},
		// 11.5.2: "Division of an infinity by an infinity results in NaN."
		{"11.5.2 infinity divided by infinity is NaN", output(`print(Infinity / Infinity, -Infinity / Infinity)`, "NaN NaN\n")},
		// 11.5.2: "Division of an infinity by a zero results in an infinity."
		{"11.5.2 infinity divided by zero is a signed infinity", output(`print(Infinity / 0, Infinity / -0, -Infinity / 0)`, "Infinity -Infinity -Infinity\n")},
		// 11.5.2: "Division of an infinity by a non-zero finite value results in a signed infinity."
		{"11.5.2 infinity divided by a finite number is a signed infinity", output(`print(Infinity / 2, -Infinity / 2, Infinity / -2)`, "Infinity -Infinity -Infinity\n")},
		// 11.5.2: "Division of a finite value by an infinity results in zero."
		{"11.5.2 a finite number divided by infinity is a signed zero", output(`print(1 / (1 / Infinity), 1 / (-1 / Infinity), 1 / (1 / -Infinity))`, "Infinity -Infinity -Infinity\n")},
		// 11.5.2: "Division of a zero by a zero results in NaN"
		{"11.5.2 zero divided by zero is NaN", output(`print(0 / 0, -0 / 0)`, "NaN NaN\n")},
		// 11.5.2: "division of zero by any other finite value results in zero, with the sign determined by the rule"
		{"11.5.2 zero divided by a finite number is a signed zero", output(`print(1 / (0 / 5), 1 / (-0 / 5), 1 / (0 / -5))`, "Infinity -Infinity -Infinity\n")},
		// 11.5.2: "Division of a non-zero finite value by a zero results in a signed infinity."
		{"11.5.2 a finite number divided by zero is a signed infinity", output(`print(1 / 0, -1 / 0, 1 / -0, -1 / -0)`, "Infinity -Infinity -Infinity Infinity\n")},
		// 11.5.2: "If the magnitude is too large to represent, we say the operation overflows"
		{"11.5.2 an overflowing quotient is infinity", output(`print(1e308 / 1e-10, -1e308 / 1e-10)`, "Infinity -Infinity\n")},
		// 11.5.2: "If the magnitude is too small to represent, we say the operation underflows and the result is a zero"
		{"11.5.2 an underflowing quotient is a signed zero", output(`print(1e-308 / 1e300, 1 / (-1e-308 / 1e300))`, "0 -Infinity\n")},

		// 11.5.3: "If either operand is NaN, the result is NaN."
		{"11.5.3 a NaN operand gives NaN", output(`print(NaN % 1, 1 % NaN)`, "NaN NaN\n")},
		// 11.5.3: "The sign of the result equals the sign of the dividend."
		{"11.5.3 the remainder has the sign of the dividend", output(`print(7 % 2, -7 % 2, 7 % -2, -7 % -2)`, "1 -1 1 -1\n")},
		// 11.5.3: "If the dividend is an infinity, or the divisor is a zero, or both, the result is NaN."
		{"11.5.3 an infinite dividend or a zero divisor gives NaN", output(`print(Infinity % 2, 5 % 0, Infinity % 0, -Infinity % Infinity)`, "NaN NaN NaN NaN\n")},
		// 11.5.3: "If the dividend is finite and the divisor is an infinity, the result equals the dividend."
		{"11.5.3 a finite dividend modulo infinity is the dividend", output(`print(5 % Infinity, -5 % -Infinity, 1 / (-0 % Infinity))`, "5 -5 -Infinity\n")},
		// 11.5.3: "If the dividend is a zero and the divisor is finite, the result is the same as the dividend."
		{"11.5.3 a zero dividend gives that zero", output(`print(1 / (0 % 5), 1 / (-0 % 5), 1 / (-0 % -5))`, "Infinity -Infinity -Infinity\n")},
		// 11.5.3: "r = n - (d * q) where q is an integer ... whose magnitude is as large as possible without exceeding the magnitude of the true mathematical quotient"
		{"11.5.3 the remainder comes from a truncating division", output(`print(7 % 4, 5.5 % 2, -5.5 % 2, 6.75 % 0.5)`, "3 1.5 -1.5 0.25\n")},
	})
}

func TestS11_6_AdditiveOperators(t *testing.T) {
	runRows(t, []row{
		// 11.6.1 steps 1-6: both operands are evaluated, then converted with ToPrimitive, left first
		{"11.6.1 both operands are evaluated before either is converted",
			output(logJS+objJS+`print(log("x", obj("a", 1)) + log("y", obj("b", 2)))`, "x\ny\na\nb\n3\n")},
		// 11.6.1 step 7: "If Type(Result(5)) is String or Type(Result(6)) is String, go to step 12."
		{"11.6.1 + concatenates when either operand is a string", output(`print(1 + "2", "1" + 2, "a" + null, true + "", "" + void 0)`, "12 12 anull true undefined\n")},
		// 11.6.1 steps 8-10: otherwise "Call ToNumber" on both and add
		{"11.6.1 + adds numbers when neither operand is a string", output(`print(1 + 2, true + 1, null + 1, void 0 + 1, true + true)`, "3 2 1 NaN 2\n")},
		// 11.6.1 step 5: ToPrimitive with no hint uses valueOf for an ordinary object
		{"11.6.1 an object is converted with no hint, through valueOf",
			output(`function vo() { return 5 } function ts() { return "str" } var o = new Object(); o.valueOf = vo; o.toString = ts; print(o + 1, o + "!")`, "6 5!\n")},
		// 11.6.1 step 7: the type check comes after ToPrimitive, so an object whose primitive is a string concatenates
		{"11.6.1 an object whose primitive value is a string concatenates",
			output(`function vo() { return "s" } var o = new Object(); o.valueOf = vo; print(o + 1)`, "s1\n")},
		// 11.6.1: "Date objects handle the absence of a hint as if the hint String were given"
		{"11.6.1 a Date is converted with toString", output(`var d = new Date(0); print(typeof (d + 1), d + 1 == d.toString() + "1")`, "string true\n")},
		// 11.6.1 steps 12-14: "Concatenate Result(12) followed by Result(13)."
		{"11.6.1 concatenation keeps the order of the operands", output(`print("ab" + "cd", "" + "", ("x" + "").length)`, "abcd  1\n")},

		// 11.6.2 steps 5-7: "Call ToNumber" on both and subtract
		{"11.6.2 - converts both operands to numbers", output(`print("5" - 2, "a" - 1, true - false, 5 - "")`, "3 NaN 1 5\n")},
		// 11.6.2 steps 1-6: both operands are evaluated before either is converted
		{"11.6.2 both operands are evaluated before either is converted",
			output(logJS+objJS+`print(log("x", obj("a", 5)) - log("y", obj("b", 2)))`, "x\ny\na\nb\n3\n")},

		// 11.6.3: "If either operand is NaN, the result is NaN."
		{"11.6.3 a NaN operand gives NaN", output(`print(NaN + 1, 1 - NaN)`, "NaN NaN\n")},
		// 11.6.3: "The sum of two infinities of opposite sign is NaN."
		{"11.6.3 infinities of opposite sign sum to NaN", output(`print(Infinity + -Infinity, Infinity - Infinity)`, "NaN NaN\n")},
		// 11.6.3: "The sum of two infinities of the same sign is the infinity of that sign."
		{"11.6.3 infinities of the same sign sum to that infinity", output(`print(Infinity + Infinity, -Infinity + -Infinity)`, "Infinity -Infinity\n")},
		// 11.6.3: "The sum of an infinity and a finite value is equal to the infinite operand."
		{"11.6.3 infinity plus a finite number is that infinity", output(`print(Infinity + 1, -Infinity + 1e308)`, "Infinity -Infinity\n")},
		// 11.6.3: "The sum of two negative zeros is -0. The sum of two positive zeros, or of two zeros of opposite sign, is +0."
		{"11.6.3 the sign of a sum of zeros", output(`print(1 / (-0 + -0), 1 / (0 + 0), 1 / (0 + -0), 1 / (-0 + 0))`, "-Infinity Infinity Infinity Infinity\n")},
		// 11.6.3: "The sum of a zero and a nonzero finite value is equal to the nonzero operand."
		{"11.6.3 zero plus a number is that number", output(`print(0 + 5, -0 + -5)`, "5 -5\n")},
		// 11.6.3: "The sum of two nonzero finite values of the same magnitude and opposite sign is +0."
		{"11.6.3 a number plus its negative is +0", output(`print(1 / (5 + -5), 1 / (-5 + 5))`, "Infinity Infinity\n")},
		// 11.6.3: "the sum is computed and rounded to the nearest representable value"
		{"11.6.3 a sum is rounded to the nearest number", output(`print(0.1 + 0.2, 9007199254740992 + 1)`, "0.30000000000000004 9007199254740992\n")},
		// 11.6.3: "If the magnitude is too large to represent, the operation overflows"
		{"11.6.3 an overflowing sum is infinity", output(`print(1e308 + 1e308, -1e308 - 1e308)`, "Infinity -Infinity\n")},
		// 11.6.3: "a-b produces the same result as a+(-b)"
		{"11.6.3 subtraction is addition of the negation", output(`print(1 / (-0 - 0), 1 / (0 - 0), 1 / (-0 - -0), 5 - 3)`, "-Infinity Infinity Infinity 2\n")},
	})
}

func TestS11_7_BitwiseShiftOperators(t *testing.T) {
	runRows(t, []row{
		// 11.7: "The result of evaluating an AdditiveExpresion that is the right-hand operand of a shift operator is always truncated to five bits."
		{"11.7 the shift count is taken modulo 32", output(`print(1 << 32, 1 << 33, 16 >> 34, 1 >>> 32)`, "1 2 4 1\n")},
		// 11.7: "If the result of evaluating ShiftExpression produces a fractional component, the fractional component is discarded."
		{"11.7 fractions are discarded", output(`print(5.9 << 1.9, -5.9 >> 0.5)`, "10 -5\n")},
		// 11.7.x steps 1-6: both operands are evaluated, then converted left first
		{"11.7 both operands are evaluated before either is converted",
			output(logJS+objJS+`print(log("x", obj("a", 1)) << log("y", obj("b", 2)))`, "x\ny\na\nb\n4\n")},

		// 11.7.1 step 8: "Left shift Result(5) by Result(7) bits. The result is a signed 32 bit integer."
		{"11.7.1 << shifts left and gives a signed 32-bit result", output(`print(1 << 3, 1 << 31, 3 << 31, 1073741824 << 2)`, "8 -2147483648 -2147483648 0\n")},
		// 11.7.1 steps 5-7: ToInt32 of the left operand, ToUint32 of the right masked to 5 bits
		{"11.7.1 << converts its operands to integers", output(`print(4294967297 << 1, 1 << -1, "2" << "1")`, "2 -2147483648 4\n")},

		// 11.7.2 step 8: "Perform sign-extending right shift ... The most significant bit is propagated."
		{"11.7.2 >> fills with the sign bit", output(`print(-16 >> 2, -1 >> 31, 16 >> 2, -7 >> 1)`, "-4 -1 4 -4\n")},
		// 11.7.2 steps 6-7: "Call ToUint32(Result(4))." "compute Result(6) & 0x1F."
		{"11.7.2 >> masks a negative or large shift count to 5 bits", output(`print(-16 >> -30, -16 >> 33, 5 >> 32)`, "-4 -8 5\n")},
		// 11.7.2 step 5: "Call ToInt32(Result(2))."
		{"11.7.2 >> treats its left operand as signed", output(`print(2147483648 >> 0, 4294967295 >> 1)`, "-2147483648 -1\n")},

		// 11.7.3 step 8: "Perform zero-filling right shift ... The result is an unsigned 32 bit integer."
		{"11.7.3 >>> fills with zeros", output(`print(-1 >>> 28, -16 >>> 2, 16 >>> 2)`, "15 1073741820 4\n")},
		// 11.7.3 steps 6-7: "Call ToUint32(Result(4))." "compute Result(6) & 0x1F."
		{"11.7.3 >>> masks a negative or large shift count to 5 bits", output(`print(-1 >>> -4, -1 >>> 32, -16 >>> 34)`, "15 4294967295 1073741820\n")},
		// 11.7.3 step 5: "Call ToUint32(Result(2))."
		{"11.7.3 >>> treats its left operand as unsigned", output(`print(-1 >>> 0, 4294967296 >>> 0, -2147483648 >>> 31)`, "4294967295 0 1\n")},
	})
}

func TestS11_8_RelationalOperators(t *testing.T) {
	runRows(t, []row{
		// 11.8: "The result of evaluating RelationalExpression is always of type Boolean"
		{"11.8 a comparison evaluates to a boolean", output(`print(typeof (1 < 2), typeof (NaN < 1), typeof ("a" >= "b"))`, "boolean boolean boolean\n")},

		// 11.8.1 step 5: "Perform the comparison Result(2) < Result(4)."
		{"11.8.1 < compares its operands", output(`print(1 < 2, 2 < 1, 1 < 1)`, "true false false\n")},
		// 11.8.1 step 6: "If Result(5) is undefined, return false."
		{"11.8.1 < is false when an operand is NaN", output(`print(NaN < 1, 1 < NaN)`, "false false\n")},
		// 11.8.1 steps 1-5: the left operand is converted first (x is Result(2) in §11.8.5 step 1)
		{"11.8.1 < converts its left operand first", output(objJS+`print(obj("a", 1) < obj("b", 2))`, "a\nb\ntrue\n")},

		// 11.8.2 step 5: "Perform the comparison Result(4) < Result(2)."
		{"11.8.2 > compares its operands", output(`print(2 > 1, 1 > 2, 1 > 1)`, "true false false\n")},
		// 11.8.2 step 6: "If Result(5) is undefined, return false."
		{"11.8.2 > is false when an operand is NaN", output(`print(NaN > 1, 1 > NaN)`, "false false\n")},
		// 11.8.2 step 5: the comparison is Result(4) < Result(2), so the right operand is converted first
		{"11.8.2 > converts its right operand first", output(objJS+`print(obj("a", 1) > obj("b", 2))`, "b\na\nfalse\n")},

		// 11.8.3 step 6: "If Result(5) is true or undefined, return false. Otherwise, return true."
		{"11.8.3 <= compares its operands", output(`print(1 <= 2, 2 <= 1, 1 <= 1)`, "true false true\n")},
		// 11.8.3 step 6: undefined (a NaN operand) gives false
		{"11.8.3 <= is false when an operand is NaN", output(`print(NaN <= 1, 1 <= NaN, NaN <= NaN)`, "false false false\n")},
		// 11.8.3 step 5: "Perform the comparison Result(4) < Result(2)." The right operand is converted first.
		{"11.8.3 <= converts its right operand first", output(objJS+`print(obj("a", 1) <= obj("b", 2))`, "b\na\ntrue\n")},

		// 11.8.4 step 6: "If Result(5) is true or undefined, return false. Otherwise, return true."
		{"11.8.4 >= compares its operands", output(`print(2 >= 1, 1 >= 2, 1 >= 1)`, "true false true\n")},
		// 11.8.4 step 6: undefined (a NaN operand) gives false
		{"11.8.4 >= is false when an operand is NaN", output(`print(NaN >= 1, 1 >= NaN)`, "false false\n")},
		// 11.8.4 step 5: "Perform the comparison Result(2) < Result(4)." The left operand is converted first.
		{"11.8.4 >= converts its left operand first", output(objJS+`print(obj("a", 1) >= obj("b", 2))`, "a\nb\nfalse\n")},

		// 11.8.5 steps 1-2: "Call ToPrimitive(x, hint Number)." valueOf is preferred over toString.
		{"11.8.5 objects are compared through their valueOf",
			output(`function vo() { return 1 } function ts() { return "5" } var o = new Object(); o.valueOf = vo; o.toString = ts; print(o < 2, o > 0)`, "true true\n")},
		// 11.8.5 steps 1-2: hint Number applies to a Date too, unlike + (§11.6.1)
		{"11.8.5 dates are compared as numbers", output(`print(new Date(5) < new Date(6), new Date(6) < new Date(5))`, "true false\n")},
		// 11.8.5 step 3: "If Type(Result(1)) is String and Type(Result(2)) is String, go to step 16."
		{"11.8.5 two strings are compared as strings", output(`print("10" < "9", "abc" < "abd")`, "true true\n")},
		// 11.8.5 steps 3-5: if one operand is not a string, both are compared as numbers
		{"11.8.5 a string and a number are compared as numbers", output(`print("10" < 9, 10 < "9", "2" < 10)`, "false false true\n")},
		// 11.8.5 steps 6-7: "If Result(4) is NaN, return undefined." "If Result(5) is NaN, return undefined."
		{"11.8.5 a NaN operand makes < false", output(`print(NaN < 1, 1 < NaN, "a" < 1, NaN < NaN)`, "false false false false\n")},
		// 11.8.5 step 8: "If Result(4) and Result(5) are the same number value, return false."
		{"11.8.5 equal numbers are not less than each other", output(`print(1 < 1, -5 < -5)`, "false false\n")},
		// 11.8.5 steps 9-10: "If Result(4) is +0 and Result(5) is -0, return false." and the reverse
		{"11.8.5 the two zeros are not less than each other", output(`print(0 < -0, -0 < 0)`, "false false\n")},
		// 11.8.5 step 11: "If Result(4) is +Infinity, return false."
		{"11.8.5 +Infinity is not less than anything", output(`print(Infinity < Infinity, Infinity < 1)`, "false false\n")},
		// 11.8.5 step 12: "If Result(5) is +Infinity, return true."
		{"11.8.5 every other number is less than +Infinity", output(`print(1 < Infinity, -Infinity < Infinity, 1e308 < Infinity)`, "true true true\n")},
		// 11.8.5 step 13: "If Result(5) is -Infinity, return false."
		{"11.8.5 nothing is less than -Infinity", output(`print(1 < -Infinity, -1e308 < -Infinity)`, "false false\n")},
		// 11.8.5 step 14: "If Result(4) is -Infinity, return true."
		{"11.8.5 -Infinity is less than every finite number", output(`print(-Infinity < 1, -Infinity < -1e308)`, "true true\n")},
		// 11.8.5 step 15: "If the mathematical value of Result(4) is less than the mathematical value of Result(5) ... return true. Otherwise, return false."
		{"11.8.5 finite numbers are compared by value", output(`print(1 < 2, 2 < 1, -2 < -1, 1e-300 < 2e-300, 0 < 5e-324)`, "true false true true true\n")},
		// 11.8.5 step 16: "If Result(2) is a prefix of Result(1), return false."
		{"11.8.5 a string is not less than its prefix", output(`print("abc" < "ab", "a" < "a", "a" < "")`, "false false false\n")},
		// 11.8.5 step 17: "If Result(1) is a prefix of Result(2), return true."
		{"11.8.5 a proper prefix is less than the string", output(`print("ab" < "abc", "" < "a")`, "true true\n")},
		// 11.8.5 steps 18-21: the first differing character decides, by its Unicode value
		{"11.8.5 strings are ordered by the first differing character code",
			output(`print("abd" < "abc", "B" < "a", "a" < "B", "Z" < "a", "\u00e9" < "z", "abc" < "b")`, "false true false true false true\n")},
	})
}

func TestS11_9_EqualityOperators(t *testing.T) {
	runRows(t, []row{
		// 11.9: "The result of evaluating EqualityExpression is always of type Boolean"
		{"11.9 an equality evaluates to a boolean", output(`print(typeof (1 == 1), typeof ("a" != null))`, "boolean boolean\n")},

		// 11.9.1 steps 1-4: both operands are evaluated, left first
		{"11.9.1 == evaluates the left operand first", output(logJS+`print(log("a", 1) == log("b", 1))`, "a\nb\ntrue\n")},
		// 11.9.1 step 6: "Return Result(5)."
		{"11.9.1 == gives the result of the comparison", output(`print(1 == 1, 1 == 2)`, "true false\n")},

		// 11.9.2 step 6: "If Result(5) is true, return false. Otherwise, return true."
		{"11.9.2 != negates the comparison", output(`print(1 != 2, 1 != 1, NaN != NaN, null != undefined)`, "true false true false\n")},

		// 11.9.3 step 2: "If Type(x) is Undefined, return true."
		{"11.9.3 undefined equals undefined", output(`var a, b; print(a == b)`, "true\n")},
		// 11.9.3 step 3: "If Type(x) is Null, return true."
		{"11.9.3 null equals null", output(`print(null == null)`, "true\n")},
		// 11.9.3 step 5: "If x is NaN, return false."
		{"11.9.3 NaN on the left is not equal to anything", output(`print(NaN == NaN, NaN == 1)`, "false false\n")},
		// 11.9.3 step 6: "If y is NaN, return false."
		{"11.9.3 NaN on the right is not equal to anything", output(`print(1 == NaN, Infinity == NaN)`, "false false\n")},
		// 11.9.3 step 7: "If x is the same number value as y, return true."
		{"11.9.3 equal numbers are equal", output(`print(1 == 1, 1.5 == 1.5, Infinity == Infinity)`, "true true true\n")},
		// 11.9.3 steps 8-9: "If x is +0 and y is -0, return true." and the reverse
		{"11.9.3 the two zeros are equal", output(`print(0 == -0, -0 == 0)`, "true true\n")},
		// 11.9.3 step 10: "Return false."
		{"11.9.3 different numbers are not equal", output(`print(1 == 2, Infinity == -Infinity)`, "false false\n")},
		// 11.9.3 step 11: "return true if x and y are exactly the same sequence of characters"
		{"11.9.3 strings are equal when their characters are", output(`print("ab" == "ab", "ab" == "abc", "a" == "A", "" == "")`, "true false false true\n")},
		// 11.9.3 step 12: "return true if x and y are both true or both false"
		{"11.9.3 booleans are equal when both are true or both false", output(`print(true == true, false == false, true == false)`, "true true false\n")},
		// 11.9.3 step 13: "Return true if x and y refer to the same object."
		{"11.9.3 objects are equal only to themselves",
			output(`var o = new Object(), p = o; print(o == p, new Object() == new Object(), new String("a") == new String("a"))`, "true false false\n")},
		// 11.9.3 step 14: "If x is null and y is undefined, return true."
		{"11.9.3 null equals undefined", output(`print(null == void 0)`, "true\n")},
		// 11.9.3 step 15: "If x is undefined and y is null, return true."
		{"11.9.3 undefined equals null", output(`print(void 0 == null)`, "true\n")},
		// 11.9.3 step 16: "If Type(x) is Number and Type(y) is String, return the result of the comparison x == ToNumber(y)."
		{"11.9.3 a number and a string are compared as numbers", output(`print(1 == "1", 0 == "", 1 == "x", 255 == "0xff", 1 == " 1 ")`, "true true false true true\n")},
		// 11.9.3 step 17: "If Type(x) is String and Type(y) is Number, return the result of the comparison ToNumber(x) == y."
		{"11.9.3 a string and a number are compared as numbers", output(`print("1.0" == 1, "x" == NaN, "" == 0)`, "true false true\n")},
		// 11.9.3 step 18: "If Type(x) is Boolean, return the result of the comparison ToNumber(x) == y."
		{"11.9.3 a boolean on the left is compared as a number", output(`print(true == 1, false == "0", true == "true", true == 2)`, "true true false false\n")},
		// 11.9.3 step 19: "If Type(y) is Boolean, return the result of the comparison x == ToNumber(y)."
		{"11.9.3 a boolean on the right is compared as a number", output(`print(1 == true, 2 == true, "1" == true, "" == false)`, "true false true true\n")},
		// 11.9.3 step 20: "If Type(x) is either String or Number and Type(y) is Object, return the result of the comparison x == ToPrimitive(y)."
		{"11.9.3 a primitive is compared with the primitive value of an object", output(`print(1 == new Number(1), "a" == new String("a"), 2 == new Number(1))`, "true true false\n")},
		// 11.9.3 step 21: "If Type(x) is Object and Type(y) is either String or Number, return the result of the comparison ToPrimitive(x) == y."
		{"11.9.3 an object is compared through its primitive value",
			output(`function vo() { return 3 } var o = new Object(); o.valueOf = vo; print(o == 3, o == "3", new String("b") == "b")`, "true true true\n")},
		// 11.9.3 step 22: "Return false."
		{"11.9.3 null and undefined equal nothing but each other",
			output(`print(null == 0, void 0 == 0, null == false, void 0 == "", null == "null", new Object() == null)`, "false false false false false false\n")},
		// 11.9.3: "the equality operator is not always transitive"
		{"11.9.3 two String objects can each equal a string but not each other",
			output(`var a = new String("x"), b = new String("x"); print(a == "x", b == "x", a == b)`, "true true false\n")},
	})
}

func TestS11_10_BinaryBitwiseOperators(t *testing.T) {
	runRows(t, []row{
		// 11.10 step 7: "Apply the bitwise operator @ to Result(5) and Result(6)."
		{"11.10 &, ^ and | combine the bits of their operands", output(`print(6 & 3, 6 ^ 3, 6 | 3, 0xF0 & 0x3C, 0xF0 ^ 0x3C, 0xF0 | 0x3C)`, "2 5 7 48 204 252\n")},
		// 11.10 steps 5-6: "Call ToInt32(Result(2)). Call ToInt32(Result(4))."
		{"11.10 the operands are converted with ToInt32", output(`print(4294967295 & 1, 1.9 | 0.9, "12" & "10", NaN | 5, 4294967296 | 3)`, "1 1 8 5 3\n")},
		// 11.10 step 7: "The result is a signed 32 bit integer."
		{"11.10 the result is a signed 32-bit integer", output(`print(2147483648 | 0, -1 & -1, 0x80000000 ^ 0)`, "-2147483648 -1 -2147483648\n")},
		// 11.10 steps 1-6: both operands are evaluated, then converted left first
		{"11.10 both operands are evaluated before either is converted",
			output(logJS+objJS+`print(log("x", obj("a", 6)) & log("y", obj("b", 3)))`, "x\ny\na\nb\n2\n")},
		// 11.10 grammar: & binds tighter than ^, which binds tighter than |
		{"11.10 & binds tighter than ^, and ^ tighter than |", output(`print(1 | 2 ^ 3 & 5)`, "3\n")},
	})
}

func TestS11_11_BinaryLogicalOperators(t *testing.T) {
	runRows(t, []row{
		// 11.11 && step 4: "If Result(3) is false, return Result(2)."
		{"11.11 && returns a false left operand without evaluating the right one",
			output(logJS+`print(log("a", 0) && log("b", 1))`, "a\n0\n")},
		// 11.11 && steps 5-7: "Evaluate BitwiseORExpression. Call GetValue(Result(5)). Return Result(6)."
		{"11.11 && returns the right operand when the left one is true",
			output(logJS+`print(log("a", 1) && log("b", "right"))`, "a\nb\nright\n")},
		// 11.11 || step 4: "If Result(3) is true, return Result(2)."
		{"11.11 || returns a true left operand without evaluating the right one",
			output(logJS+`print(log("a", "left") || log("b", 1))`, "a\nleft\n")},
		// 11.11 || steps 5-7: the right operand's value is returned
		{"11.11 || returns the right operand when the left one is false",
			output(logJS+`print(log("a", "") || log("b", 0))`, "a\nb\n0\n")},
		// 11.11: "The value produced will always be the value of one of the two operand expressions."
		{"11.11 && and || produce operand values, not booleans", output(`print(null && 1, "" || "x", 2 && 3, 0 || null)`, "null x 3 null\n")},
		// 11.11 grammar: && binds tighter than ||
		{"11.11 && binds tighter than ||", output(`print(1 || 0 && 0, 0 && 1 || 2)`, "1 2\n")},
	})
}

func TestS11_12_ConditionalOperator(t *testing.T) {
	runRows(t, []row{
		// 11.12 steps 4-7: a true condition evaluates only the first AssignmentExpression
		{"11.12 a true condition evaluates only the first branch",
			output(logJS+`print(log("c", 1) ? log("t", "T") : log("f", "F"))`, "c\nt\nT\n")},
		// 11.12 steps 8-10: a false condition evaluates only the second AssignmentExpression
		{"11.12 a false condition evaluates only the second branch",
			output(logJS+`print(log("c", 0) ? log("t", "T") : log("f", "F"))`, "c\nf\nF\n")},
		// 11.12 step 3: "Call ToBoolean(Result(2))."
		{"11.12 the condition is converted with ToBoolean", output(`print("" ? 1 : 2, "0" ? 1 : 2, new Boolean(false) ? 1 : 2)`, "2 1 1\n")},
		// 11.12: "to allow an assignment expression to be governed by either arm of a conditional"
		{"11.12 either branch may be an assignment", output(`var x; true ? x = 1 : x = 2; var y; false ? y = 1 : y = 2; print(x, y)`, "1 2\n")},
		// 11.12: "eliminate the confusing and fairly useless case of a comma expression as the center expression"
		{"11.12 the middle branch cannot be a comma expression", syntaxError(`var x = true ? 1, 2 : 3;`)},
		// 11.12 grammar: the conditional operator nests to the right
		{"11.12 conditional operators nest to the right", output(`print(false ? 1 : true ? 2 : 3, true ? false ? 1 : 2 : 3)`, "2 2\n")},
	})
}

func TestS11_13_AssignmentOperators(t *testing.T) {
	runRows(t, []row{
		// 11.13.1 step 5: "Return Result(3)."
		{"11.13.1 an assignment evaluates to the assigned value", output(`var x; print(x = 5, x)`, "5 5\n")},
		// 11.13.1 step 4: PutValue; assignment is right associative
		{"11.13.1 chained assignments assign from the right", output(`var a, b, c; a = b = c = "v"; print(a, b, c)`, "v v v\n")},
		// 11.13.1 steps 1-2: "Evaluate LeftHandSideExpression. Evaluate AssignmentExpression."
		{"11.13.1 the left side is evaluated before the right side",
			output(logJS+`var o = new Object(); o[log("key", "p")] = log("value", 1); print(o.p)`, "key\nvalue\n1\n")},
		// 11.13.1 step 1: evaluating a property reference on an undeclared base fails before the right side runs
		{"11.13.1 a failing left side stops the assignment before the right side",
			runtimeError(logJS+`print("before"); undeclared.p = log("value", 1)`, "before\n")},
		// 11.13.1 step 4: PutValue of a value that is not a reference is an error (§8.7.4)
		{"11.13.1 assigning to a literal is an error", earlyOrRuntimeError(`"s" = 1`)},

		// 11.13.2 steps 5-7: "Apply operator @ to Result(2) and Result(4). Call PutValue. Return Result(5)."
		{"11.13.2 += adds or concatenates and stores", output(`var n = 1, s = "a"; print(n += 2, s += 1, n, s)`, "3 a1 3 a1\n")},
		// 11.13.2: -= *= /= %=
		{"11.13.2 -=, *=, /= and %= apply their operator",
			output(`var a = 10, b = 10, c = 10, d = 10; a -= 3; b *= 3; c /= 4; d %= 4; print(a, b, c, d)`, "7 30 2.5 2\n")},
		// 11.13.2: <<= >>= >>>=
		{"11.13.2 <<=, >>= and >>>= apply their operator",
			output(`var a = 1, b = -16, c = -16; a <<= 4; b >>= 2; c >>>= 28; print(a, b, c)`, "16 -4 15\n")},
		// 11.13.2: &= ^= |=
		{"11.13.2 &=, ^= and |= apply their operator",
			output(`var a = 6, b = 6, c = 6; a &= 3; b ^= 3; c |= 3; print(a, b, c)`, "2 5 7\n")},
		// 11.13.2 steps 2-3: "Call GetValue(Result(1))" happens before "Evaluate AssignmentExpression"
		{"11.13.2 the left value is read before the right side is evaluated",
			output(`var x = 1; x += (x = 10, 5); print(x)`, "6\n")},
		// 11.13.2 steps 1-3: the left reference is evaluated before the right side
		{"11.13.2 the left side is evaluated before the right side",
			output(logJS+`var o = new Object(); o.p = 1; o[log("key", "p")] += log("value", 2); print(o.p)`, "key\nvalue\n3\n")},
		// 11.13.2 step 2: GetValue of an undeclared variable fails
		{"11.13.2 a compound assignment to an undeclared variable is a runtime error", runtimeError(`print("before"); undeclared += 1`, "before\n")},
		// 11.13.2 step 6: PutValue of a value that is not a reference is an error
		{"11.13.2 a compound assignment to a literal is an error", earlyOrRuntimeError(`1 += 2`)},
	})
}

func TestS11_14_CommaOperator(t *testing.T) {
	runRows(t, []row{
		// 11.14 steps 1-5: both operands are evaluated left to right and the right value is returned
		{"11.14 the comma operator evaluates both operands and returns the right one", output(logJS+`print((log("a", 1), log("b", 2)))`, "a\nb\n2\n")},
		// 11.14 step 4: GetValue of the right operand, so the result is not a reference
		{"11.14 the comma operator returns a value, not a reference", output(`var x = (1, 2); print(x)`, "2\n")},
		// 11.14 step 2: "Call GetValue(Result(1))." even though the value is not used
		{"11.14 the left operand is read even though its value is unused", runtimeError(`print("before"); var x = (undeclared, 1)`, "before\n")},
	})
}
