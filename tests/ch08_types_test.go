package tests

import "testing"

// Chapter 8, Types. The internal mechanisms (§8.6.2, §8.7, §8.8, §8.9) have
// no entry point of their own and are observed through the operators and
// statements that use them.

func TestS8_1_UndefinedType(t *testing.T) {
	runRows(t, []row{
		// 8.1: "Any variable that has not been assigned a value is of type Undefined."
		{"8.1 a declared variable without a value is undefined", output(`var x; print(x, typeof x)`, "undefined undefined\n")},
		// 8.1: "The Undefined type has exactly one value, called undefined."
		{"8.1 every undefined value is the same value", output(`var a, b; print(a == b, a == void 0)`, "true true\n")},
	})
}

func TestS8_2_NullType(t *testing.T) {
	runRows(t, []row{
		// 8.2: "The Null type has exactly one value, called null."
		{"8.2 null is the one value of the Null type", output(`var a = null; print(a, a == null, typeof a)`, "null true object\n")},
	})
}

func TestS8_3_BooleanType(t *testing.T) {
	runRows(t, []row{
		// 8.3: "consists of exactly two unique values. One is called true and the other is called false."
		{"8.3 true and false are two distinct values", output(`print(true == true, false == false, true == false, typeof true)`, "true true false boolean\n")},
	})
}

func TestS8_4_StringType(t *testing.T) {
	runRows(t, []row{
		// 8.4: "The leftmost character (if any) is at position 0, the next character (if any) at position 1"
		{"8.4 the characters of a string are at positions 0, 1, ...", output(`var s = "xyz"; print(s.charAt(0), s.charAt(1), s.charAt(2))`, "x y z\n")},
		// 8.4: "The length of a string is the number of distinct positions within it."
		{"8.4 the length of a string is its number of characters", output(`print("xyz".length, "a b".length)`, "3 3\n")},
		// 8.4: "The empty string has length zero and therefore contains no characters."
		{"8.4 the empty string has length zero", output(`print("".length)`, "0\n")},
		// 8.4: a String is a "sequence of zero or more Unicode characters", so a character code above 255 is one character
		{"8.4 a character is a 16-bit Unicode character", output(`var s = String.fromCharCode(65535, 256); print(s.length, s.charCodeAt(0), s.charCodeAt(1))`, "2 65535 256\n")},
	})
}

func TestS8_5_NumberType(t *testing.T) {
	runRows(t, []row{
		// 8.5: "the ... distinct Not-a-Number values ... are represented in ECMAScript as a single special NaN value"
		{"8.5 NaN is a number", output(`print(NaN, typeof NaN, 0/0)`, "NaN number NaN\n")},
		// 8.5: "positive Infinity and negative Infinity ... are produced by the program expressions +Infinity (or simply Infinity) and -Infinity"
		{"8.5 positive and negative infinity", output(`print(Infinity, +Infinity, -Infinity, typeof Infinity)`, "Infinity Infinity -Infinity number\n")},
		// 8.5: "there is both a positive zero and a negative zero ... produced by the program expressions +0 (or simply 0) and -0"
		{"8.5 there is a positive and a negative zero", output(`print(1/0, 1/+0, 1/-0)`, "Infinity Infinity -Infinity\n")},
		// 8.5: "for every finite positive number there is a corresponding negative number having the same magnitude"
		{"8.5 every finite number has a negative of the same magnitude", output(`var max = 1.7976931348623157e308, min = 5e-324; print(-max == 0 - max, -max * 2, -min < 0, -min == 0 - min)`, "true -Infinity true true\n")},
		// 8.5: normalized numbers have "e ... ranging from -1074 to 971": the largest finite number is (2^53-1) * 2^971
		{"8.5 the largest finite number", output(`var max = 1.7976931348623157e308; print(max * 2, max + 1 == max, max / 2 * 2 == max)`, "Infinity true true\n")},
		// 8.5: denormalized numbers have "m ... less than 2^52, and e is -1074": the smallest positive number is 2^-1074
		{"8.5 numbers below the normalized range are denormalized", output(`print(2.2250738585072014e-308 / 4 > 0, 4.9406564584124654e-324 > 0, 4.9406564584124654e-324 / 2)`, "true true 0\n")},
		// 8.5: "all the positive and negative integers whose magnitude is no greater than 2^53 are representable"
		{"8.5 integers up to 2^53 are exact", output(`print(9007199254740992, 9007199254740991 + 1, -9007199254740992)`, "9007199254740992 9007199254740992 -9007199254740992\n")},
		// 8.5 "the number value for x": "If two values of the set are equally close, then the one with an even significand is chosen"
		{"8.5 a tie rounds to the even significand", output(`print(9007199254740993, 9007199254740995)`, "9007199254740992 9007199254740996\n")},
		// 8.5 "the number value for x": "if 2^1024 was chosen, replace it with +Infinity"
		{"8.5 a value closest to 2^1024 rounds to Infinity", output(`print(1.7976931348623159e308, 1.7976931348623157e308 + 1e292)`, "Infinity Infinity\n")},
		// 8.5 "the number value for x": "if +0 was chosen, replace it with -0 if and only if x is less than zero"
		{"8.5 a negative value that rounds to zero is -0", output(`print(1 / (-4.9406564584124654e-324 * 0.5), 1 / (4.9406564584124654e-324 * 0.5))`, "-Infinity Infinity\n")},
	})
}

func TestS8_6_ObjectType(t *testing.T) {
	runRows(t, []row{
		// 8.6: "An Object is an unordered collection of properties. Each property consists of a name, a value and a set of attributes."
		{"8.6 an object holds named properties", output(`var o = new Object(); o.a = 1; o["b c"] = "two"; print(o.a, o["b c"], o["a"])`, "1 two 1\n")},

		// 8.6.1 ReadOnly: "Attempts by ECMAScript code to write to the property will be ignored."
		{"8.6.1 a write to a ReadOnly property is ignored", output(`Math.PI = 3; print(Math.PI)`, "3.141592653589793\n")},
		// 8.6.1 DontEnum: "The property is not to be enumerated by a for-in enumeration"
		{"8.6.1 a DontEnum property is not enumerated", output(`var o = new Object(); o.mine = 1; for (var p in o) print(p); print(o.toString == Object.prototype.toString)`, "mine\ntrue\n")},
		// 8.6.1 DontDelete: "Attempts to delete the property will be ignored."
		{"8.6.1 a DontDelete property survives delete", output(`delete Math.PI; print(Math.PI)`, "3.141592653589793\n")},
		// 8.6.1 Internal: "Internal properties have no name and are not directly accessible via the property accessor operators."
		{"8.6.1 internal properties are not reachable by name", output(`var o = new Object(); print(o["[[Prototype]]"], o["[[Class]]"], o.Class)`, "undefined undefined undefined\n")},

		// 8.6.2: "Properties of the [[Prototype]] object are exposed as properties of the child object for the purposes of get access, but not for put access."
		{"8.6.2 prototype properties are inherited for get but not for put",
			output(`function F() {} F.prototype.x = "proto"; var a = new F(); var b = new F(); a.x = "own"; print(a.x, b.x, F.prototype.x)`, "own proto proto\n")},
		// 8.6.2: "When an algorithm uses an internal property of an object and the object does not implement the indicated internal property, a runtime error is generated."
		{"8.6.2 calling an object without [[Call]] is a runtime error",
			runtimeError(`var o = new Object(); print("before"); o()`, "before\n")},
		// 8.6.2: an object without [[Construct]] cannot be used with new
		{"8.6.2 new on an object without [[Construct]] is a runtime error",
			runtimeError(`var o = new Object(); print("before"); new o()`, "before\n")},

		// 8.6.2.1 steps 1-3: "If O doesn't have a property with name P, go to step 4. Get the value of the property."
		{"8.6.2.1 an own property is read", output(`var o = new Object(); o.p = 5; print(o.p)`, "5\n")},
		// 8.6.2.1 steps 4-6: "Call the [[Get]] method of [[Prototype]] with property name P."
		{"8.6.2.1 a missing property is read from the prototype chain",
			output(`function A() {} A.prototype.p = "deep"; function B() {} B.prototype = new A(); var o = new B(); print(o.p)`, "deep\n")},
		// 8.6.2.1 step 4: "If the [[Prototype]] of O is null, return undefined."
		{"8.6.2.1 a property found nowhere is undefined", output(`var o = new Object(); print(o.nothing)`, "undefined\n")},
		// 8.6.2.1 step 1: an own property shadows the prototype's even when its value is undefined
		{"8.6.2.1 an own property shadows the prototype even when undefined",
			output(`function F() {} F.prototype.p = 1; var o = new F(); o.p = void 0; print(o.p)`, "undefined\n")},

		// 8.6.2.2 step 4: "Set the value of the property to V. The attributes of the property are not changed."
		{"8.6.2.2 overwriting a DontEnum property keeps it DontEnum",
			output(`function f() { return "mine" } Object.prototype.toString = f; var o = new Object(); for (var p in o) print(p); print(o.toString())`, "mine\n")},
		// 8.6.2.2 step 6: "Create a property with name P, set its value to V and give it empty attributes."
		{"8.6.2.2 a new property is enumerable and deletable",
			output(`var o = new Object(); o.x = 1; for (var p in o) print(p); print(delete o.x, o.x)`, "x\ntrue undefined\n")},
		// 8.6.2.2 steps 1-2: "Call the [[CanPut]] method of O with name P. If Result(1) is false, return."
		{"8.6.2.2 a put that [[CanPut]] refuses changes nothing", output(`Math.E = 1; print(Math.E)`, "2.718281828459045\n")},
		// 8.6.2.2 step 6: a put on an object creates an own property even if the prototype has one
		{"8.6.2.2 a put creates an own property in front of the prototype's",
			output(`function F() {} F.prototype.x = 1; var o = new F(); o.x = 2; delete o.x; print(o.x)`, "1\n")},

		// 8.6.2.3 steps 4-6: "Call the [[CanPut]] method of [[Prototype]] of O with property name P."
		{"8.6.2.3 a ReadOnly property of the prototype blocks a put on the child",
			output(`function F() {} F.prototype = Math; var o = new F(); o.PI = 3; print(o.PI)`, "3.141592653589793\n")},
		// 8.6.2.3 step 3: a property that is not ReadOnly can be put
		{"8.6.2.3 a property that is not ReadOnly can be written", output(`var o = new Object(); o.x = 1; o.x = 2; print(o.x)`, "2\n")},
		// 8.6.2.3 step 4: "If the [[Prototype]] of O is null, return true."
		{"8.6.2.3 a property absent from the whole chain can be created", output(`var o = new Object(); o.fresh = "yes"; print(o.fresh)`, "yes\n")},

		// 8.6.2.4 step 1: "If O has a property with name P, return true." Observed through with (§10.1.4).
		{"8.6.2.4 an own property is found by identifier resolution",
			output(`var x = "global"; var o = new Object(); o.x = "own"; with (o) print(x)`, "own\n")},
		// 8.6.2.4 step 3: "Call the [[HasProperty]] method of [[Prototype]] with property name P."
		{"8.6.2.4 an inherited property is found by identifier resolution",
			output(`var x = "global"; function F() {} F.prototype.x = "proto"; with (new F()) print(x)`, "proto\n")},
		// 8.6.2.4 step 2: "If the [[Prototype]] of O is null, return false."
		{"8.6.2.4 a property absent from the chain is not found",
			output(`var x = "global"; with (new Object()) print(x)`, "global\n")},
		// 8.6.2.4 step 1: a property whose value is undefined is still a property
		{"8.6.2.4 a property holding undefined is found",
			output(`var x = "global"; var o = new Object(); o.x = void 0; with (o) print(x)`, "undefined\n")},

		// 8.6.2.5 step 1: "If O doesn't have a property with name P, return true."
		{"8.6.2.5 deleting a missing property returns true", output(`var o = new Object(); print(delete o.nothing)`, "true\n")},
		// 8.6.2.5 step 2: "If the property has the DontDelete attribute, return false."
		{"8.6.2.5 deleting a DontDelete property returns false", output(`print(delete Math.PI)`, "false\n")},
		// 8.6.2.5 steps 3-4: "Remove the property with name P from O. Return true."
		{"8.6.2.5 deleting a property removes it", output(`var o = new Object(); o.p = 1; print(delete o.p, o.p)`, "true undefined\n")},
		// 8.6.2.5: [[Delete]] acts on O only, not on its prototype
		{"8.6.2.5 deleting an own property uncovers the prototype's",
			output(`function F() {} F.prototype.p = "proto"; var o = new F(); o.p = "own"; print(delete o.p, o.p, delete o.p, o.p)`, "true proto true proto\n")},

		// 8.6.2.6 hint String step 4: "If Result(3) is a primitive value, return Result(3)."
		{"8.6.2.6 with hint String, toString is called first",
			output(`function ts() { return "from toString" } function vo() { return 42 } var o = new Object(); o.toString = ts; o.valueOf = vo; print(String(o))`, "from toString\n")},
		// 8.6.2.6 hint String step 2: "If Result(1) is not an object, go to step 5."
		{"8.6.2.6 with hint String, a toString that is not an object is skipped",
			output(`function vo() { return 42 } var o = new Object(); o.toString = 1; o.valueOf = vo; print(String(o))`, "42\n")},
		// 8.6.2.6 hint String step 4 then 5: a toString that returns an object falls through to valueOf
		{"8.6.2.6 with hint String, an object from toString falls back to valueOf",
			output(`function ts() { return new Object() } function vo() { return 42 } var o = new Object(); o.toString = ts; o.valueOf = vo; print(String(o))`, "42\n")},
		// 8.6.2.6 hint String step 9: "Generate a runtime error."
		{"8.6.2.6 with hint String, no primitive from either method is a runtime error",
			runtimeError(`function self() { return this } var o = new Object(); o.toString = self; o.valueOf = self; print("before"); String(o)`, "before\n")},
		// 8.6.2.6 hint Number step 4: valueOf is called first
		{"8.6.2.6 with hint Number, valueOf is called first",
			output(`function ts() { return "7" } function vo() { return 42 } var o = new Object(); o.toString = ts; o.valueOf = vo; print(+o)`, "42\n")},
		// 8.6.2.6 hint Number step 2: "If Result(1) is not an object, go to step 5."
		{"8.6.2.6 with hint Number, a valueOf that is not an object is skipped",
			output(`function ts() { return "7" } var o = new Object(); o.toString = ts; o.valueOf = "x"; print(+o)`, "7\n")},
		// 8.6.2.6 hint Number step 4 then 5: a valueOf that returns an object falls through to toString
		{"8.6.2.6 with hint Number, an object from valueOf falls back to toString",
			output(`function ts() { return "7" } function vo() { return this } var o = new Object(); o.toString = ts; o.valueOf = vo; print(+o)`, "7\n")},
		// 8.6.2.6 hint Number step 9: "Generate a runtime error."
		{"8.6.2.6 with hint Number, no primitive from either method is a runtime error",
			runtimeError(`function self() { return this } var o = new Object(); o.toString = self; o.valueOf = self; print("before"); +o`, "before\n")},
		// 8.6.2.6: "When the [[DefaultValue]] method of O is called with no hint, then it behaves as if the hint were Number"
		{"8.6.2.6 with no hint, an ordinary object prefers valueOf",
			output(`function ts() { return "7" } function vo() { return 42 } var o = new Object(); o.toString = ts; o.valueOf = vo; print(o + 1, o == 42)`, "43 true\n")},
		// 8.6.2.6: "... unless O is a Date object (see section 15.9), in which case it behaves as if the hint were String."
		{"8.6.2.6 with no hint, a Date prefers toString",
			output(`function ts() { return "7" } var d = new Date(0); d.toString = ts; print(d + 1, d == "7")`, "71 true\n")},
	})
}

func TestS8_7_ReferenceType(t *testing.T) {
	runRows(t, []row{
		// 8.7: "the left-hand operand of an assignment is expected to produce a reference"
		{"8.7 a property access on the left of = names the property to set",
			output(`var o = new Object(); o["a" + "b"] = 1; print(o.ab)`, "1\n")},
		// 8.7: "Another use of the Reference type is to explain the determination of the this value for a function call."
		{"8.7 calling through a property reference passes its base object as this",
			output(`function who() { return this.name } var o = new Object(); o.name = "o"; o.who = who; print(o.who(), o["who"]())`, "o o\n")},

		// 8.7.1 step 1: "If Type(V) is Reference, return the base object component of V." delete acts on that base (§11.4.1).
		{"8.7.1 delete acts on the base object of its reference",
			output(`var o = new Object(); o.p = 1; var q = new Object(); q.p = 2; print(delete o.p, o.p, q.p)`, "true undefined 2\n")},
		// 8.7.1 step 2: "Generate a runtime error." delete calls GetBase on its operand (§11.4.1 step 2).
		{"8.7.1 delete of a value that is not a reference is an error",
			earlyOrRuntimeError(`print(delete 1)`)},

		// 8.7.2 step 1: "If Type(V) is Reference, return the property name component of V."
		{"8.7.2 delete removes the property its reference names",
			output(`var o = new Object(); o.ab = 1; o.a = 2; print(delete o["a" + "b"], o.ab, o.a)`, "true undefined 2\n")},
		// 8.7.2 step 2: "Generate a runtime error." delete calls GetPropertyName after GetBase (§11.4.1 step 3).
		{"8.7.2 delete of a parenthesised value that is not a reference is an error",
			earlyOrRuntimeError(`print(delete (1 + 2))`)},

		// 8.7.3 step 1: "If Type(V) is not Reference, return V."
		{"8.7.3 a value that is not a reference is used as is", output(`print((1 + 2) * 2)`, "6\n")},
		// 8.7.3 step 3: "If Result(2) is null, generate a runtime error."
		{"8.7.3 reading an undeclared variable is a runtime error",
			runtimeError(`print("before"); print(undeclared)`, "before\n")},
		// 8.7.3 step 4: "Call the [[Get]] method of Result(2), passing GetPropertyName(V) for the property name."
		{"8.7.3 reading a property reference gets the property", output(`var o = new Object(); o.p = "v"; print(o.p)`, "v\n")},

		// 8.7.4 step 1: "If Type(V) is not Reference, generate a runtime error."
		{"8.7.4 assigning to a value that is not a reference is an error", earlyOrRuntimeError(`1 = 2`)},
		// 8.7.4 step 1: the error comes when the assignment runs, after the left operand is evaluated
		{"8.7.4 assigning to a call result is an error", earlyOrRuntimeError(`function f() { return 1 } f() = 2`)},
		// 8.7.4 step 4: "Call the [[Put]] method of Result(2), passing GetPropertyName(V) for the property name and W for the value."
		{"8.7.4 assigning to a property reference puts the property", output(`var o = new Object(); o.p = "v"; print(o.p)`, "v\n")},
		// 8.7.4 step 6: "Call the [[Put]] method for the global object"
		{"8.7.4 assigning to an undeclared variable creates a global property",
			output(`function f() { created = "global" } f(); print(created)`, "global\n")},
	})
}

func TestS8_8_ListType(t *testing.T) {
	runRows(t, []row{
		// 8.8: "Values of the List type are simply ordered sequences of values. These sequences may be of any length."
		{"8.8 arguments are passed as an ordered list",
			output(`function f(a, b, c) { print(a, b, c) } f("x", "y", "z")`, "x y z\n")},
		// 8.8: "These sequences may be of any length."
		{"8.8 an argument list may be empty or longer than the parameters",
			output(`function f(a) { print(a, arguments.length) } f(); f(1, 2, 3)`, "undefined 0\n1 3\n")},
	})
}

func TestS8_9_CompletionType(t *testing.T) {
	runRows(t, []row{
		// 8.9: "abrupt completion because of return V"
		{"8.9 a return completion leaves the function from inside a loop",
			output(`function f() { while (true) { return "out" } } print(f())`, "out\n")},
		// 8.9: "abrupt completion because of break"
		{"8.9 a break completion leaves the loop",
			output(`var i = 0; while (true) { i++; if (i == 3) break; } print(i)`, "3\n")},
		// 8.9: "abrupt completion because of continue"
		{"8.9 a continue completion skips the rest of the iteration",
			output(`var i = 0, s = ""; while (i < 4) { i++; if (i == 2) continue; s += i; } print(s)`, "134\n")},
		// 8.9: "normal completion after value V", observed as the value of eval (§15.1.2.1)
		{"8.9 a value completion is the value of a program", output(`print(eval("1; 2; 3"))`, "3\n")},
		// 8.9: "abrupt completion after value V because of break": the loop's value is the last value before the break
		{"8.9 a break completion carries the value computed before it",
			output(`print(eval("var i = 0; while (true) { i++; 'v' + i; break; }"))`, "v1\n")},
	})
}
