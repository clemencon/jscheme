package tests

import (
	"math"
	"testing"
)

// Chapter 15, Native ECMAScript objects. One Test function per
// second-level section; the chapter introduction is TestS15_NativeObjects.

func TestS15_NativeObjects(t *testing.T) {
	runRows(t, []row{
		// 15: "if a function ... is given fewer arguments than the function is specified to require, the function ... shall behave exactly as if it had been given sufficient additional arguments, each such argument being the undefined value"
		{"15 a missing argument of a built-in function is undefined", output(`print(isNaN(), parseInt(), "undefined".indexOf())`, "true NaN 0\n")},
		// 15: "Every built-in function and every built-in constructor has the Function prototype object ... as the value of its internal [[Prototype]] property"
		{"15 built-in functions inherit from Function.prototype",
			output(`Function.prototype.marker = "fp"; print(Math.abs.marker, parseInt.marker, Object.marker, String.prototype.charAt.marker)`, "fp fp fp fp\n")},
		// 15: "Every built-in prototype object has the Object prototype object ... as the value of its internal [[Prototype]] property"
		{"15 built-in prototype objects inherit from Object.prototype",
			output(`Object.prototype.marker = "op"; print(Function.prototype.marker, Array.prototype.marker, String.prototype.marker, Boolean.prototype.marker, Number.prototype.marker, Date.prototype.marker)`, "op op op op op op\n")},
		// 15: "Every native prototype object associated with a program-created function also has the Object prototype object as the value of its internal [[Prototype]] property."
		{"15 the prototype of a declared function inherits from Object.prototype",
			output(`function F() {} Object.prototype.marker = "op"; print(F.prototype.marker)`, "op\n")},
		// 15: "None of the built-in functions described in this section shall implement the internal [[Construct]] method unless otherwise specified"
		{"15 a built-in function that is not a constructor cannot be used with new", runtimeError(`print("before"); new Math.abs(1)`, "before\n")},
		// 15: "None of the built-in functions ... shall initially have a prototype property unless otherwise specified"
		{"15 a built-in function that is not a constructor has no prototype property", output(`print(typeof Math.abs.prototype, typeof parseInt.prototype)`, "undefined undefined\n")},
		// 15: "None of the built-in functions described in this section shall initially have an arguments property"
		{"15 a built-in function has no arguments property",
			output(`print(typeof parseInt.arguments, typeof Math.abs.arguments)`, "undefined undefined\n")},
		// 15: "nor is one ever automatically added during execution in the manner described for script functions in section 10.1.8"
		{"15 a built-in function gets no arguments property while it runs",
			output(`var seen; function cmp(x, y) { seen = typeof a.sort.arguments; return x - y } var a = new Array(2, 1); a.sort(cmp); print(seen)`, "undefined\n")},
		// 15: "the value of the length property is equal to the number of named arguments shown in the section heading"
		{"15 the length of a built-in function is its number of named arguments",
			output(`print(parseInt.length, parseFloat.length, Math.atan2.length, String.prototype.substring.length, Math.random.length, Object.length)`, "2 1 2 2 0 1\n")},
		// 15: "the value of the length property of that function object [indexOf(searchString, position)] is 2"
		{"15 the length of a built-in function counts every named argument",
			output(`print(String.prototype.indexOf.length, String.prototype.lastIndexOf.length, String.prototype.split.length)`, "2 2 1\n")},
		// 15: "a length property of a built-in function object ... has the attributes { ReadOnly, DontDelete, DontEnum }": ReadOnly
		{"15 the length of a built-in function cannot be changed", output(`parseInt.length = 5; print(parseInt.length)`, "2\n")},
		// 15: length has the attribute DontDelete
		{"15 the length of a built-in function cannot be deleted", output(`print(delete parseInt.length, parseInt.length)`, "false 2\n")},
		// 15: "Every other property described in this section has the attribute { DontEnum }"
		{"15 the properties of built-in objects are not enumerated",
			output(`var found = ""; for (var p in Math) if (p == "PI" || p == "abs") found += p; for (var p in String.prototype) if (p == "charAt" || p == "constructor") found += p; for (var p in parseInt) if (p == "length") found += p; print("[" + found + "]")`, "[]\n")},
		// 15: "Every other property described in this section has the attribute { DontEnum } (and no others)": not DontDelete
		{"15 a built-in method can be deleted",
			output(`print(delete Math.abs, typeof Math.abs, delete String.prototype.charAt, typeof "a".charAt)`, "true undefined true undefined\n")},
	})
}

func TestS15_1_TheGlobalObject(t *testing.T) {
	runRows(t, []row{
		// 15.1: "The global object does not have a [[Construct]] property"
		{"15.1 the global object cannot be used with new", runtimeError(`print("before"); new this`, "before\n")},
		// 15.1: "The global object does not have a [[Call]] property"
		{"15.1 the global object cannot be called", runtimeError(`print("before"); this()`, "before\n")},

		// 15.1.1.1: "The initial value of NaN is NaN."
		{"15.1.1.1 the global NaN is NaN", output(`print(NaN, NaN == NaN, typeof NaN)`, "NaN false number\n")},
		// 15.1.1.1 with §15: NaN has the attribute DontEnum only, so it can be changed
		{"15.1.1.1 the global NaN can be assigned", output(`NaN = 1; print(NaN)`, "1\n")},
		// 15.1.1.2: "The initial value of Infinity is +Infinity."
		{"15.1.1.2 the global Infinity is positive infinity", output(`print(Infinity, Infinity > 1e308, -Infinity)`, "Infinity true -Infinity\n")},
		// 15.1.1.2 with §15: Infinity has the attribute DontEnum only, so it can be changed
		{"15.1.1.2 the global Infinity can be assigned", output(`Infinity = 2; print(Infinity)`, "2\n")},
		// 15.1.1: the value properties have the attribute DontEnum (§15)
		{"15.1.1 NaN and Infinity are not enumerated",
			output(`var found = ""; for (var p in this) if (p == "NaN" || p == "Infinity") found += p; print("[" + found + "]")`, "[]\n")},

		// 15.1.2.1 step 1: "If x is not a string value, return x."
		{"15.1.2.1 eval of a value that is not a string returns it", output(`var o = new Object(); print(eval(5), eval(true), eval(o) == o, eval())`, "5 true true undefined\n")},
		// 15.1.2.1 step 1: a String object is not a string value
		{"15.1.2.1 eval of a String object returns the object", output(`var s = new String("1 + 1"); print(eval(s) == s, typeof eval(s))`, "true object\n")},
		// 15.1.2.1 steps 2-4: "Parse x as an ECMAScript Program." "Evaluate the program" "return the value V"
		{"15.1.2.1 eval runs a string as a program and returns its value", output(`print(eval("1 + 2"), eval("var x = 4; x * x"))`, "3 16\n")},
		// 15.1.2.1 step 2: "If the parse fails, generate a runtime error."
		{"15.1.2.1 eval of a string that does not parse is a runtime error", runtimeError(`print("before"); eval("1 +")`, "before\n")},
		// 15.1.2.1 step 2: the string is parsed as a Program, so return is not allowed (§12.9)
		{"15.1.2.1 eval of a return statement is a runtime error", runtimeError(`function f() { return eval("return 1") } print("before"); f()`, "before\n")},
		// 15.1.2.1 step 5: "Return undefined." when the program has no value
		{"15.1.2.1 eval of a program without a value returns undefined", output(`print(eval("var x = 1"), eval("function f() {}"))`, "undefined undefined\n")},
		// 15.1.2.1 step 3: a runtime error in the program is a runtime error of the caller
		{"15.1.2.1 a runtime error inside eval stops the program", runtimeError(`print("before"); eval("undeclared"); print("after")`, "before\n")},

		// 15.1.2.2 step 22: digits up to the first character that is not a radix-R digit
		{"15.1.2.2 parseInt reads the leading digits of a string", output(`print(parseInt("123abc"), parseInt("12.9"), parseInt("1e3"))`, "123 12 1\n")},
		// 15.1.2.2 step 1: "Call ToString(string)."
		{"15.1.2.2 parseInt converts its argument to a string", output(`print(parseInt(42.7), parseInt(null), parseInt(true))`, "42 NaN NaN\n")},
		// 15.1.2.2 step 2: "remove leading whitespace" (StrWhiteSpaceChar)
		{"15.1.2.2 parseInt skips leading white space", output(`print(parseInt(" \t\n\r\f\x0B42"))`, "42\n")},
		// 15.1.2.2 step 2: only StrWhiteSpaceChar is skipped, not a no-break space
		{"15.1.2.2 parseInt does not skip a no-break space", output(`print(parseInt("\u00A042"))`, "NaN\n")},
		// 15.1.2.2 steps 3-5: a leading + or - sets the sign
		{"15.1.2.2 parseInt reads a sign", output(`print(parseInt("-42"), parseInt("+42"), parseInt(" -7x"))`, "-42 42 -7\n")},
		// 15.1.2.2 step 23: "If Z is empty, return NaN."
		{"15.1.2.2 parseInt without digits returns NaN", output(`print(parseInt(""), parseInt("abc"), parseInt("-"), parseInt(" "))`, "NaN NaN NaN NaN\n")},
		// 15.1.2.2 steps 7-9: a radix from 2 to 36
		{"15.1.2.2 parseInt reads digits in the given radix", output(`print(parseInt("ff", 16), parseInt("FF", 16), parseInt("101", 2), parseInt("z", 36), parseInt("Z", 36), parseInt("777", 8))`, "255 255 5 35 35 511\n")},
		// 15.1.2.2 step 22: a digit too large for the radix ends the number
		{"15.1.2.2 parseInt stops at a digit outside the radix", output(`print(parseInt("1012", 2), parseInt("19", 8), parseInt("2", 2))`, "5 1 NaN\n")},
		// 15.1.2.2 step 8: "if Result(7) < 2 or Result(7) > 36, return NaN"
		{"15.1.2.2 parseInt with a radix outside 2 to 36 returns NaN", output(`print(parseInt("1", 1), parseInt("1", 37), parseInt("1", -2))`, "NaN NaN NaN\n")},
		// 15.1.2.2 step 7: "Call ToInt32(radix)."
		{"15.1.2.2 parseInt converts the radix with ToInt32", output(`print(parseInt("11", 2.9), parseInt("11", "16"), parseInt("11", 4294967298))`, "3 17 3\n")},
		// 15.1.2.2 step 8: "If Result(7) is zero, go to step 12" (as if no radix were given)
		{"15.1.2.2 parseInt with radix 0 behaves as without a radix", output(`print(parseInt("42", 0), parseInt("0x1A", 0))`, "42 26\n")},
		// 15.1.2.2 step 10: "If R = 16 and ... the first two characters of Result(5) are either 0x or 0X," they are skipped
		{"15.1.2.2 parseInt with radix 16 skips a 0x prefix", output(`print(parseInt("0x1f", 16), parseInt("0X1F", 16), parseInt("-0x10", 16))`, "31 31 -16\n")},
		// 15.1.2.2 step 10: with another radix, the 0x prefix is not special
		{"15.1.2.2 parseInt with radix 10 does not skip 0x", output(`print(parseInt("0x1f", 10))`, "0\n")},
		// 15.1.2.2 steps 13 and 17-18: without a radix, 0x or 0X means hexadecimal
		{"15.1.2.2 parseInt without a radix reads 0x as hexadecimal", output(`print(parseInt("0x1A"), parseInt("0XFF"), parseInt("-0x10"), parseInt("0x"))`, "26 255 -16 NaN\n")},
		// 15.1.2.2 steps 14-15: "Let R be 8." for a leading 0 without x
		{"15.1.2.2 parseInt without a radix reads a leading 0 as octal", output(`print(parseInt("010"), parseInt("0777"), parseInt("08"), parseInt("-012"))`, "8 511 0 -10\n")},
		// 15.1.2.2 steps 20-21: "Let R be 10." otherwise
		{"15.1.2.2 parseInt without a radix reads other strings as decimal", output(`print(parseInt("10"), parseInt("0"), parseInt("90"))`, "10 0 90\n")},
		// 15.1.2.2 steps 25-26: "Compute the number value for Result(24)." "Return sign * Result(25)."
		{"15.1.2.2 parseInt rounds a long integer to a number value", output(`print(parseInt("9007199254740993"), parseInt("-0") == 0, 1 / parseInt("-0"))`, "9007199254740992 true -Infinity\n")},

		// 15.1.2.3 step 4: "Compute the longest prefix of Result(2) ... that satisfies the syntax of a StrDecimalLiteral"
		{"15.1.2.3 parseFloat reads the longest decimal prefix", output(`print(parseFloat("3.25abc"), parseFloat("1e3x"), parseFloat("1.5.5"), parseFloat(".5"), parseFloat("5."))`, "3.25 1000 1.5 0.5 5\n")},
		// 15.1.2.3 step 4: an exponent without digits is not part of the prefix
		{"15.1.2.3 parseFloat leaves out an incomplete exponent", output(`print(parseFloat("1e"), parseFloat("2e+"), parseFloat("3E-x"))`, "1 2 3\n")},
		// 15.1.2.3 step 4: StrDecimalLiteral ::: Infinity
		{"15.1.2.3 parseFloat reads Infinity", output(`print(parseFloat("Infinity"), parseFloat("Infinityx"))`, "Infinity Infinity\n")},
		// 15.1.2.3: StrDecimalLiteral has no hexadecimal form, so 0x stops after the 0
		{"15.1.2.3 parseFloat does not read hexadecimal", output(`print(parseFloat("0x10"))`, "0\n")},
		// 15.1.2.3 step 3: "If neither Result(2) nor any prefix of Result(2) satisfies the syntax of a StrDecimalLiteral ... return NaN."
		{"15.1.2.3 parseFloat without a decimal prefix returns NaN", output(`print(parseFloat(""), parseFloat("abc"), parseFloat("."), parseFloat("e5"))`, "NaN NaN NaN NaN\n")},
		// 15.1.2.3 step 2: "remove leading whitespace"
		{"15.1.2.3 parseFloat skips leading white space", output(`print(parseFloat(" \t\n\r\f\x0B2.5"))`, "2.5\n")},
		// 15.1.2.3 step 1: "Call ToString(string)."
		{"15.1.2.3 parseFloat converts its argument to a string", output(`print(parseFloat(1.5e3), parseFloat(true))`, "1500 NaN\n")},
		// 15.1.2.3 step 3: StrDecimalLiteral (§9.3.1) has no sign, so a signed string has no decimal prefix
		{"15.1.2.3 parseFloat of a signed string returns NaN", output(`print(parseFloat("-1.5"), parseFloat("+1"))`, "NaN NaN\n")},
		// 15.1.2.3 step 5: "Return the number value for the MV of Result(4)."
		{"15.1.2.3 parseFloat rounds to the nearest number value", output(`print(parseFloat("0.1"), parseFloat("9007199254740993"))`, "0.1 9007199254740992\n")},

		// 15.1.2.4 step 7: the 69 characters A-Z a-z 0-9 @ * _ + - . / are kept
		{"15.1.2.4 escape keeps letters, digits and @*_+-./",
			output(`print(escape("ABCXYZabcxyz0189@*_+-./"))`, "ABCXYZabcxyz0189@*_+-./\n")},
		// 15.1.2.4 step 12: "a string containing three characters %xy where xy are two hexadecimal digits"
		{"15.1.2.4 escape writes other characters below 256 as %xx",
			output(`print(escape(" !\"#"), escape("\0").length, escape(",:;=?").toUpperCase(), escape("\u00ff").toUpperCase())`, "%20%21%22%23 3 %2C%3A%3B%3D%3F %FF\n")},
		// 15.1.2.4 step 10: "a string containing six characters %uwxyz where wxyz are four hexadecimal digits"
		{"15.1.2.4 escape writes characters from 256 as %uxxxx",
			output(`print(escape("\u0100").toUpperCase(), escape("\u4e2d").toUpperCase(), escape("a\uabcdb").toUpperCase())`, "%U0100 %U4E2D A%UABCDB\n")},
		// 15.1.2.4 step 1: "Call ToString(string)."
		{"15.1.2.4 escape converts its argument to a string", output(`print(escape(1.5), escape(null))`, "1.5 null\n")},

		// 15.1.2.5 steps 14-17: "%xy" with two hexadecimal digits is replaced by the character
		{"15.1.2.5 unescape decodes %xx", output(`print(unescape("%41%62%20c"), unescape("%4a%4A"))`, "Ab c JJ\n")},
		// 15.1.2.5 steps 9-12: "%uwxyz" with four hexadecimal digits is replaced by the character
		{"15.1.2.5 unescape decodes %uxxxx", output(`print(unescape("%u0041%u0062"), unescape("%u4e2d").charCodeAt(0), unescape("%uABCD").charCodeAt(0))`, "Ab 20013 43981\n")},
		// 15.1.2.5 steps 8, 10, 14, 15 and 18: an incomplete or invalid escape is kept as it is
		{"15.1.2.5 unescape keeps an invalid escape", output(`print(unescape("%"), unescape("%4"), unescape("%G1"), unescape("%u12"), unescape("%u12G4"), unescape("100%"))`, "% %4 %G1 %u12 %u12G4 100%\n")},
		// 15.1.2.5 step 10 then 14: a %u that is not followed by four hex digits may still start nothing else
		{"15.1.2.5 unescape of %u with too few hex digits keeps them", output(`print(unescape("%u00"), unescape("%u004"))`, "%u00 %u004\n")},
		// 15.1.2.5: unescape undoes escape
		{"15.1.2.5 unescape reverses escape", output(`var s = "a b,c\u00e9\u4e2d%"; print(unescape(escape(s)) == s)`, "true\n")},

		// 15.1.2.6: "Applies ToNumber to its argument, then returns true if the result is NaN"
		{"15.1.2.6 isNaN tests its argument converted to a number", output(`print(isNaN(NaN), isNaN("abc"), isNaN(void 0), isNaN(new Object()))`, "true true true true\n")},
		// 15.1.2.6: "and otherwise returns false"
		{"15.1.2.6 isNaN is false for anything that converts to a number", output(`print(isNaN(1), isNaN("12"), isNaN(""), isNaN(null), isNaN(Infinity), isNaN(true))`, "false false false false false false\n")},

		// 15.1.2.7: "returns false if the result is NaN, +Infinity, or -Infinity"
		{"15.1.2.7 isFinite is false for NaN and the infinities", output(`print(isFinite(NaN), isFinite(Infinity), isFinite(-Infinity), isFinite("x"), isFinite("Infinity"))`, "false false false false false\n")},
		// 15.1.2.7: "and otherwise returns true"
		{"15.1.2.7 isFinite is true for finite numbers", output(`print(isFinite(0), isFinite(-1.5), isFinite("12"), isFinite(null), isFinite(1.7976931348623157e308))`, "true true true true true\n")},

		// 15.1.3.1: "Object ( . . . ) See sections 15.2.1 and 15.2.2."
		{"15.1.3.1 Object is the Object constructor", output(`print(typeof Object, Object.prototype.constructor == Object, typeof new Object())`, "function true object\n")},
		// 15.1.3.2: "Function ( . . . ) See sections 15.3.1 and 15.3.2."
		{"15.1.3.2 Function is the Function constructor", output(`print(typeof Function, Function.prototype.constructor == Function, typeof new Function())`, "function true function\n")},
		// 15.1.3.3: "Array ( . . . ) See section 15.4.1."
		{"15.1.3.3 Array is the Array constructor", output(`print(typeof Array, Array.prototype.constructor == Array, new Array(1, 2).length)`, "function true 2\n")},
		// 15.1.3.4: "String ( . . . ) See sections 15.5.1 and 15.5.2."
		{"15.1.3.4 String is the String constructor", output(`print(typeof String, String.prototype.constructor == String, String(12) + 3)`, "function true 123\n")},
		// 15.1.3.5: "Boolean ( . . . ) See sections 15.6.1 and 15,6,2."
		{"15.1.3.5 Boolean is the Boolean constructor", output(`print(typeof Boolean, Boolean.prototype.constructor == Boolean, Boolean(1))`, "function true true\n")},
		// 15.1.3.6: "Number ( . . . ) See sections 15.7.1 and 15.7.2."
		{"15.1.3.6 Number is the Number constructor", output(`print(typeof Number, Number.prototype.constructor == Number, Number("12") + 3)`, "function true 15\n")},
		// 15.1.3.7: "Date ( . . . ) See section 15.9.2."
		{"15.1.3.7 Date is the Date constructor", output(`print(typeof Date, Date.prototype.constructor == Date, new Date(0).getTime())`, "function true 0\n")},
		// 15.1.4.1: "Math See section 15.8."
		{"15.1.4.1 Math is the Math object", output(`print(typeof Math, Math.abs(-2), Math.PI == 3.141592653589793)`, "object 2 true\n")},
	})
}

// §15.2, Object objects.

// classOfJS defines classOf(v): Object.prototype.toString applied to v,
// which shows the [[Class]] of an object (§15.2.4.2). ES1 has no call or
// apply, so the method is copied onto the object first.
const classOfJS = `function classOf(o) { o.classOfMethod = Object.prototype.toString; return o.classOfMethod() } `

func TestS15_2_ObjectObjects(t *testing.T) {
	runRows(t, []row{
		// 15.2.1.1 step 1: "If the value is null or undefined, create and return a new object with no properties"
		{"15.2.1.1 Object(null) and Object(undefined) create a new object",
			output(`var a = Object(null), b = Object(void 0); print(typeof a, typeof b, a == b, a.toString())`, "object object false [object Object]\n")},
		// 15.2.1.1 step 2: "Return ToObject(value)."
		{"15.2.1.1 Object(value) converts a primitive to an object",
			output(`var s = Object("ab"), n = Object(5), b = Object(true); print(typeof s, s.length, typeof n, n + 1, typeof b, b.valueOf())`, "object 2 object 6 object true\n")},
		// 15.2.1.1 step 2: ToObject of an object is the object itself (§9.9)
		{"15.2.1.1 Object(object) returns the same object", output(`var o = new Object(); print(Object(o) == o)`, "true\n")},
		// 15.2.1.2: "Create and return a new object with no properties"
		{"15.2.1.2 Object() creates a new plain object", output(`var a = Object(), b = Object(); print(typeof a, a == b, a.toString())`, "object false [object Object]\n")},

		// 15.2.2.1 step 2: "If the value is a native ECMAScript object, do not create a new object; simply return value."
		{"15.2.2.1 new Object(object) returns the same object", output(`var o = new Object(); var f = new Function(); print(new Object(o) == o, new Object(f) == f)`, "true true\n")},
		// 15.2.2.1 steps 4-6: "If the type of the value is String ... Boolean ... Number, return ToObject(value)."
		{"15.2.2.1 new Object(primitive) wraps it in an object",
			output(classOfJS+`print(classOf(new Object("s")), classOf(new Object(true)), classOf(new Object(1)), new Object("s") + "!")`, "[object String] [object Boolean] [object Number] s!\n")},
		// 15.2.2.1 step 7: for null or undefined, "Create a new native ECMAScript object ... [[Class]] ... "Object""
		{"15.2.2.1 new Object(null) creates a plain object", output(classOfJS+`print(classOf(new Object(null)), classOf(new Object(void 0)))`, "[object Object] [object Object]\n")},
		// 15.2.2.2: "The [[Prototype]] property of the newly constructed object is set to the Object prototype object."
		{"15.2.2.2 new Object() inherits from Object.prototype", output(`Object.prototype.inherited = "yes"; var o = new Object(); print(o.inherited, o.toString())`, "yes [object Object]\n")},
		// 15.2.2.2: each call creates a new object
		{"15.2.2.2 new Object() creates a distinct object each time", output(`print(new Object() == new Object())`, "false\n")},

		// 15.2.3: "The value of the internal [[Prototype]] property of the Object constructor is the Function prototype object."
		{"15.2.3 the Object constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(Object.marker)`, "fp\n")},
		// 15.2.3: "Besides the internal [[Call]] and [[Construct]] properties and the length property"
		{"15.2.3 the Object constructor has length 1", output(`print(Object.length)`, "1\n")},
		// 15.2.3.1: "The initial value of Object.prototype is the built-in Object prototype object"
		{"15.2.3.1 Object.prototype is the prototype of new objects", output(`Object.prototype.p = 1; print(new Object().p)`, "1\n")},
		// 15.2.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.2.3.1 Object.prototype cannot be replaced, deleted or enumerated",
			output(`var original = Object.prototype; Object.prototype = null; var deleted = delete Object.prototype; var found = ""; for (var p in Object) if (p == "prototype") found += p; print(Object.prototype == original, deleted, "[" + found + "]")`, "true false []\n")},

		// 15.2.4: "The value of the internal [[Prototype]] property of the Object prototype object is null."
		{"15.2.4 the prototype chain ends at Object.prototype", output(`print(Object.prototype.nothing, new Object().nothing)`, "undefined undefined\n")},
		// 15.2.4.1: "The initial value of Object.prototype.constructor is the built-in Object constructor."
		{"15.2.4.1 Object.prototype.constructor is Object", output(`print(Object.prototype.constructor == Object, new Object().constructor == Object)`, "true true\n")},
		// 15.2.4.2 steps 1-4: "[object " + [[Class]] + "]"
		{"15.2.4.2 toString gives [object Class]", output(`print(new Object().toString(), Object.prototype.toString())`, "[object Object] [object Object]\n")},
		// 15.2.4.2: the [[Class]] of each kind of built-in object
		{"15.2.4.2 toString shows the class of built-in objects",
			output(classOfJS+`print(classOf(new Function()), classOf(new Array()), classOf(new String("")), classOf(new Boolean(false)), classOf(new Number(0)), classOf(new Date(0)))`,
				"[object Function] [object Array] [object String] [object Boolean] [object Number] [object Date]\n")},
		// 15.2.4.2: a declared function has the class Function (§13, §15.3.2.1 step 15)
		{"15.2.4.2 a declared function and its instances have their classes",
			output(classOfJS+`function F() {} print(classOf(F), classOf(new F()))`, "[object Function] [object Object]\n")},
		// 15.2.4.3: "As a rule, the valueOf method for an object simply returns the object"
		{"15.2.4.3 valueOf returns the object itself", output(`var o = new Object(); print(o.valueOf() == o, typeof o.valueOf())`, "true object\n")},

		// 15.2.5: "Object instances have no special properties beyond those inherited from the Object prototype object."
		{"15.2.5 a new object has no own properties", output(`var n = 0; for (var p in new Object()) n++; var o = new Object(); print(n, o.toString == Object.prototype.toString)`, "0 true\n")},
	})
}

// §15.3, Function objects.

func TestS15_3_FunctionObjects(t *testing.T) {
	runRows(t, []row{
		// 15.3.1: "the function call Function(...) is equivalent to the object creation expression new Function(...)"
		{"15.3.1 Function called as a function creates a function", output(`var f = Function("a", "return a * 2"); print(typeof f, f(21))`, "function 42\n")},
		// 15.3.1.1: "Create and return a new Function object exactly if the function constructor had been called with the same arguments"
		{"15.3.1.1 Function(p1, ..., body) builds the same function as new Function", output(`var f = Function("a", "b", "return a - b"); print(f(5, 3), f.length)`, "2 2\n")},

		// 15.3.2.1 step 2: "If no arguments were given, let body be the empty string"
		{"15.3.2.1 new Function() creates a function that returns undefined", output(`var f = new Function(); print(typeof f, f(), f.length)`, "function undefined 0\n")},
		// 15.3.2.1 step 3: "If one argument was given, let body be that argument"
		{"15.3.2.1 a single argument is the body", output(`print(new Function("return 7")())`, "7\n")},
		// 15.3.2.1 steps 4-12: P is the first arguments joined with commas
		{"15.3.2.1 the arguments before the body are joined into the parameter list",
			output(`var a = new Function("a", "b", "c", "return a+b+c"), b = new Function("a, b, c", "return a+b+c"), c = new Function("a,b", "c", "return a+b+c"); print(a(1, 2, 3), b(1, 2, 3), c(1, 2, 3))`, "6 6 6\n")},
		// 15.3.2.1 steps 5, 9 and 13: the parameters and the body are converted with ToString
		{"15.3.2.1 the parameters and the body are converted to strings",
			output(`function p() { return "x" } var o = new Object(); o.toString = p; var f = new Function(o, new String("return x + 1")); print(f(1))`, "2\n")},
		// 15.3.2.1 step 17: "both P and Result(13) may contain whitespace, line terminators, and comments"
		{"15.3.2.1 the parameters and body may contain comments and line breaks",
			output(`var f = new Function("a /* first */,\n b", "// sum\nreturn a + b"); print(f(2, 3))`, "5\n")},
		// 15.3.2.1 step 17: "if either P or Result(13) is syntactically incorrect ... a runtime error is generated"
		{"15.3.2.1 a body that does not parse is a runtime error", runtimeError(`print("before"); new Function("return (")`, "before\n")},
		// 15.3.2.1 step 17: "The string value P must be parsable as a FormalParameterList(opt)"
		{"15.3.2.1 parameters that do not parse are a runtime error", runtimeError(`print("before"); new Function("1a", "return 1")`, "before\n")},
		// 15.3.2.1 step 18: "constructs a new object whose [[Prototype]] property is equal to the value of F.prototype at the time the [[Construct]] method is invoked"
		{"15.3.2.1 new uses the prototype property at the time of construction",
			output(`function F() {} var p1 = new Object(); p1.tag = "first"; var p2 = new Object(); p2.tag = "second"; F.prototype = p1; var a = new F(); F.prototype = p2; var b = new F(); print(a.tag, b.tag)`, "first second\n")},
		// 15.3.2.1 step 18: "(but if this value is not an object then the value of Object.prototype is used)"
		{"15.3.2.1 new uses Object.prototype when the prototype property is not an object",
			output(`function F() {} F.prototype = 5; var o = new F(); print(o.toString == Object.prototype.toString, typeof o)`, "true object\n")},
		// 15.3.2.1 step 18: "then invokes F as a function ... with the new object as the this value and the arguments given"
		{"15.3.2.1 new calls the function with the new object as this", output(`function F(a, b) { this.sum = a + b } print(new F(2, 3).sum)`, "5\n")},
		// 15.3.2.1 step 18: "If the result of invoking the [[Call]] method is an object, that object becomes the result"
		{"15.3.2.1 an object returned by a constructor replaces the new object",
			output(`function F() { this.tag = "new"; var other = new Object(); other.tag = "returned"; return other } print(new F().tag)`, "returned\n")},
		// 15.3.2.1 step 18: "otherwise the new object becomes the result"
		{"15.3.2.1 a primitive returned by a constructor is ignored", output(`function F() { this.tag = "new"; return 5 } print(new F().tag)`, "new\n")},
		// 15.3.2.1 step 19: "it will use "anonymous" as the name of the function in rendering the function as a string"
		{"15.3.2.1 a function from the constructor is shown with the name anonymous",
			output(`var f = new Function("x", "return x * 3"); eval(f.toString()); print(anonymous(5))`, "15\n")},
		// 15.3.2.1 steps 20-21: "The length property of F is set to" the number of formal parameters
		{"15.3.2.1 length is the number of formal parameters", output(`print(new Function("a", "b, c", "").length, new Function("").length)`, "3 0\n")},
		// 15.3.2.1 step 21: length has the attributes { DontDelete, DontEnum, ReadOnly }: ReadOnly
		{"15.3.2.1 length cannot be changed", output(`var f = new Function("a", ""); f.length = 9; print(f.length)`, "1\n")},
		// 15.3.2.1 step 21: DontDelete
		{"15.3.2.1 length cannot be deleted", output(`var f = new Function("a", ""); print(delete f.length, f.length)`, "false 1\n")},
		// 15.3.2.1 steps 22-23: "The prototype property of F is set to" a new object "with attributes { DontEnum }"
		{"15.3.2.1 a new function has a prototype object", output(`var f = new Function(""); var found = ""; for (var p in f) if (p == "prototype") found += p; print(typeof f.prototype, "[" + found + "]")`, "object []\n")},
		// 15.3.2.1 step 24: "The constructor property of Result(22) is set to F. This property is given attributes { DontEnum }."
		{"15.3.2.1 the prototype object points back to the function", output(`var f = new Function(""); var n = 0; for (var p in f.prototype) n++; print(f.prototype.constructor == f, n)`, "true 0\n")},
		// 15.3.2.1 step 25: "The arguments property of F is set to null."
		{"15.3.2.1 a new function has a null arguments property", output(`var f = new Function(""); print(f.arguments == null)`, "true\n")},
		// 15.3.2.1: the body is function code, so it can use arguments and return (§10.1.2)
		{"15.3.2.1 the body is function code", output(`var f = new Function("return arguments.length"); print(f(1, 2, 3))`, "3\n")},
		// 15.3.2.1 with §10.2.3: the body's scope chain is its activation object and the global object
		{"15.3.2.1 the body does not see the variables of the code that created it",
			output(`var x = "global"; function make() { var x = "local"; return new Function("return x") } print(make()())`, "global\n")},

		// 15.3.3.1: "The initial value of Function.prototype is the built-in Function prototype object"
		{"15.3.3.1 Function.prototype is inherited by functions", output(`Function.prototype.marker = "fp"; function f() {} print(f.marker, new Function("").marker)`, "fp fp\n")},
		// 15.3.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.3.3.1 Function.prototype cannot be replaced or deleted",
			output(`var original = Function.prototype; Function.prototype = null; print(delete Function.prototype, Function.prototype == original)`, "false true\n")},
		// 15.3.3.2: "The length property is 1."
		{"15.3.3.2 Function.length is 1", output(`print(Function.length)`, "1\n")},

		// 15.3.4: "The Function prototype object is itself a Function object ... that, when invoked, accepts any arguments and returns undefined."
		{"15.3.4 Function.prototype is a function returning undefined", output(`print(typeof Function.prototype, Function.prototype(), Function.prototype(1, 2))`, "function undefined undefined\n")},
		// 15.3.4: "The value of the internal [[Prototype]] property of the Function prototype object is the Object prototype object"
		{"15.3.4 Function.prototype inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(Function.prototype.marker)`, "op\n")},
		// 15.3.4: "it inherits the valueOf property from the Object prototype Object"
		{"15.3.4 functions inherit valueOf from Object.prototype", output(`function f() {} print(Function.prototype.valueOf == Object.prototype.valueOf, f.valueOf() == f)`, "true true\n")},
		// 15.3.4.1: "The initial value of Function.prototype.constructor is the built-in Function constructor."
		{"15.3.4.1 Function.prototype.constructor is Function", output(`function f() {} print(Function.prototype.constructor == Function, f.constructor == Function)`, "true true\n")},
		// 15.3.4.2: "This representation has the syntax of a FunctionDeclaration."
		{"15.3.4.2 toString of a declared function has FunctionDeclaration syntax",
			output(`function f(a, b) { return a * b + 1 } var src = f.toString(); f = null; eval(src); print(f(3, 4))`, "13\n")},
		// 15.3.4.2: the representation of a built-in function has FunctionDeclaration syntax too (README)
		{"15.3.4.2 toString of a built-in function has FunctionDeclaration syntax",
			output(`eval(Math.abs.toString()); eval(parseInt.toString()); print("parsed")`, "parsed\n")},
		// 15.3.4.2: "The toString function is not generic; it generates a runtime error if its this value is not a Function object."
		{"15.3.4.2 toString on an object that is not a function is a runtime error",
			runtimeError(`var o = new Object(); o.toString = Function.prototype.toString; print("before"); o.toString()`, "before\n")},

		// 15.3.5: "Every function instance has a [[Call]] property and a [[Construct]] property."
		{"15.3.5 every function can be called and constructed", output(`function f() { return 1 } var g = new Function("return 2"); print(f(), g(), typeof new f(), typeof new g())`, "1 2 object object\n")},
		// 15.3.5.1: "the language permits the function to be invoked with some other number of arguments"
		{"15.3.5.1 length is the declared number of arguments, not a limit", output(`function f(a, b) { return arguments.length } print(f.length, f(1), f(1, 2, 3))`, "2 1 3\n")},
		// 15.3.5.2: "The value of the prototype property is used to initialize the internal [[Prototype]] property of a newly created object"
		{"15.3.5.2 objects created by new inherit from the prototype property", output(`function F() {} F.prototype.greet = "hi"; var o = new F(); print(o.greet, typeof F.prototype)`, "hi object\n")},
		// 15.3.5.2: the prototype of a declared function points back to it (§13 creates it as in §15.3.2.1)
		{"15.3.5.2 the prototype of a declared function has a constructor property", output(`function F() {} print(F.prototype.constructor == F, new F().constructor == F)`, "true true\n")},
		// 15.3.5.3: "The value of the arguments property is normally null if there is no outstanding invocation of the function in progress"
		{"15.3.5.3 arguments is null when the function is not running", output(`function f() {} print(f.arguments == null); f(1); print(f.arguments == null)`, "true\n"+"true\n")},
		// 15.3.5.3: "its arguments property is dynamically bound to a newly created object that contains the arguments"
		{"15.3.5.3 arguments holds the arguments while the function runs", output(`function f(a) { return f.arguments.length + " " + f.arguments[1] } print(f(1, "two"))`, "2 two\n")},
	})
}

// §15.4, Array objects. ES1 has no array literals; arrays are built with
// new Array and element assignment.

func TestS15_4_ArrayObjects(t *testing.T) {
	runRows(t, []row{
		// 15.4: "whenever a property is added whose name is an array index, the length property is changed ... to be one more than the numeric value of that array index"
		{"15.4 adding an element past the end grows length", output(`var a = new Array(); a[0] = "x"; a[9] = "y"; print(a.length, a[5])`, "10 undefined\n")},
		// 15.4: "whenever the length property is changed, every property whose name is an array index whose value is not smaller than the new length is automatically deleted"
		{"15.4 shrinking length deletes the elements past it", output(`var a = new Array("a", "b", "c", "d"); a.length = 2; print(a.length, a[1], a[2], a[3])`, "2 b undefined undefined\n")},
		// 15.4: "A property name P ... is an array index if and only if ToString(ToUint32(P)) is equal to P"
		{"15.4 a property name that is not a canonical integer is not an index", output(`var a = new Array(); a["1.0"] = 1; a["01"] = 2; a["-1"] = 3; a.x = 4; print(a.length)`, "0\n")},
		// 15.4: "and ToUint32(P) is not equal to 2^32-1"
		{"15.4 4294967295 is not an array index", output(`var a = new Array(); a[4294967295] = 1; print(a.length); a[4294967294] = 2; print(a.length)`, "0\n4294967295\n")},
		// 15.4: a numeric property name is converted with ToString, so a[1] and a["1"] are the same index
		{"15.4 a number and its string name the same element", output(`var a = new Array(); a["2"] = "s"; print(a[2], a.length)`, "s 3\n")},
		// 15.4: "This constraint applies only to properties of the Array object itself"
		{"15.4 inherited index properties do not change length", output(`Array.prototype[5] = "inherited"; var a = new Array(); print(a.length, a[5])`, "0 inherited\n")},

		// 15.4.1: "the function call Array(...) is equivalent to the object creation expression new Array(...)"
		{"15.4.1 Array called as a function creates an array", output(classOfJS+`var a = Array(); print(classOf(a), a.length)`, "[object Array] 0\n")},
		// 15.4.1.1: "An array is created and returned as if by the expression new Array(item0, item1, . . .)."
		{"15.4.1.1 Array(item0, item1, ...) holds the items", output(`var a = Array("x", "y", "z"); print(a.length, a[0], a[2])`, "3 x z\n")},
		// 15.4.1.2: "An array is created and returned as if by the expression new Array(len)."
		{"15.4.1.2 Array(len) has that length", output(`var a = Array(4); print(a.length, a[0])`, "4 undefined\n")},
		// 15.4.1.3: "An array is created and returned as if by the expression new Array()."
		{"15.4.1.3 Array() is empty", output(`print(Array().length)`, "0\n")},

		// 15.4.2.1: "The length property of the newly constructed object is set to the number of arguments."
		{"15.4.2.1 new Array with two or more arguments holds them in order",
			output(`var a = new Array(1, "two", null); print(a.length, a[0], a[1], a[2])`, "3 1 two null\n")},
		// 15.4.2.1: "The [[Class]] property of the newly constructed object is set to "Array"."
		{"15.4.2.1 new Array creates an object of class Array", output(classOfJS+`print(classOf(new Array(1, 2)))`, "[object Array]\n")},
		// 15.4.2.1: "The [[Prototype]] property of the newly constructed object is set to the original Array prototype object"
		{"15.4.2.1 new Array inherits from Array.prototype", output(`Array.prototype.marker = "ap"; print(new Array(1, 2).marker)`, "ap\n")},
		// 15.4.2.2: "If the argument len is a number, then the length property of the newly constructed object is set to ToUint32(len)."
		{"15.4.2.2 new Array(number) sets the length", output(`var a = new Array(3); var n = 0; for (var p in a) n++; print(a.length, n)`, "3 0\n")},
		// 15.4.2.2: ToUint32 of a length that is not a valid uint32
		{"15.4.2.2 new Array converts a numeric length with ToUint32", output(`print(new Array(-1).length, new Array(2.5).length)`, "4294967295 2\n")},
		// 15.4.2.2: "If the argument len is not a number, then the length property ... is set to 1 and the 0 property ... is set to len."
		{"15.4.2.2 new Array with one argument that is not a number holds it", output(`var a = new Array("3"); print(a.length, a[0], typeof a[0])`, "1 3 string\n")},
		// 15.4.2.3: "The length property of the newly constructed object is set to +0."
		{"15.4.2.3 new Array() is empty", output(classOfJS+`var a = new Array(); print(a.length, 1 / a.length, classOf(a))`, "0 Infinity [object Array]\n")},

		// 15.4.3: "The value of the internal [[Prototype]] property of the Array constructor is the Function prototype object."
		{"15.4.3 the Array constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(Array.marker)`, "fp\n")},
		// 15.4.3.1: "The initial value of Array.prototype is the built-in Array prototype object"
		{"15.4.3.1 Array.prototype provides the array methods", output(`print(new Array().join == Array.prototype.join)`, "true\n")},
		// 15.4.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.4.3.1 Array.prototype cannot be replaced or deleted", output(`var original = Array.prototype; Array.prototype = null; print(delete Array.prototype, Array.prototype == original)`, "false true\n")},
		// 15.4.3.2: "The length property is 1."
		{"15.4.3.2 Array.length is 1", output(`print(Array.length)`, "1\n")},

		// 15.4.4: "the Array prototype object is itself an array; it has a length property (whose initial value is +0)"
		{"15.4.4 Array.prototype is an empty array", output(`print(Array.prototype.length)`, "0\n")},
		// 15.4.4: "The value of the internal [[Prototype]] property of the Array prototype object is the Object prototype object"
		{"15.4.4 Array.prototype inherits valueOf from Object.prototype", output(`var a = new Array(); print(Array.prototype.valueOf == Object.prototype.valueOf, a.valueOf() == a)`, "true true\n")},
		// 15.4.4.1: "The initial value of Array.prototype.constructor is the built-in Array constructor."
		{"15.4.4.1 Array.prototype.constructor is Array", output(`print(Array.prototype.constructor == Array, new Array().constructor == Array)`, "true true\n")},
		// 15.4.4.2: "The result is the same as if the built-in join method were invoked for this object with no argument."
		{"15.4.4.2 toString joins the elements with commas", output(`var a = new Array(1, "b", null); a[4] = 5; print(a.toString(), "" + a, String(new Array()))`, "1,b,,,5 1,b,,,5 \n")},

		// 15.4.4.3 step 3: "If separator is not supplied, let separator be the single-character string ","."
		{"15.4.4.3 join without a separator uses a comma", output(`print(new Array(1, 2, 3).join())`, "1,2,3\n")},
		// 15.4.4.3 steps 4 and 11: "Call ToString(separator)." and concatenate it between elements
		{"15.4.4.3 join puts the separator between elements", output(`var a = new Array("a", "b", "c"); print(a.join("-"), a.join(""), a.join(", "), a.join(0))`, "a-b-c abc a, b, c a0b0c\n")},
		// 15.4.4.3 step 5: "If Result(2) is 0, return the empty string."
		{"15.4.4.3 join of an empty array is the empty string", output(`print("[" + new Array().join("-") + "]")`, "[]\n")},
		// 15.4.4.3 steps 7 and 13: "If Result(6) is undefined or null, use the empty string; otherwise, call ToString"
		{"15.4.4.3 join writes undefined and null as empty strings", output(`var a = new Array(void 0, null, 0, false); a[5] = "end"; print(a.join("|"))`, "||0|false||end\n")},
		// 15.4.4.3 steps 1-2: "Call the [[Get]] method of this object with argument "length"." "Call ToUint32(Result(1))."
		{"15.4.4.3 join is generic and uses the length property",
			output(`var o = new Object(); o.length = "3"; o[0] = "x"; o[2] = "z"; o[3] = "ignored"; o.join = Array.prototype.join; print(o.join("+"))`, "x++z\n")},
		// 15.4.4.3: each element is converted with ToString (§9.8), so objects use their toString
		{"15.4.4.3 join converts elements with ToString", output(`var inner = new Array(1, 2); var a = new Array(inner, "x"); print(a.join(";"))`, "1,2;x\n")},

		// 15.4.4.4: "The elements of the array are rearranged so as to reverse their order. This object is returned"
		{"15.4.4.4 reverse reverses the array in place and returns it",
			output(`var a = new Array(1, 2, 3, 4); var r = a.reverse(); print(a.join(), r == a)`, "4,3,2,1 true\n")},
		// 15.4.4.4 step 3: "Compute floor(Result(2)/2)." The middle element of an odd length stays.
		{"15.4.4.4 reverse of an odd number of elements keeps the middle one", output(`var a = new Array("a", "b", "c"); a.reverse(); print(a.join())`, "c,b,a\n")},
		// 15.4.4.4 step 5: with length 0 or 1, nothing changes
		{"15.4.4.4 reverse of a short array changes nothing", output(`var a = new Array(); a.reverse(); var b = new Array(1); b[0] = "only"; b.reverse(); print(a.length, b[0])`, "0 only\n")},
		// 15.4.4.4: "the reverse function is intentionally generic"
		{"15.4.4.4 reverse is generic",
			output(`var o = new Object(); o.length = 3; o[0] = "a"; o[1] = "b"; o[2] = "c"; o.reverse = Array.prototype.reverse; o.reverse(); print(o[0], o[1], o[2])`, "c b a\n")},

		// 15.4.4.5 SortCompare steps 7-11: without comparefn, elements are compared as strings
		{"15.4.4.5 sort without a comparison function sorts by string value", output(`var a = new Array(10, 9, 1, 100); a.sort(); print(a.join())`, "1,10,100,9\n")},
		// 15.4.4.5 SortCompare steps 9-11: string comparison uses character codes (§11.8.5)
		{"15.4.4.5 sort orders strings by character codes", output(`var a = new Array("b", "a", "C", "B"); a.sort(); print(a.join())`, "B,C,a,b\n")},
		// 15.4.4.5 SortCompare steps 5-6: "Call comparefn with arguments x and y." "Return Result(5)."
		{"15.4.4.5 sort uses the comparison function", output(`function byNumber(x, y) { return x - y } var a = new Array(10, 9, 1, 100); a.sort(byNumber); print(a.join())`, "1,9,10,100\n")},
		// 15.4.4.5: comparefn "returns a negative value if x < y, zero if x = y, or a positive value if x > y"
		{"15.4.4.5 sort follows the sign returned by the comparison function", output(`function desc(x, y) { return y - x } var a = new Array(3, 1, 2); a.sort(desc); print(a.join())`, "3,2,1\n")},
		// 15.4.4.5 SortCompare steps 1-3: "undefined always compared greater than any other value"
		{"15.4.4.5 sort puts undefined last", output(`var a = new Array("b", void 0, "a", void 0, "c"); a.sort(); print(a[0], a[1], a[2], a[3], a[4], a.length)`, "a b c undefined undefined 5\n")},
		// 15.4.4.5 SortCompare steps 1-3 come before comparefn, so it never sees undefined
		{"15.4.4.5 the comparison function is never given undefined",
			output(`var seen = 0; function cmp(x, y) { if (x == void 0 || y == void 0) seen++; return x - y } var a = new Array(2, void 0, 1); a.sort(cmp); print(a[0], a[1], a[2], seen)`, "1 2 undefined 0\n")},
		// 15.4.4.5 note: "undefined and nonexistent property values always sort to the end"; whether they exist afterwards is implementation-dependent
		{"15.4.4.5 sort puts missing elements last", output(`var a = new Array(4); a[0] = "c"; a[2] = "a"; a[3] = "b"; a.sort(); print(a[0], a[1], a[2], a[3], a.length)`, "a b c undefined 4\n")},
		// 15.4.4.5: old[j] of a missing element is the result of [[Get]], undefined, so SortCompare steps 1-3 keep it from comparefn
		{"15.4.4.5 the comparison function is never given a missing element",
			output(`var seen = 0; function cmp(x, y) { if (x == void 0 || y == void 0) seen++; return x - y } var a = new Array(4); a[1] = 3; a[3] = 1; a.sort(cmp); print(a[0], a[1], a[2], a[3], seen, a.length)`, "1 3 undefined undefined 0 4\n")},
		// 15.4.4.5 step 3 (1): the result is a permutation of the elements
		{"15.4.4.5 sort keeps every element", output(`var a = new Array(3, 1, 2, 3, 1); a.sort(); print(a.join(), a.length)`, "1,1,2,3,3 5\n")},
		// 15.4.4.5 step 4: "Return this object."
		{"15.4.4.5 sort returns the array itself", output(`var a = new Array(2, 1); print(a.sort() == a)`, "true\n")},
		// 15.4.4.5: "the sort function is intentionally generic"
		{"15.4.4.5 sort is generic",
			output(`var o = new Object(); o.length = 3; o[0] = "c"; o[1] = "a"; o[2] = "b"; o.sort = Array.prototype.sort; o.sort(); print(o[0], o[1], o[2])`, "a b c\n")},
		// 15.4.4.5 SortCompare steps 7-8: elements are converted with ToString before comparing
		{"15.4.4.5 sort compares objects through their string values",
			output(`function s1() { return "b" } function s2() { return "a" } var x = new Object(); x.toString = s1; var y = new Object(); y.toString = s2; var a = new Array(x, y); a.sort(); print(a[0] == y, a[1] == x)`, "true true\n")},

		// 15.4.5.1 step 7 then 10: "Change (or set) the value of the length property of A to ToUint32(P)+1."
		{"15.4.5.1 putting an element at or past length sets length to index + 1", output(`var a = new Array(); a[0] = 1; print(a.length); a[2] = 1; print(a.length)`, "1\n3\n")},
		// 15.4.5.1 step 9: "if ToUint32(P) is less than the value of the length property of A, then return"
		{"15.4.5.1 putting an element below length keeps length", output(`var a = new Array(5); a[1] = "x"; print(a.length)`, "5\n")},
		// 15.4.5.1 step 8: "If P is not an array index, return."
		{"15.4.5.1 putting a property that is not an index keeps length", output(`var a = new Array(1, 2); a.name = "n"; a["1.5"] = 3; print(a.length, a.name)`, "2 n\n")},
		// 15.4.5.1 steps 12-14: "Compute ToUint32(V)." delete elements at or above it, then set length
		{"15.4.5.1 setting length deletes the elements at or above it", output(`var a = new Array("a", "b", "c"); a.length = 1; a.length = 3; print(a.join("|"), a[1], a[2])`, "a|| undefined undefined\n")},
		// 15.4.5.1 step 12: the new length is converted with ToUint32
		{"15.4.5.1 length is converted with ToUint32", output(`var a = new Array(1, 2, 3); a.length = "2"; print(a.length, typeof a.length)`, "2 number\n")},
		// 15.4.5.1 step 12: ToUint32 of a value that is not a valid length
		{"15.4.5.1 a fractional length is truncated", output(`var a = new Array(1, 2, 3); a.length = 2.5; print(a.length, a[1], a[2])`, "2 2 undefined\n")},
		// 15.4.5.1 step 14: setting length higher creates no elements
		{"15.4.5.1 growing length creates no elements", output(`var a = new Array(); a.length = 3; var n = 0; for (var p in a) n++; print(a.length, n)`, "3 0\n")},

		// 15.4.5.2: "The length property has the attributes { DontEnum, DontDelete }."
		{"15.4.5.2 length is neither enumerated nor deletable",
			output(`var a = new Array(1, 2); var found = ""; for (var p in a) if (p == "length") found += p; print(delete a.length, a.length, "[" + found + "]")`, "false 2 []\n")},
		// 15.4.5.2: "The length property of this Array object is always numerically greater than the name of every property whose name is an array index."
		{"15.4.5.2 length is always greater than every index", output(`var a = new Array(); a[7] = 1; a[3] = 1; print(a.length)`, "8\n")},
	})
}

// §15.5, String objects.

func TestS15_5_StringObjects(t *testing.T) {
	runRows(t, []row{
		// 15.5.1.1: "Returns a string value (not a String object) computed by ToString(value)."
		{"15.5.1.1 String(value) converts to a string value", output(`print(typeof String(12), String(12), String(null), String(true), String(new Object()))`, "string 12 null true [object Object]\n")},
		// 15.5.1.2: "Returns the empty string """
		{"15.5.1.2 String() is the empty string", output(`print(typeof String(), String().length)`, "string 0\n")},

		// 15.5.2.1: "The [[Value]] property of the newly constructed object is set to ToString(value)."
		{"15.5.2.1 new String(value) wraps the string value", output(classOfJS+`var s = new String(12); print(typeof s, s.valueOf(), typeof s.valueOf(), classOf(s))`, "object 12 string [object String]\n")},
		// 15.5.2.1: "The [[Prototype]] property of the newly constructed object is set to the original String prototype object"
		{"15.5.2.1 new String inherits from String.prototype", output(`String.prototype.marker = "sp"; print(new String("a").marker, "a".marker)`, "sp sp\n")},
		// 15.5.2.2: "The [[Value]] property of the newly constructed object is set to the empty string."
		{"15.5.2.2 new String() wraps the empty string", output(`var s = new String(); print(typeof s, s.length, "[" + s + "]")`, "object 0 []\n")},

		// 15.5.3: "The value of the internal [[Prototype]] property of the String constructor is the Function prototype object."
		{"15.5.3 the String constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(String.marker, String.length)`, "fp 1\n")},
		// 15.5.3.1: "The initial value of String.prototype is the built-in String prototype object"
		{"15.5.3.1 String.prototype provides the string methods", output(`print("a".charAt == String.prototype.charAt)`, "true\n")},
		// 15.5.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.5.3.1 String.prototype cannot be replaced or deleted", output(`var original = String.prototype; String.prototype = null; print(delete String.prototype, String.prototype == original)`, "false true\n")},
		// 15.5.3.2: "Each argument specifies one character of the resulting string ... from left to right"
		{"15.5.3.2 fromCharCode builds a string from character codes", output(`print(String.fromCharCode(72, 105, 33), String.fromCharCode(97).length)`, "Hi! 1\n")},
		// 15.5.3.2: "If no arguments are supplied, the result is the empty string."
		{"15.5.3.2 fromCharCode with no arguments is the empty string", output(`print(String.fromCharCode().length, typeof String.fromCharCode())`, "0 string\n")},
		// 15.5.3.2: "An argument is converted to a character by applying the operation ToUint16"
		{"15.5.3.2 fromCharCode converts its arguments with ToUint16", output(`print(String.fromCharCode(65601, "66", 67.9), String.fromCharCode(-1).charCodeAt(0))`, "ABC 65535\n")},

		// 15.5.4: "The String prototype object is itself a String object ... whose value is an empty string."
		{"15.5.4 String.prototype is the empty string", output(classOfJS+`print("[" + String.prototype.toString() + "]", classOf(String.prototype))`, "[] [object String]\n")},
		// 15.5.4: "The value of the internal [[Prototype]] property of the String prototype object is the Object prototype object"
		{"15.5.4 String.prototype inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(String.prototype.marker)`, "op\n")},
		// 15.5.4.1: "The initial value of String.prototype.constructor is the built-in String constructor."
		{"15.5.4.1 String.prototype.constructor is String", output(`print(String.prototype.constructor == String, "a".constructor == String)`, "true true\n")},
		// 15.5.4.2: "Returns this string value."
		{"15.5.4.2 toString returns the string value", output(`var s = new String("abc"); print(s.toString(), typeof s.toString(), "xy".toString())`, "abc string xy\n")},
		// 15.5.4.2: "it generates a runtime error if its this value is not a String object"
		{"15.5.4.2 toString on an object that is not a String is a runtime error",
			runtimeError(`var o = new Object(); o.toString = String.prototype.toString; print("before"); o.toString()`, "before\n")},
		// 15.5.4.3: "Returns this string value."
		{"15.5.4.3 valueOf returns the string value", output(`var s = new String("abc"); print(s.valueOf(), typeof s.valueOf())`, "abc string\n")},
		// 15.5.4.3: "it generates a runtime error if its this value is not a String object"
		{"15.5.4.3 valueOf on an object that is not a String is a runtime error",
			runtimeError(`var o = new Number(1); o.valueOf = String.prototype.valueOf; print("before"); o.valueOf()`, "before\n")},

		// 15.5.4.4 step 5: "Return a string of length 1, containing one character from Result(1)"
		{"15.5.4.4 charAt returns the character at a position", output(`var s = "abc"; print(s.charAt(0), s.charAt(2), typeof s.charAt(1))`, "a c string\n")},
		// 15.5.4.4 step 4: "If Result(2) is less than 0 or is not less than Result(3), return the empty string."
		{"15.5.4.4 charAt outside the string is the empty string", output(`var s = "abc"; print("[" + s.charAt(-1) + s.charAt(3) + s.charAt(100) + "]")`, "[]\n")},
		// 15.5.4.4 step 2: "Call ToInteger(pos)."
		{"15.5.4.4 charAt converts its position with ToInteger", output(`var s = "abc"; print(s.charAt(), s.charAt(1.7), s.charAt("2"), s.charAt(NaN))`, "a b c a\n")},
		// 15.5.4.4: "If pos is a value of Number type that is an integer, then the result of x.charAt(pos) is equal to the result of x.substring(pos, pos+1)."
		{"15.5.4.4 charAt(pos) equals substring(pos, pos + 1)", output(`var x = "hello"; print(x.charAt(1) == x.substring(1, 2), x.charAt(4) == x.substring(4, 5))`, "true true\n")},
		// 15.5.4.4: "the charAt function is intentionally generic"
		{"15.5.4.4 charAt is generic", output(`var o = new Object(); o.charAt = String.prototype.charAt; var n = new Number(123); n.charAt = String.prototype.charAt; print(o.charAt(1), n.charAt(2))`, "o 3\n")},

		// 15.5.4.5 step 5: "Return a value of Number type ... the Unicode encoding of one character"
		{"15.5.4.5 charCodeAt returns the character code at a position", output(`print("Ab".charCodeAt(0), "Ab".charCodeAt(1), "\uffff".charCodeAt(0))`, "65 98 65535\n")},
		// 15.5.4.5 step 4: "If Result(2) is less than 0 or is not less than Result(3), return NaN."
		{"15.5.4.5 charCodeAt outside the string is NaN", output(`print("ab".charCodeAt(-1), "ab".charCodeAt(2), "".charCodeAt(0))`, "NaN NaN NaN\n")},
		// 15.5.4.5 step 2: "Call ToInteger(pos)."
		{"15.5.4.5 charCodeAt converts its position with ToInteger", output(`print("ab".charCodeAt(), "ab".charCodeAt(1.9), "ab".charCodeAt("1"))`, "97 98 98\n")},
		// 15.5.4.5: "the charCodeAt function is intentionally generic"
		{"15.5.4.5 charCodeAt is generic", output(`var b = new Boolean(true); b.charCodeAt = String.prototype.charCodeAt; print(b.charCodeAt(0))`, "116\n")},

		// 15.5.4.6 step 7: "Compute the smallest possible integer k not smaller than Result(5) such that" the search string occurs at k
		{"15.5.4.6 indexOf finds the first occurrence", output(`var s = "abcabc"; print(s.indexOf("b"), s.indexOf("bc"), s.indexOf("abc"), s.indexOf("x"))`, "1 1 0 -1\n")},
		// 15.5.4.6 steps 3 and 5: the search starts at min(max(position, 0), length)
		{"15.5.4.6 indexOf starts at the given position", output(`var s = "abcabc"; print(s.indexOf("b", 2), s.indexOf("b", 5), s.indexOf("a", -5), s.indexOf("a", 100))`, "4 -1 0 -1\n")},
		// 15.5.4.6 step 7: an empty search string is found at the start position
		{"15.5.4.6 indexOf of the empty string is the start position", output(`var s = "abc"; print(s.indexOf(""), s.indexOf("", 2), s.indexOf("", 10))`, "0 2 3\n")},
		// 15.5.4.6 steps 1-3: "Call ToString(searchString)." "Call ToInteger(position)."
		{"15.5.4.6 indexOf converts its arguments", output(`print("a1b2".indexOf(2), "undefined".indexOf(), "abcabc".indexOf("c", "3"), "abc".indexOf("c", 1.9))`, "3 0 5 2\n")},
		// 15.5.4.6: "the indexOf function is intentionally generic"
		{"15.5.4.6 indexOf is generic", output(`var n = new Number(12345); n.indexOf = String.prototype.indexOf; print(n.indexOf("34"))`, "2\n")},

		// 15.5.4.7 step 8: "Compute the largest possible integer k not larger than Result(6) such that" the search string occurs at k
		{"15.5.4.7 lastIndexOf finds the last occurrence", output(`var s = "abcabc"; print(s.lastIndexOf("b"), s.lastIndexOf("abc"), s.lastIndexOf("x"))`, "4 3 -1\n")},
		// 15.5.4.7 steps 4 and 6: the search starts at min(max(position, 0), length), going left
		{"15.5.4.7 lastIndexOf searches left from the given position", output(`var s = "abcabc"; print(s.lastIndexOf("b", 3), s.lastIndexOf("b", 4), s.lastIndexOf("a", -1), s.lastIndexOf("b", 0), s.lastIndexOf("c", 100))`, "1 4 0 -1 5\n")},
		// 15.5.4.7 step 4: "If Result(3) is NaN, use +Infinity"
		{"15.5.4.7 lastIndexOf with a NaN position searches the whole string", output(`var s = "abcabc"; print(s.lastIndexOf("a", NaN), s.lastIndexOf("a", "x"), s.lastIndexOf("a", void 0))`, "3 3 3\n")},
		// 15.5.4.7 step 8: an empty search string is found at the start position
		{"15.5.4.7 lastIndexOf of the empty string is the start position", output(`var s = "abc"; print(s.lastIndexOf(""), s.lastIndexOf("", 1))`, "3 1\n")},
		// 15.5.4.7: "the lastIndexOf function is intentionally generic"
		{"15.5.4.7 lastIndexOf is generic", output(`var n = new Number(12121); n.lastIndexOf = String.prototype.lastIndexOf; print(n.lastIndexOf("12"))`, "2\n")},

		// 15.5.4.8 steps 9-16: substrings between the occurrences of the separator
		{"15.5.4.8 split divides a string at each separator",
			output(classOfJS+`var a = "a,b,c".split(","); print(a.length, a[0], a[1], a[2], classOf(a))`, "3 a b c [object Array]\n")},
		// 15.5.4.8 step 10: adjacent and edge separators give empty strings
		{"15.5.4.8 split keeps empty substrings", output(`var a = ",a,,b,".split(","); print(a.length, "[" + a.join("|") + "]")`, "5 [|a||b|]\n")},
		// 15.5.4.8: "The separator may be a string of any length."
		{"15.5.4.8 split with a longer separator", output(`var a = "a--b--c".split("--"); print(a.length, a.join("|"))`, "3 a|b|c\n")},
		// 15.5.4.8 steps 9 and 14: a separator that does not occur gives the whole string
		{"15.5.4.8 split with a separator that does not occur gives one element", output(`var a = "abc".split("x"); print(a.length, a[0])`, "1 abc\n")},
		// 15.5.4.8 steps 3-4: only a separator that is not supplied is special; an undefined one is converted by ToString to "undefined"
		{"15.5.4.8 split with an undefined separator splits at \"undefined\"", output(`var a = "aundefinedb".split(void 0); print(a.length, a.join("|"))`, "2 a|b\n")},
		// 15.5.4.8 step 3: "If separator is not supplied, call the [[Put]] method of A with 0 and Result(1) as arguments, and then return A."
		{"15.5.4.8 split without a separator gives the whole string", output(`var a = "a,b".split(); print(a.length, a[0])`, "1 a,b\n")},
		// 15.5.4.8 steps 17-21: "if the separator is the empty string, the string is split up into individual characters"
		{"15.5.4.8 split with the empty separator gives the characters", output(`var a = "abc".split(""); print(a.length, a.join("|"))`, "3 a|b|c\n")},
		// 15.5.4.8 step 17: an empty string split by the empty separator gives no elements
		{"15.5.4.8 the empty string split by the empty separator is empty", output(`print("".split("").length)`, "0\n")},
		// 15.5.4.8 step 14: the empty string split by a separator gives one empty string
		{"15.5.4.8 the empty string split by a separator gives one empty string", output(`var a = "".split(","); print(a.length, "[" + a[0] + "]")`, "1 []\n")},
		// 15.5.4.8 step 4: "Call ToString(separator)."
		{"15.5.4.8 split converts the separator to a string", output(`var a = "1a1b".split(1); print(a.length, "[" + a.join("|") + "]")`, "3 [|a|b]\n")},
		// 15.5.4.8: "the split function is intentionally generic"
		{"15.5.4.8 split is generic", output(`var n = new Number(10203); n.split = String.prototype.split; print(n.split("0").join("|"))`, "1|2|3\n")},

		// 15.5.4.9 step 5: the characters from min(max(start, 0), length) to the end
		{"15.5.4.9 substring(start) runs to the end of the string", output(`var s = "abcdef"; print(s.substring(2), s.substring(0), typeof s.substring(1))`, "cdef abcdef string\n")},
		// 15.5.4.9: "If the argument is NaN or negative, it is replaced with zero; if the argument is larger than the length of the string, it is replaced with the length"
		{"15.5.4.9 substring(start) clamps start to the string", output(`var s = "abc"; print(s.substring(-2), s.substring(NaN), "[" + s.substring(10) + "]")`, "abc abc []\n")},
		// 15.5.4.9 step 2: "Call ToInteger(start)."
		{"15.5.4.9 substring(start) converts start with ToInteger", output(`print("abcdef".substring(2.9), "abcdef".substring("4"))`, "cdef ef\n")},
		// 15.5.4.10 step 9: the characters from min(start, end) to max(start, end) - 1
		{"15.5.4.10 substring(start, end) returns the characters between", output(`var s = "abcdef"; print(s.substring(1, 4), "[" + s.substring(2, 2) + "]")`, "bcd []\n")},
		// 15.5.4.10: "If start is larger than end, they are swapped."
		{"15.5.4.10 substring swaps start and end", output(`print("abcdef".substring(4, 1))`, "bcd\n")},
		// 15.5.4.10: "If either argument is NaN or negative, it is replaced with zero; if ... larger than the length ... replaced with the length"
		{"15.5.4.10 substring clamps both positions", output(`var s = "abcdef"; print(s.substring(-1, 2), s.substring(2, NaN), s.substring(1, 100), s.substring(100, -100))`, "ab ab bcdef abcdef\n")},
		// 15.5.4.10 steps 2-3: ToInteger of both positions
		{"15.5.4.10 substring converts both positions with ToInteger", output(`print("abcdef".substring(1.5, "3.9"))`, "bc\n")},
		// 15.5.4.10: "the substring function is intentionally generic"
		{"15.5.4.10 substring is generic", output(`var n = new Number(12345); n.substring = String.prototype.substring; print(n.substring(1, 3))`, "23\n")},

		// 15.5.4.11: "unless that character has a Unicode 2.0 lowercase equivalent, in which case the lowercase equivalent is used"
		{"15.5.4.11 toLowerCase converts letters to lower case", output(`print("Hello, World! 123".toLowerCase(), typeof "A".toLowerCase())`, "hello, world! 123 string\n")},
		// 15.5.4.11: "The canonical Unicode 2.0 case mapping shall be used" beyond ASCII
		{"15.5.4.11 toLowerCase uses the Unicode case mapping", output(`print("\u00C9".toLowerCase().charCodeAt(0), "\u0391".toLowerCase().charCodeAt(0), "\u0416".toLowerCase().charCodeAt(0))`, "233 945 1078\n")},
		// 15.5.4.11: "the toLowerCase function is intentionally generic"
		{"15.5.4.11 toLowerCase is generic", output(`var b = new Boolean(true); b.toLowerCase = String.prototype.toLowerCase; print(b.toLowerCase())`, "true\n")},
		// 15.5.4.12: "unless that character has a Unicode 2.0 uppercase equivalent, in which case the uppercase equivalent is used"
		{"15.5.4.12 toUpperCase converts letters to upper case", output(`print("Hello, World! 123".toUpperCase())`, "HELLO, WORLD! 123\n")},
		// 15.5.4.12: "The canonical Unicode 2.0 case mapping shall be used" beyond ASCII
		{"15.5.4.12 toUpperCase uses the Unicode case mapping", output(`print("\u00e9".toUpperCase().charCodeAt(0), "\u03b1".toUpperCase().charCodeAt(0), "\u0436".toUpperCase().charCodeAt(0))`, "201 913 1046\n")},
		// 15.5.4.12: "the toUpperCase function is intentionally generic"
		{"15.5.4.12 toUpperCase is generic", output(`var o = new Object(); o.toUpperCase = String.prototype.toUpperCase; print(o.toUpperCase())`, "[OBJECT OBJECT]\n")},

		// 15.5.5: "String instances inherit properties from the String prototype object and also have a [[Value]] property"
		{"15.5.5 a String object holds its string value", output(`var s = new String("abc"); print(s + "!", s == "abc", s.indexOf("c"))`, "abc! true 2\n")},
		// 15.5.5.1: "The number of characters in the String value represented by this String object."
		{"15.5.5.1 length is the number of characters", output(`print(new String("abc").length, "".length, "a\nb".length)`, "3 0 3\n")},
		// 15.5.5.1: "this property is unchanging. It has the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.5.5.1 length cannot be changed, deleted or enumerated",
			output(`var s = new String("abc"); s.length = 1; var deleted = delete s.length; var found = ""; for (var p in s) if (p == "length") found += p; print(s.length, deleted, "[" + found + "]")`, "3 false []\n")},
	})
}

// §15.6, Boolean objects.

func TestS15_6_BooleanObjects(t *testing.T) {
	runRows(t, []row{
		// 15.6.1.1: "Returns a boolean value (not a Boolean object) computed by ToBoolean(value)."
		{"15.6.1.1 Boolean(value) converts to a boolean value",
			output(`print(typeof Boolean(1), Boolean(1), Boolean(0), Boolean(""), Boolean("false"), Boolean(null), Boolean(new Boolean(false)))`, "boolean true false false true false true\n")},
		// 15.6.1.2: "Returns false."
		{"15.6.1.2 Boolean() is false", output(`print(Boolean(), typeof Boolean())`, "false boolean\n")},

		// 15.6.2.1: "The [[Value]] property of the newly constructed Boolean object is set to ToBoolean(value)."
		{"15.6.2.1 new Boolean(value) wraps the converted value",
			output(classOfJS+`var b = new Boolean("x"); print(typeof b, b.valueOf(), new Boolean(0).valueOf(), classOf(b))`, "object true false [object Boolean]\n")},
		// 15.6.2.1: "The [[Prototype]] property of the newly constructed object is set to the original Boolean prototype object"
		{"15.6.2.1 new Boolean inherits from Boolean.prototype", output(`Boolean.prototype.marker = "bp"; print(new Boolean(true).marker, true.marker)`, "bp bp\n")},
		// 15.6.2.1 with §9.2: a Boolean object is an object, so it converts to true even when its value is false
		{"15.6.2.1 a Boolean object holding false is still true in a condition", output(`if (new Boolean(false)) print("taken")`, "taken\n")},
		// 15.6.2.2: "The [[Value]] property of the newly constructed Boolean object is set to false."
		{"15.6.2.2 new Boolean() wraps false", output(`var b = new Boolean(); print(typeof b, b.valueOf())`, "object false\n")},

		// 15.6.3: "The value of the internal [[Prototype]] property of the Boolean constructor is the Function prototype object."
		{"15.6.3 the Boolean constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(Boolean.marker, Boolean.length)`, "fp 1\n")},
		// 15.6.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.6.3.1 Boolean.prototype cannot be replaced or deleted", output(`var original = Boolean.prototype; Boolean.prototype = null; print(delete Boolean.prototype, Boolean.prototype == original)`, "false true\n")},

		// 15.6.4: "The Boolean prototype object is itself a Boolean object (its [[Class]] is "Boolean") whose value is false."
		{"15.6.4 Boolean.prototype is a Boolean object holding false", output(classOfJS+`print(Boolean.prototype.valueOf(), Boolean.prototype.toString(), classOf(Boolean.prototype))`, "false false [object Boolean]\n")},
		// 15.6.4: "The value of the internal [[Prototype]] property of the Boolean prototype object is the Object prototype object"
		{"15.6.4 Boolean.prototype inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(Boolean.prototype.marker)`, "op\n")},
		// 15.6.4.1: "The initial value of Boolean.prototype.constructor is the built-in Boolean constructor."
		{"15.6.4.1 Boolean.prototype.constructor is Boolean", output(`print(Boolean.prototype.constructor == Boolean, true.constructor == Boolean)`, "true true\n")},
		// 15.6.4.2: "If this boolean value is true, then the string "true" is returned ... the string "false" is returned."
		{"15.6.4.2 toString gives true or false", output(`print(new Boolean(true).toString(), new Boolean(false).toString(), typeof true.toString(), false.toString())`, "true false string false\n")},
		// 15.6.4.2: "it generates a runtime error if its this value is not a Boolean object"
		{"15.6.4.2 toString on an object that is not a Boolean is a runtime error",
			runtimeError(`var o = new Object(); o.toString = Boolean.prototype.toString; print("before"); o.toString()`, "before\n")},
		// 15.6.4.3: "Returns this boolean value."
		{"15.6.4.3 valueOf returns the boolean value", output(`print(new Boolean(true).valueOf(), typeof new Boolean(true).valueOf(), false.valueOf())`, "true boolean false\n")},
		// 15.6.4.3: "it generates a runtime error if its this value is not a Boolean object"
		{"15.6.4.3 valueOf on an object that is not a Boolean is a runtime error",
			runtimeError(`var o = new String("true"); o.valueOf = Boolean.prototype.valueOf; print("before"); o.valueOf()`, "before\n")},

		// 15.6.5: "Boolean instances have no special properties beyond those inherited from the Boolean prototype object."
		{"15.6.5 a Boolean object has no own properties", output(`var n = 0; for (var p in new Boolean(true)) n++; print(n)`, "0\n")},
	})
}

// §15.7, Number objects.

func TestS15_7_NumberObjects(t *testing.T) {
	runRows(t, []row{
		// 15.7.1.1: "Returns a number value (not a Number object) computed by ToNumber(value)."
		{"15.7.1.1 Number(value) converts to a number value",
			output(`print(typeof Number("12"), Number("12"), Number(""), Number(true), Number(null), Number(void 0), Number(new Number(3)))`, "number 12 0 1 0 NaN 3\n")},
		// 15.7.1.2: "Returns +0."
		{"15.7.1.2 Number() is +0", output(`print(Number(), typeof Number(), 1 / Number())`, "0 number Infinity\n")},

		// 15.7.2.1: "The [[Value]] property of the newly constructed object is set to ToNumber(value)."
		{"15.7.2.1 new Number(value) wraps the converted value",
			output(classOfJS+`var n = new Number("42"); print(typeof n, n.valueOf(), typeof n.valueOf(), n + 1, classOf(n))`, "object 42 number 43 [object Number]\n")},
		// 15.7.2.1: "The [[Prototype]] property of the newly constructed object is set to the original Number prototype object"
		{"15.7.2.1 new Number inherits from Number.prototype", output(`Number.prototype.marker = "np"; print(new Number(1).marker, (5).marker)`, "np np\n")},
		// 15.7.2.2: "The [[Value]] property of the newly constructed object is set to +0."
		{"15.7.2.2 new Number() wraps +0", output(`var n = new Number(); print(typeof n, n.valueOf(), 1 / n.valueOf())`, "object 0 Infinity\n")},

		// 15.7.3: "The value of the internal [[Prototype]] property of the Number constructor is the Function prototype object."
		{"15.7.3 the Number constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(Number.marker, Number.length)`, "fp 1\n")},
		// 15.7.3.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.7.3.1 Number.prototype cannot be replaced or deleted", output(`var original = Number.prototype; Number.prototype = null; print(delete Number.prototype, Number.prototype == original)`, "false true\n")},
		// 15.7.3.2: "the largest positive finite value of the number type"
		{"15.7.3.2 MAX_VALUE is the largest finite number",
			output(`var m = Number.MAX_VALUE; print(m == 1.7976931348623157e308, isFinite(m), m * 2)`, "true true Infinity\n")},
		// 15.7.3.3: "the smallest positive nonzero value of the number type"
		{"15.7.3.3 MIN_VALUE is the smallest positive number",
			output(`var m = Number.MIN_VALUE; print(m > 0, m == 5e-324, m / 2, m == 4.9406564584124654e-324)`, "true true 0 true\n")},
		// 15.7.3.4: "The value of Number.NaN is NaN."
		{"15.7.3.4 Number.NaN is NaN", output(`print(Number.NaN, isNaN(Number.NaN))`, "NaN true\n")},
		// 15.7.3.5: "The value of Number.NEGATIVE_INFINITY is -Infinity."
		{"15.7.3.5 NEGATIVE_INFINITY is negative infinity", output(`print(Number.NEGATIVE_INFINITY, Number.NEGATIVE_INFINITY == -Infinity)`, "-Infinity true\n")},
		// 15.7.3.6: "The value of Number.POSITIVE_INFINITY is +Infinity."
		{"15.7.3.6 POSITIVE_INFINITY is positive infinity", output(`print(Number.POSITIVE_INFINITY, Number.POSITIVE_INFINITY == Infinity)`, "Infinity true\n")},
		// 15.7.3.2-15.7.3.6: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.7.3 the Number constants cannot be changed, deleted or enumerated",
			output(`Number.MAX_VALUE = 1; Number.MIN_VALUE = 1; Number.NaN = 1; Number.NEGATIVE_INFINITY = 1; Number.POSITIVE_INFINITY = 1; `+
				`var d = delete Number.MAX_VALUE || delete Number.MIN_VALUE || delete Number.NaN || delete Number.NEGATIVE_INFINITY || delete Number.POSITIVE_INFINITY; `+
				`var found = ""; for (var p in Number) found += p; `+
				`print(Number.MAX_VALUE > 1, Number.MIN_VALUE < 1, isNaN(Number.NaN), Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY, d, "[" + found + "]")`,
				"true true true -Infinity Infinity false []\n")},

		// 15.7.4: "The Number prototype object is itself a Number object (its [[Class]] is "Number") whose value is +0."
		{"15.7.4 Number.prototype is a Number object holding +0", output(classOfJS+`print(Number.prototype.valueOf(), 1 / Number.prototype.valueOf(), classOf(Number.prototype))`, "0 Infinity [object Number]\n")},
		// 15.7.4: "The value of the internal [[Prototype]] property of the Number prototype object is the Object prototype object"
		{"15.7.4 Number.prototype inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(Number.prototype.marker)`, "op\n")},
		// 15.7.4.1: "The initial value of Number.prototype.constructor is the built-in Number constructor."
		{"15.7.4.1 Number.prototype.constructor is Number", output(`print(Number.prototype.constructor == Number, (1).constructor == Number)`, "true true\n")},
		// 15.7.4.2: "If the radix is the number 10 or not supplied, then this number value is given as an argument to the ToString operator"
		{"15.7.4.2 toString without a radix or with radix 10 is ToString",
			output(`print(new Number(12.5).toString(), (1e21).toString(), (-0.000001).toString(10), NaN.toString(), (-Infinity).toString(10), typeof (1).toString())`, "12.5 1e+21 -0.000001 NaN -Infinity string\n")},
		// 15.7.4.2: "If the radix is supplied and is an integer from 2 to 36, but not 10, the result is a string" (README: positional digits for integers)
		{"15.7.4.2 toString of an integer in another radix gives its digits",
			output(`print((255).toString(16), (255).toString(2), (35).toString(36), (8).toString(8), (0).toString(2), new Number(4095).toString(16))`, "ff 11111111 z 10 0 fff\n")},
		// 15.7.4.2 (README): "a leading - if negative"
		{"15.7.4.2 toString of a negative integer in another radix has a minus sign", output(`print((-255).toString(16), (-5).toString(2))`, "-ff -101\n")},
		// 15.7.4.2: "it generates a runtime error if its this value is not a Number object"
		{"15.7.4.2 toString on an object that is not a Number is a runtime error",
			runtimeError(`var o = new Object(); o.toString = Number.prototype.toString; print("before"); o.toString()`, "before\n")},
		// 15.7.4.3: "Returns this number value."
		{"15.7.4.3 valueOf returns the number value", output(`print(new Number(7).valueOf(), typeof new Number(7).valueOf(), (3).valueOf())`, "7 number 3\n")},
		// 15.7.4.3: "it generates a runtime error if its this value is not a Number object"
		{"15.7.4.3 valueOf on an object that is not a Number is a runtime error",
			runtimeError(`var o = new String("1"); o.valueOf = Number.prototype.valueOf; print("before"); o.valueOf()`, "before\n")},

		// 15.7.5: "Number instances have no special properties beyond those inherited from the Number prototype object."
		{"15.7.5 a Number object has no own properties", output(`var n = 0; for (var p in new Number(1)) n++; print(n)`, "0\n")},
	})
}

// §15.8, the Math object. Exact results are asserted where the spec
// requires them; other results use approx. A signed zero is shown by
// printing 1/x, which is Infinity for +0 and -Infinity for -0.

func TestS15_8_TheMathObject(t *testing.T) {
	runRows(t, []row{
		// 15.8: "The value of the internal [[Prototype]] property of the Math object is the Object prototype object"
		{"15.8 Math inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(Math.marker, typeof Math)`, "op object\n")},
		// 15.8: "The Math object does not have a [[Construct]] property"
		{"15.8 Math cannot be used with new", runtimeError(`print("before"); new Math()`, "before\n")},
		// 15.8: "The Math object does not have a [[Call]] property"
		{"15.8 Math cannot be called", runtimeError(`print("before"); Math()`, "before\n")},

		// 15.8.1.1: "The number value for e"
		{"15.8.1.1 E", approx(`print(Math.E)`, math.E)},
		// 15.8.1.2: "The number value for the natural logarithm of 10"
		{"15.8.1.2 LN10", approx(`print(Math.LN10)`, math.Ln10)},
		// 15.8.1.3: "The number value for the natural logarithm of 2"
		{"15.8.1.3 LN2", approx(`print(Math.LN2)`, math.Ln2)},
		// 15.8.1.4: "The number value for the base-2 logarithm of e"
		{"15.8.1.4 LOG2E", approx(`print(Math.LOG2E)`, math.Log2E)},
		// 15.8.1.5: "approximately 0.4342944819032518" (the base-10 logarithm of e; the spec text repeats 15.8.1.4 by mistake)
		{"15.8.1.5 LOG10E", approx(`print(Math.LOG10E)`, math.Log10E)},
		// 15.8.1.6: "The number value for pi"
		{"15.8.1.6 PI", approx(`print(Math.PI)`, math.Pi)},
		// 15.8.1.7: "The number value for the square root of 1/2"
		{"15.8.1.7 SQRT1_2", approx(`print(Math.SQRT1_2)`, math.Sqrt2/2)},
		// 15.8.1.8: "The number value for the square root of 2"
		{"15.8.1.8 SQRT2", approx(`print(Math.SQRT2)`, math.Sqrt2)},
		// 15.8.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.8.1 the Math constants cannot be changed, deleted or enumerated",
			output(`var before = Math.PI; Math.PI = 3; Math.E = 2; var d = delete Math.PI || delete Math.E || delete Math.LN10 || delete Math.LN2 || delete Math.LOG2E || delete Math.LOG10E || delete Math.SQRT1_2 || delete Math.SQRT2; `+
				`var found = ""; for (var p in Math) found += p; print(Math.PI == before, Math.E > 2.7, d, "[" + found + "]")`,
				"true true false []\n")},

		// 15.8.2: "Every function listed in this section applies the ToNumber operator to each of its arguments"
		{"15.8.2 the Math functions convert their arguments with ToNumber", output(`print(Math.abs("-2"), Math.floor("1.5"), Math.max("3", true), Math.pow("2", "3") == 8, Math.abs(null), Math.abs("x"))`, "2 1 3 true 0 NaN\n")},
		// 15.8.2: "(in left-to-right order if there is more than one)"
		{"15.8.2 two arguments are converted from left to right", output(objJS+`Math.atan2(obj("y", 1), obj("x", 1)); Math.pow(obj("base", 2), obj("exponent", 2)); Math.max(obj("a", 1), obj("b", 2)); Math.min(obj("c", 1), obj("d", 2))`, "y\nx\nbase\nexponent\na\nb\nc\nd\n")},

		// 15.8.2.1: "the result has the same magnitude as the argument but has positive sign"
		{"15.8.2.1 abs returns the magnitude", output(`print(Math.abs(-3.5), Math.abs(3.5), Math.abs(-1e308), Math.abs(-5e-324) > 0)`, "3.5 3.5 1e+308 true\n")},
		// 15.8.2.1: NaN gives NaN; -0 gives +0; -Infinity gives +Infinity
		{"15.8.2.1 abs special cases", output(`print(Math.abs(NaN), 1 / Math.abs(-0), Math.abs(-Infinity))`, "NaN Infinity Infinity\n")},

		// 15.8.2.2: NaN, above 1 and below -1 give NaN; exactly 1 gives +0
		{"15.8.2.2 acos special cases", output(`print(Math.acos(NaN), Math.acos(1.0000001), Math.acos(-1.0000001), 1 / Math.acos(1))`, "NaN NaN NaN Infinity\n")},
		// 15.8.2.2: "Returns an implementation-dependent approximation to the arc cosine of the argument."
		{"15.8.2.2 acos(-1) is pi", approx(`print(Math.acos(-1))`, math.Pi)},
		{"15.8.2.2 acos(0.5)", approx(`print(Math.acos(0.5))`, math.Acos(0.5))},

		// 15.8.2.3: NaN, above 1 and below -1 give NaN; +0 gives +0; -0 gives -0
		{"15.8.2.3 asin special cases", output(`print(Math.asin(NaN), Math.asin(1.0000001), Math.asin(-1.0000001), 1 / Math.asin(0), 1 / Math.asin(-0))`, "NaN NaN NaN Infinity -Infinity\n")},
		// 15.8.2.3: "Returns an implementation-dependent approximation to the arc sine of the argument."
		{"15.8.2.3 asin(1) is pi/2", approx(`print(Math.asin(1))`, math.Pi/2)},
		{"15.8.2.3 asin(-0.5)", approx(`print(Math.asin(-0.5))`, math.Asin(-0.5))},

		// 15.8.2.4: NaN gives NaN; +0 gives +0; -0 gives -0
		{"15.8.2.4 atan special cases", output(`print(Math.atan(NaN), 1 / Math.atan(0), 1 / Math.atan(-0))`, "NaN Infinity -Infinity\n")},
		// 15.8.2.4: "If the argument is +Infinity, the result is an implementation-dependent approximation to +pi/2."
		{"15.8.2.4 atan(+Infinity) is pi/2", approx(`print(Math.atan(Infinity))`, math.Pi/2)},
		// 15.8.2.4: "If the argument is -Infinity, the result is an implementation-dependent approximation to -pi/2."
		{"15.8.2.4 atan(-Infinity) is -pi/2", approx(`print(Math.atan(-Infinity))`, -math.Pi/2)},
		{"15.8.2.4 atan(1) is pi/4", approx(`print(Math.atan(1))`, math.Pi/4)},

		// 15.8.2.5: "If either argument is NaN, the result is NaN."
		{"15.8.2.5 atan2 with a NaN argument is NaN", output(`print(Math.atan2(NaN, 1), Math.atan2(1, NaN), Math.atan2(NaN, NaN))`, "NaN NaN NaN\n")},
		// 15.8.2.5: the cases whose result is +0 or -0
		{"15.8.2.5 atan2 cases that give a zero",
			output(`print(1 / Math.atan2(0, 1), 1 / Math.atan2(0, 0), 1 / Math.atan2(-0, 1), 1 / Math.atan2(-0, 0), 1 / Math.atan2(1, Infinity), 1 / Math.atan2(-1, Infinity))`,
				"Infinity Infinity -Infinity -Infinity Infinity -Infinity\n")},
		// 15.8.2.5: "If y>0 and x is +0 [or -0], the result is an implementation-dependent approximation to +pi/2."
		{"15.8.2.5 atan2(1, +0) is pi/2", approx(`print(Math.atan2(1, 0))`, math.Pi/2)},
		{"15.8.2.5 atan2(1, -0) is pi/2", approx(`print(Math.atan2(1, -0))`, math.Pi/2)},
		// 15.8.2.5: "If y is +0 and x is -0 [or x<0], the result is an implementation-dependent approximation to +pi."
		{"15.8.2.5 atan2(+0, -0) is pi", approx(`print(Math.atan2(0, -0))`, math.Pi)},
		{"15.8.2.5 atan2(+0, -1) is pi", approx(`print(Math.atan2(0, -1))`, math.Pi)},
		// 15.8.2.5: "If y is -0 and x is -0 [or x<0], the result is an implementation-dependent approximation to -pi."
		{"15.8.2.5 atan2(-0, -0) is -pi", approx(`print(Math.atan2(-0, -0))`, -math.Pi)},
		{"15.8.2.5 atan2(-0, -1) is -pi", approx(`print(Math.atan2(-0, -1))`, -math.Pi)},
		// 15.8.2.5: "If y<0 and x is +0 [or -0], the result is an implementation-dependent approximation to -pi/2."
		{"15.8.2.5 atan2(-1, +0) is -pi/2", approx(`print(Math.atan2(-1, 0))`, -math.Pi/2)},
		{"15.8.2.5 atan2(-1, -0) is -pi/2", approx(`print(Math.atan2(-1, -0))`, -math.Pi/2)},
		// 15.8.2.5: "If y>0 and y is finite and x is -Infinity, the result if an implementation-dependent approximation to +pi."
		{"15.8.2.5 atan2(1, -Infinity) is pi", approx(`print(Math.atan2(1, -Infinity))`, math.Pi)},
		// 15.8.2.5: "If y<0 and y is finite and x is -Infinity, the result is an implementation-dependent approximation to -pi."
		{"15.8.2.5 atan2(-1, -Infinity) is -pi", approx(`print(Math.atan2(-1, -Infinity))`, -math.Pi)},
		// 15.8.2.5: "If y is +Infinity and x is finite, the result is an implementation-dependent approximation to +pi/2."
		{"15.8.2.5 atan2(+Infinity, 1) is pi/2", approx(`print(Math.atan2(Infinity, 1))`, math.Pi/2)},
		// 15.8.2.5: "If y is -Infinity and x is finite, the result is an implementation-dependent approximation to -pi/2."
		{"15.8.2.5 atan2(-Infinity, 1) is -pi/2", approx(`print(Math.atan2(-Infinity, 1))`, -math.Pi/2)},
		// 15.8.2.5: "If y is +Infinity and x is +Infinity, the result is an implementation-dependent approximation to +pi/4."
		{"15.8.2.5 atan2(+Infinity, +Infinity) is pi/4", approx(`print(Math.atan2(Infinity, Infinity))`, math.Pi/4)},
		// 15.8.2.5: "If y is +Infinity and x is -Infinity, the result is an implementation-dependent approximation to +3pi/4."
		{"15.8.2.5 atan2(+Infinity, -Infinity) is 3pi/4", approx(`print(Math.atan2(Infinity, -Infinity))`, 3*math.Pi/4)},
		// 15.8.2.5: "If y is -Infinity and x is +Infinity, the result is an implementation-dependent approximation to -pi/4."
		{"15.8.2.5 atan2(-Infinity, +Infinity) is -pi/4", approx(`print(Math.atan2(-Infinity, Infinity))`, -math.Pi/4)},
		// 15.8.2.5: "If y is -Infinity and x is -Infinity, the result is an implementation-dependent approximation to -3pi/4."
		{"15.8.2.5 atan2(-Infinity, -Infinity) is -3pi/4", approx(`print(Math.atan2(-Infinity, -Infinity))`, -3*math.Pi/4)},
		// 15.8.2.5: "the signs of the arguments are used to determine the quadrant of the result"
		{"15.8.2.5 atan2(-1, -1) is in the third quadrant", approx(`print(Math.atan2(-1, -1))`, -3*math.Pi/4)},
		{"15.8.2.5 atan2(1, 2)", approx(`print(Math.atan2(1, 2))`, math.Atan2(1, 2))},

		// 15.8.2.6: "Returns the smallest ... number value that is not less than the argument and is equal to a mathematical integer."
		{"15.8.2.6 ceil rounds up", output(`print(Math.ceil(1.2), Math.ceil(-1.2), Math.ceil(5), Math.ceil(-5), Math.ceil(4503599627370495.5))`, "2 -1 5 -5 4503599627370496\n")},
		// 15.8.2.6: NaN, +0, -0, +Infinity, -Infinity are returned as they are
		{"15.8.2.6 ceil special cases", output(`print(Math.ceil(NaN), 1 / Math.ceil(0), 1 / Math.ceil(-0), Math.ceil(Infinity), Math.ceil(-Infinity))`, "NaN Infinity -Infinity Infinity -Infinity\n")},
		// 15.8.2.6: "If the argument is less than 0 but greater than -1, the result is -0."
		{"15.8.2.6 ceil of a value between -1 and 0 is -0", output(`print(1 / Math.ceil(-0.5), 1 / Math.ceil(-5e-324))`, "-Infinity -Infinity\n")},
		// 15.8.2.6: "The value of Math.ceil(x) is the same as the value of -Math.floor(-x)."
		{"15.8.2.6 ceil(x) equals -floor(-x)", output(`print(Math.ceil(2.5) == -Math.floor(-2.5), Math.ceil(-2.5) == -Math.floor(2.5))`, "true true\n")},

		// 15.8.2.7: NaN gives NaN; +0 and -0 give 1; the infinities give NaN
		{"15.8.2.7 cos special cases", output(`print(Math.cos(NaN), Math.cos(0), Math.cos(-0), Math.cos(Infinity), Math.cos(-Infinity))`, "NaN 1 1 NaN NaN\n")},
		// 15.8.2.7: "Returns an implementation-dependent approximation to the cosine of the argument."
		{"15.8.2.7 cos(pi) is -1", approx(`print(Math.cos(Math.PI))`, -1)},
		{"15.8.2.7 cos(1)", approx(`print(Math.cos(1))`, math.Cos(1))},

		// 15.8.2.8: NaN gives NaN; +0 and -0 give 1; +Infinity gives +Infinity; -Infinity gives +0
		{"15.8.2.8 exp special cases", output(`print(Math.exp(NaN), Math.exp(0), Math.exp(-0), Math.exp(Infinity), 1 / Math.exp(-Infinity))`, "NaN 1 1 Infinity Infinity\n")},
		// 15.8.2.8: "e raised to the power of the argument"
		{"15.8.2.8 exp(1) is e", approx(`print(Math.exp(1))`, math.E)},
		{"15.8.2.8 exp(-2.5)", approx(`print(Math.exp(-2.5))`, math.Exp(-2.5))},

		// 15.8.2.9: "Returns the greatest ... number value that is not greater than the argument and is equal to a mathematical integer."
		{"15.8.2.9 floor rounds down", output(`print(Math.floor(1.8), Math.floor(-1.2), Math.floor(5), Math.floor(-5), Math.floor(-4503599627370495.5))`, "1 -2 5 -5 -4503599627370496\n")},
		// 15.8.2.9: NaN, +0, -0, +Infinity, -Infinity are returned as they are
		{"15.8.2.9 floor special cases", output(`print(Math.floor(NaN), 1 / Math.floor(0), 1 / Math.floor(-0), Math.floor(Infinity), Math.floor(-Infinity))`, "NaN Infinity -Infinity Infinity -Infinity\n")},
		// 15.8.2.9: "If the argument is greater than 0 but less than 1, the result is +0."
		{"15.8.2.9 floor of a value between 0 and 1 is +0", output(`print(1 / Math.floor(0.5), 1 / Math.floor(5e-324))`, "Infinity Infinity\n")},
		// 15.8.2.9: "The value of Math.floor(x) is the same as the value of -Math.ceil(-x)."
		{"15.8.2.9 floor(x) equals -ceil(-x)", output(`print(Math.floor(2.5) == -Math.ceil(-2.5), Math.floor(-2.5) == -Math.ceil(2.5))`, "true true\n")},

		// 15.8.2.10: NaN and negative give NaN; +0 and -0 give -Infinity; 1 gives +0; +Infinity gives +Infinity
		{"15.8.2.10 log special cases", output(`print(Math.log(NaN), Math.log(-1), Math.log(-Infinity), Math.log(0), Math.log(-0), 1 / Math.log(1), Math.log(Infinity))`, "NaN NaN NaN -Infinity -Infinity Infinity Infinity\n")},
		// 15.8.2.10: "Returns an implementation-dependent approximation to natural logarithm of the argument."
		{"15.8.2.10 log(e) is 1", approx(`print(Math.log(Math.E))`, 1)},
		{"15.8.2.10 log(10)", approx(`print(Math.log(10))`, math.Ln10)},

		// 15.8.2.11: "If x>y, the result is x. If y>x, the result is y."
		{"15.8.2.11 max returns the larger argument", output(`print(Math.max(1, 2), Math.max(2, 1), Math.max(-1, -2), Math.max(-Infinity, 5), Math.max(3, 3))`, "2 2 -1 5 3\n")},
		// 15.8.2.11: "If either argument is NaN, the result is NaN."
		{"15.8.2.11 max with a NaN argument is NaN", output(`print(Math.max(NaN, 1), Math.max(1, NaN), Math.max(NaN, Infinity))`, "NaN NaN NaN\n")},
		// 15.8.2.11: +0 and +0, +0 and -0, -0 and +0 give +0; -0 and -0 give -0
		{"15.8.2.11 max of zeros prefers +0", output(`print(1 / Math.max(0, 0), 1 / Math.max(0, -0), 1 / Math.max(-0, 0), 1 / Math.max(-0, -0))`, "Infinity Infinity Infinity -Infinity\n")},
		// 15.8.2.11 with §15: max takes x and y; a missing y is undefined, which converts to NaN
		{"15.8.2.11 max with fewer or more than two arguments", output(`print(Math.max(1), Math.max(), Math.max(1, 2, 3))`, "NaN NaN 2\n")},

		// 15.8.2.12: "If x<y, the result is x. If y<x, the result is y."
		{"15.8.2.12 min returns the smaller argument", output(`print(Math.min(1, 2), Math.min(2, 1), Math.min(-1, -2), Math.min(Infinity, 5), Math.min(3, 3))`, "1 1 -2 5 3\n")},
		// 15.8.2.12: "If either argument is NaN, the result is NaN."
		{"15.8.2.12 min with a NaN argument is NaN", output(`print(Math.min(NaN, 1), Math.min(1, NaN), Math.min(NaN, -Infinity))`, "NaN NaN NaN\n")},
		// 15.8.2.12: +0 and +0 give +0; +0 and -0, -0 and +0, -0 and -0 give -0
		{"15.8.2.12 min of zeros prefers -0", output(`print(1 / Math.min(0, 0), 1 / Math.min(0, -0), 1 / Math.min(-0, 0), 1 / Math.min(-0, -0))`, "Infinity -Infinity -Infinity -Infinity\n")},
		// 15.8.2.12 with §15: min takes x and y; a missing y is undefined, which converts to NaN
		{"15.8.2.12 min with fewer or more than two arguments", output(`print(Math.min(1), Math.min(), Math.min(3, 2, 1))`, "NaN NaN 2\n")},

		// 15.8.2.13: "If y is NaN, the result is NaN." "If y is +0 [or -0], the result is 1, even if x is NaN." "If x is NaN and y is nonzero, the result is NaN."
		{"15.8.2.13 pow with NaN or a zero exponent", output(`print(Math.pow(2, NaN), Math.pow(NaN, 0), Math.pow(NaN, -0), Math.pow(Infinity, 0), Math.pow(NaN, 1))`, "NaN 1 1 1 NaN\n")},
		// 15.8.2.13: abs(x)>1, ==1 and <1 with y = +Infinity or -Infinity
		{"15.8.2.13 pow with an infinite exponent",
			output(`print(Math.pow(2, Infinity), 1 / Math.pow(2, -Infinity), Math.pow(1, Infinity), Math.pow(-1, -Infinity), 1 / Math.pow(0.5, Infinity), Math.pow(-0.5, -Infinity), Math.pow(-2, Infinity))`,
				"Infinity Infinity NaN NaN Infinity Infinity Infinity\n")},
		// 15.8.2.13: x is +Infinity or -Infinity
		{"15.8.2.13 pow with an infinite base",
			output(`print(Math.pow(Infinity, 0.5), 1 / Math.pow(Infinity, -1), Math.pow(-Infinity, 3), Math.pow(-Infinity, 2), Math.pow(-Infinity, 0.5), 1 / Math.pow(-Infinity, -3), 1 / Math.pow(-Infinity, -2))`,
				"Infinity Infinity -Infinity Infinity Infinity -Infinity Infinity\n")},
		// 15.8.2.13: x is +0 or -0
		{"15.8.2.13 pow with a zero base",
			output(`print(1 / Math.pow(0, 3), Math.pow(0, -3), 1 / Math.pow(-0, 3), 1 / Math.pow(-0, 2), 1 / Math.pow(-0, 0.5), Math.pow(-0, -3), Math.pow(-0, -2), Math.pow(-0, -0.5))`,
				"Infinity Infinity -Infinity Infinity Infinity -Infinity Infinity Infinity\n")},
		// 15.8.2.13: "If x<0 and x is finite and y is finite and y is not an integer, the result is NaN."
		{"15.8.2.13 pow of a negative base to a fractional exponent is NaN", output(`print(Math.pow(-8, 1 / 3), Math.pow(-2, 0.5), Math.pow(-2, 2))`, "NaN NaN 4\n")},
		// 15.8.2.13: "Returns an implementation-dependent approximation to the result of raising x to the power y."
		{"15.8.2.13 pow(2, 10)", approx(`print(Math.pow(2, 10))`, 1024)},
		{"15.8.2.13 pow(2, 0.5)", approx(`print(Math.pow(2, 0.5))`, math.Sqrt2)},
		{"15.8.2.13 pow(10, -2)", approx(`print(Math.pow(10, -2))`, 0.01)},
		{"15.8.2.13 pow(-3, 3)", approx(`print(Math.pow(-3, 3))`, -27)},

		// 15.8.2.14: "greater than or equal to 0 but less than 1"
		{"15.8.2.14 random returns a number from 0 up to 1",
			output(`var ok = true; for (var i = 0; i < 1000; i++) { var r = Math.random(); if (!(r >= 0 && r < 1) || 1 / r < 0) ok = false } print(ok, typeof Math.random())`, "true number\n")},

		// 15.8.2.15: "Returns the number value that is closest to the argument and is equal to a mathematical integer."
		{"15.8.2.15 round goes to the nearest integer", output(`print(Math.round(2.4), Math.round(2.6), Math.round(-2.4), Math.round(-2.6), Math.round(7))`, "2 3 -2 -3 7\n")},
		// 15.8.2.15: "If two integer number values are equally close to the argument, then the result is the number value that is closer to +Infinity."
		{"15.8.2.15 round of a half goes toward +Infinity", output(`print(Math.round(3.5), Math.round(-3.5), Math.round(0.5), Math.round(-1.5))`, "4 -3 1 -1\n")},
		// 15.8.2.15: NaN, +0, -0, +Infinity, -Infinity are returned as they are
		{"15.8.2.15 round special cases", output(`print(Math.round(NaN), 1 / Math.round(0), 1 / Math.round(-0), Math.round(Infinity), Math.round(-Infinity))`, "NaN Infinity -Infinity Infinity -Infinity\n")},
		// 15.8.2.15: "If the argument is greater than 0 but less than 0.5, the result is +0."
		{"15.8.2.15 round of a value between 0 and 0.5 is +0", output(`print(1 / Math.round(0.2), 1 / Math.round(5e-324))`, "Infinity Infinity\n")},
		// 15.8.2.15: "If the argument is less than 0 but greater than or equal to -0.5, the result is -0."
		{"15.8.2.15 round of a value from -0.5 up to 0 is -0", output(`print(1 / Math.round(-0.5), 1 / Math.round(-0.2), 1 / Math.round(-5e-324))`, "-Infinity -Infinity -Infinity\n")},
		// 15.8.2.15: "If the argument is already an integer, the result is the argument itself."
		{"15.8.2.15 round of a large integer is the integer", output(`print(Math.round(9007199254740991) == 9007199254740991, Math.round(-4503599627370497) == -4503599627370497)`, "true true\n")},

		// 15.8.2.16: NaN gives NaN; +0 gives +0; -0 gives -0; the infinities give NaN
		{"15.8.2.16 sin special cases", output(`print(Math.sin(NaN), 1 / Math.sin(0), 1 / Math.sin(-0), Math.sin(Infinity), Math.sin(-Infinity))`, "NaN Infinity -Infinity NaN NaN\n")},
		// 15.8.2.16: "Returns an implementation-dependent approximation to the sine of the argument."
		{"15.8.2.16 sin(pi/2) is 1", approx(`print(Math.sin(Math.PI / 2))`, 1)},
		{"15.8.2.16 sin(-1)", approx(`print(Math.sin(-1))`, math.Sin(-1))},

		// 15.8.2.17: NaN and negative give NaN; +0 gives +0; -0 gives -0; +Infinity gives +Infinity
		{"15.8.2.17 sqrt special cases", output(`print(Math.sqrt(NaN), Math.sqrt(-1), Math.sqrt(-Infinity), 1 / Math.sqrt(0), 1 / Math.sqrt(-0), Math.sqrt(Infinity))`, "NaN NaN NaN Infinity -Infinity Infinity\n")},
		// 15.8.2.17: "Returns an implementation-dependent approximation to the square root of the argument."
		{"15.8.2.17 sqrt(16) is 4", approx(`print(Math.sqrt(16))`, 4)},
		{"15.8.2.17 sqrt(2)", approx(`print(Math.sqrt(2))`, math.Sqrt2)},

		// 15.8.2.18: NaN gives NaN; +0 gives +0; -0 gives -0; the infinities give NaN
		{"15.8.2.18 tan special cases", output(`print(Math.tan(NaN), 1 / Math.tan(0), 1 / Math.tan(-0), Math.tan(Infinity), Math.tan(-Infinity))`, "NaN Infinity -Infinity NaN NaN\n")},
		// 15.8.2.18: "Returns an implementation-dependent approximation to the tangent of the argument."
		{"15.8.2.18 tan(pi/4) is 1", approx(`print(Math.tan(Math.PI / 4))`, 1)},
		{"15.8.2.18 tan(0.5)", approx(`print(Math.tan(0.5))`, math.Tan(0.5))},
	})
}

// §15.9, Date objects. The harness runs with TZ=UTC (README), so LocalTZA
// and DaylightSavingTA are 0 and local time equals UTC. Strings from
// toString, toUTCString and toLocaleString are never compared; only their
// round-trips through Date.parse are.

// dateJS defines:
//   - utc(d) and local(d): the components of d from the UTC and the local
//     getters, as "year-month-date hours:minutes:seconds.ms" with month
//     counted from 1;
//   - base(): a new Date for 2001-06-15 10:20:30.400 UTC, a Friday;
//   - nearNow(before, s): whether s is a string that Date.parse reads as a
//     time from before up to now.
const dateJS = `function utc(d) { return d.getUTCFullYear() + "-" + (d.getUTCMonth() + 1) + "-" + d.getUTCDate() + " " + d.getUTCHours() + ":" + d.getUTCMinutes() + ":" + d.getUTCSeconds() + "." + d.getUTCMilliseconds() } ` +
	`function local(d) { return d.getFullYear() + "-" + (d.getMonth() + 1) + "-" + d.getDate() + " " + d.getHours() + ":" + d.getMinutes() + ":" + d.getSeconds() + "." + d.getMilliseconds() } ` +
	`function base() { return new Date(Date.UTC(2001, 5, 15, 10, 20, 30, 400)) } ` +
	`function nearNow(before, s) { var t = Date.parse(s); var after = new Date().getTime(); return typeof s == "string" && t >= before && t <= after } `

// nowJS sets before to the current time rounded down to the second, since
// the string forms of a date need not show milliseconds.
const nowJS = `var before = Math.floor(new Date().getTime() / 1000) * 1000; `

func TestS15_9_DateObjects(t *testing.T) {
	runRows(t, []row{
		// 15.9.1: "if any argument to such a function is NaN, the result will be NaN"
		{"15.9.1 a NaN component gives a NaN time value",
			output(`print(Date.UTC(NaN, 0, 1), Date.UTC(2000, NaN, 1), Date.UTC(2000, 0, NaN), Date.UTC(2000, 0, 1, NaN), new Date(2000, 0, 1, 0, 0, 0, NaN).getTime())`, "NaN NaN NaN NaN NaN\n")},

		// 15.9.1.1: "The exact moment of midnight at the beginning of 01 January, 1970 UTC is represented by the value +0."
		{"15.9.1.1 time value 0 is the start of 1970 UTC", output(dateJS+`print(utc(new Date(0)), Date.UTC(1970, 0, 1), 1 / Date.UTC(1970, 0, 1))`, "1970-1-1 0:0:0.0 0 Infinity\n")},
		// 15.9.1.1: "Leap seconds are ignored. It is assumed that there are exactly 86,400,000 milliseconds per day."
		{"15.9.1.1 every day has 86400000 milliseconds", output(`print(Date.UTC(1970, 0, 2), Date.UTC(1999, 0, 1) - Date.UTC(1998, 11, 31), Date.UTC(1972, 6, 1) - Date.UTC(1972, 5, 30))`, "86400000 86400000 86400000\n")},
		// 15.9.1.1: "exactly -100,000,000 days to 100,000,000 days measured relative to midnight at the beginning of 01 January, 1970 UTC"
		{"15.9.1.1 the time range is 8.64e15 milliseconds either side of 1970",
			output(`print(new Date(8.64e15).getTime(), new Date(-8.64e15).getTime() == -8.64e15, new Date(8.64e15 + 1).getTime(), new Date(-8.64e15 - 1).getTime())`, "8640000000000000 true NaN NaN\n")},
		// 15.9.1.1: the ends of the range in calendar terms
		{"15.9.1.1 the ends of the range are in 275760 and -271821",
			output(dateJS+`print(utc(new Date(8.64e15)), utc(new Date(-8.64e15)))`, "275760-9-13 0:0:0.0 -271821-4-20 0:0:0.0\n")},

		// 15.9.1.2: "Day(t) = floor(t / msPerDay)" "TimeWithinDay(t) = t modulo msPerDay"
		{"15.9.1.2 a negative time value belongs to the day before 1970",
			output(dateJS+`print(utc(new Date(-1)), utc(new Date(-86400000)), utc(new Date(86399999)))`, "1969-12-31 23:59:59.999 1969-12-31 0:0:0.0 1970-1-1 23:59:59.999\n")},

		// 15.9.1.3: "leap years are precisely those which are (divisible by 4) and ((not divisible by 100) or (divisible by 400))"
		{"15.9.1.3 leap years follow the Gregorian rule",
			output(dateJS+`print(utc(new Date(Date.UTC(2000, 1, 29))), utc(new Date(Date.UTC(2004, 1, 29))), utc(new Date(Date.UTC(1900, 1, 29))), utc(new Date(Date.UTC(2100, 1, 29))), utc(new Date(Date.UTC(2001, 1, 29))))`,
				"2000-2-29 0:0:0.0 2004-2-29 0:0:0.0 1900-3-1 0:0:0.0 2100-3-1 0:0:0.0 2001-3-1 0:0:0.0\n")},
		// 15.9.1.3: DaysInYear(y) is 365 or 366
		{"15.9.1.3 a year has 365 or 366 days",
			output(`var day = 86400000; print((Date.UTC(2001, 0, 1) - Date.UTC(2000, 0, 1)) / day, (Date.UTC(1901, 0, 1) - Date.UTC(1900, 0, 1)) / day, (Date.UTC(2002, 0, 1) - Date.UTC(2001, 0, 1)) / day)`, "366 365 365\n")},
		// 15.9.1.3: "YearFromTime(t) = the largest integer y ... such that TimeFromYear(y) <= t"
		{"15.9.1.3 the year changes exactly at the start of the year",
			output(`var t = Date.UTC(2000, 0, 1); print(new Date(t - 1).getUTCFullYear(), new Date(t).getUTCFullYear())`, "1999 2000\n")},
		// 15.9.1.3: "an extrapolated Gregorian system", also for year 0 and negative years
		{"15.9.1.3 years before year 1 are extrapolated",
			output(dateJS+`var d = new Date(0); d.setUTCFullYear(0, 1, 29); var e = new Date(0); e.setUTCFullYear(-1, 1, 29); print(utc(d), utc(e))`, "0-2-29 0:0:0.0 -1-3-1 0:0:0.0\n")},

		// 15.9.1.4: "MonthFromTime(t) = 0 if 0 <= DayWithinYear(t) < 31 ... = 11 if 334+InLeapYear(t) <= DayWithinYear(t) < 365+InLeapYear(t)"
		{"15.9.1.4 the month of a day within the year",
			output(`var day = 86400000; var y1 = Date.UTC(2001, 0, 1), y0 = Date.UTC(2000, 0, 1); `+
				`print(new Date(y1 + 30 * day).getUTCMonth(), new Date(y1 + 31 * day).getUTCMonth(), new Date(y1 + 58 * day).getUTCMonth(), new Date(y1 + 59 * day).getUTCMonth(), new Date(y0 + 59 * day).getUTCMonth(), new Date(y0 + 60 * day).getUTCMonth(), new Date(y1 + 333 * day).getUTCMonth(), new Date(y1 + 334 * day).getUTCMonth(), new Date(y0 + 365 * day).getUTCMonth())`,
				"0 1 1 2 1 2 10 11 11\n")},
		// 15.9.1.4: "Note that MonthFromTime(0) = 0, corresponding to Thursday, 01 January, 1970."
		{"15.9.1.4 time value 0 is in January", output(`print(new Date(0).getUTCMonth())`, "0\n")},

		// 15.9.1.5: DateFromTime(t) for each month
		{"15.9.1.5 the date within each month",
			output(`var d = new Date(0); var out = ""; for (var m = 0; m < 12; m++) { d.setUTCFullYear(2000, m, 1); var last = new Date(d.getTime() - 86400000); out = out + last.getUTCDate() + " " } print(out + new Date(Date.UTC(2001, 2, 1) - 1).getUTCDate())`,
				"31 31 29 31 30 31 30 31 31 30 31 30 28\n")},

		// 15.9.1.6: "WeekDay(t) = (Day(t) + 4) modulo 7" "Note that WeekDay(0) = 4, corresponding to Thursday, 01 January, 1970."
		{"15.9.1.6 the week day", output(`print(new Date(0).getUTCDay(), new Date(-86400000).getUTCDay(), new Date(-4 * 86400000).getUTCDay(), new Date(Date.UTC(2001, 5, 15)).getUTCDay(), new Date(Date.UTC(2000, 0, 2)).getUTCDay())`, "4 3 0 5 0\n")},

		// 15.9.1.7: LocalTZA "represents the local standard time"; the harness sets TZ=UTC (README)
		{"15.9.1.7 the local time zone adjustment is 0 under TZ=UTC", output(dateJS+`var d = base(); print(new Date(0).getTimezoneOffset(), local(d) == utc(d))`, "0 true\n")},
		// 15.9.1.8: DaylightSavingTA(t) "measured in milliseconds"; UTC has no daylight saving time
		{"15.9.1.8 there is no daylight saving adjustment under TZ=UTC",
			output(`var summer = new Date(Date.UTC(2001, 6, 1, 12)), winter = new Date(Date.UTC(2001, 0, 1, 12)); print(summer.getHours(), winter.getHours(), summer.getTimezoneOffset(), winter.getTimezoneOffset())`, "12 12 0 0\n")},
		// 15.9.1.9: "LocalTime(t) = t + LocalTZA + DaylightSavingTA(t)" "UTC(t) = t - LocalTZA - DaylightSavingTA(t - LocalTZA)"
		{"15.9.1.9 local time and UTC agree under TZ=UTC",
			output(`print(new Date(2001, 5, 15, 10, 20, 30, 400).getTime() == Date.UTC(2001, 5, 15, 10, 20, 30, 400), new Date(1999, 11, 31).getTime() == Date.UTC(1999, 11, 31))`, "true true\n")},

		// 15.9.1.10: HourFromTime, MinFromTime, SecFromTime, msFromTime
		{"15.9.1.10 hours, minutes, seconds and milliseconds of a time value",
			output(`var t = 3 * 3600000 + 25 * 60000 + 45 * 1000 + 678; var d = new Date(t); print(d.getUTCHours(), d.getUTCMinutes(), d.getUTCSeconds(), d.getUTCMilliseconds())`, "3 25 45 678\n")},
		// 15.9.1.10: the components of a negative time value use floor and modulo
		{"15.9.1.10 the components of a negative time value",
			output(`var d = new Date(-3723004); print(d.getUTCHours(), d.getUTCMinutes(), d.getUTCSeconds(), d.getUTCMilliseconds())`, "22 57 56 996\n")},

		// 15.9.1.11 step 1: "If hour is not finite or min is not finite or sec is not finite or ms is not finite, return NaN."
		{"15.9.1.11 MakeTime of an infinite component is NaN",
			output(`print(Date.UTC(2000, 0, 1, Infinity), Date.UTC(2000, 0, 1, 0, -Infinity), Date.UTC(2000, 0, 1, 0, 0, Infinity), Date.UTC(2000, 0, 1, 0, 0, 0, Infinity))`, "NaN NaN NaN NaN\n")},
		// 15.9.1.11 steps 2-5: "Call ToInteger(hour)." and so on
		{"15.9.1.11 MakeTime truncates each component", output(`print(Date.UTC(2000, 0, 1, 1.9, 2.9, 3.9, 4.9) == Date.UTC(2000, 0, 1, 1, 2, 3, 4), Date.UTC(1970, 0, 1, -1.5) == -3600000)`, "true true\n")},
		// 15.9.1.11 step 6: hour * msPerHour + min * msPerMinute + sec * msPerSecond + ms, so components may overflow
		{"15.9.1.11 MakeTime adds components that are out of range",
			output(`print(Date.UTC(2000, 0, 1, 24) == Date.UTC(2000, 0, 2), Date.UTC(2000, 0, 1, 0, -1) == Date.UTC(1999, 11, 31, 23, 59), Date.UTC(2000, 0, 1, 0, 0, 90) == Date.UTC(2000, 0, 1, 0, 1, 30), Date.UTC(2000, 0, 1, 0, 0, 0, -1) == Date.UTC(2000, 0, 1) - 1)`,
				"true true true true\n")},

		// 15.9.1.12 step 1: "If year is not finite or month is not finite or date is not finite, return NaN."
		{"15.9.1.12 MakeDay of an infinite component is NaN", output(`print(Date.UTC(Infinity, 0, 1), Date.UTC(2000, -Infinity, 1), Date.UTC(2000, 0, Infinity))`, "NaN NaN NaN\n")},
		// 15.9.1.12 steps 2-4: "Call ToInteger(year)." and so on
		{"15.9.1.12 MakeDay truncates each component", output(`print(Date.UTC(2000.9, 1.9, 2.9) == Date.UTC(2000, 1, 2))`, "true\n")},
		// 15.9.1.12 steps 5-6: "Compute Result(2) + floor(Result(3)/12)." "Compute Result(3) modulo 12."
		{"15.9.1.12 MakeDay carries months into years",
			output(`print(Date.UTC(2000, 12, 1) == Date.UTC(2001, 0, 1), Date.UTC(2000, 25, 1) == Date.UTC(2002, 1, 1), Date.UTC(2000, -1, 1) == Date.UTC(1999, 11, 1), Date.UTC(2000, -13, 1) == Date.UTC(1998, 11, 1))`, "true true true true\n")},
		// 15.9.1.12 step 8: "Compute Day(Result(7)) + Result(4) - 1." so the date may overflow
		{"15.9.1.12 MakeDay adds a date that is out of range",
			output(`print(Date.UTC(2000, 0, 0) == Date.UTC(1999, 11, 31), Date.UTC(2000, 0, 32) == Date.UTC(2000, 1, 1), Date.UTC(2001, 1, 29) == Date.UTC(2001, 2, 1), Date.UTC(2000, 0, -365) == Date.UTC(1998, 11, 31))`, "true true true true\n")},
		// 15.9.1.12 step 7: "but if this is not possible (because some argument is out of range), return NaN"
		{"15.9.1.12 MakeDay of a year far out of range is NaN", output(`print(Date.UTC(1e20, 0, 1), Date.UTC(2000, 1e20, 1), Date.UTC(-1e20, 0, 1))`, "NaN NaN NaN\n")},

		// 15.9.1.13 step 1: "If day is not finite or time is not finite, return NaN."
		{"15.9.1.13 MakeDate of a non-finite day or time is NaN", output(`print(Date.UTC(2000, 0, Infinity, 0), Date.UTC(2000, 0, 1, Infinity))`, "NaN NaN\n")},
		// 15.9.1.13 step 2: "Compute day * msPerDay + time."
		{"15.9.1.13 MakeDate is day times msPerDay plus time", output(`print(Date.UTC(1970, 0, 3, 1) == 2 * 86400000 + 3600000)`, "true\n")},

		// 15.9.1.14 step 1: "If time is not finite, return NaN."
		{"15.9.1.14 TimeClip of a non-finite time is NaN", output(`print(new Date(Infinity).getTime(), new Date(-Infinity).getTime(), new Date(NaN).getTime())`, "NaN NaN NaN\n")},
		// 15.9.1.14 step 2: "If abs(Result(1)) > 8.64e15, return NaN."
		{"15.9.1.14 TimeClip of a time out of range is NaN",
			output(`print(Date.UTC(275760, 8, 13) == 8.64e15, Date.UTC(275760, 8, 13, 0, 0, 0, 1), new Date(-8.64e15 - 1).getTime(), new Date(1e300).getTime())`, "true NaN NaN NaN\n")},
		// 15.9.1.14 step 3: "ToInteger(Result(2))"
		{"15.9.1.14 TimeClip truncates the time", output(`print(new Date(1.9).getTime(), new Date(-1.9).getTime(), new Date(8.64e15 + 0.5).getTime())`, "1 -1 8640000000000000\n")},

		// 15.9.2: "it returns a string representing the current time (UTC)"
		{"15.9.2 Date called as a function returns the current time as a string", output(dateJS+nowJS+`print(typeof Date(), nearNow(before, Date()))`, "string true\n")},
		// 15.9.2.1: "The arguments are accepted but are completely ignored."
		{"15.9.2.1 Date with seven arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0, 1, 0, 0, 0, 0)))`, "true\n")},
		// 15.9.2.2: "The arguments are accepted but are completely ignored."
		{"15.9.2.2 Date with six arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0, 1, 0, 0, 0)))`, "true\n")},
		// 15.9.2.3: "The arguments are accepted but are completely ignored."
		{"15.9.2.3 Date with five arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0, 1, 0, 0)))`, "true\n")},
		// 15.9.2.4: "The arguments are accepted but are completely ignored."
		{"15.9.2.4 Date with four arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0, 1, 0)))`, "true\n")},
		// 15.9.2.5: "The arguments are accepted but are completely ignored."
		{"15.9.2.5 Date with three arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0, 1)))`, "true\n")},
		// 15.9.2.6: "The arguments are accepted but are completely ignored."
		{"15.9.2.6 Date with two arguments ignores them", output(dateJS+nowJS+`print(nearNow(before, Date(2000, 0)))`, "true\n")},
		// 15.9.2.7: "The argument is accepted but is completely ignored."
		{"15.9.2.7 Date with one argument ignores it", output(dateJS+nowJS+`print(nearNow(before, Date(0)), nearNow(before, Date("1 Jan 2000")))`, "true true\n")},
		// 15.9.2.7: the ignored argument is not converted
		{"15.9.2.7 Date does not convert its argument", output(objJS+`Date(obj("value", 0)); print("done")`, "done\n")},
		// 15.9.2.8: "A string is created and returned as if by the expression new Date ().toString()."
		{"15.9.2.8 Date() is the string of the current date", output(dateJS+nowJS+`var s = Date(); print(typeof s, nearNow(before, s))`, "string true\n")},
		// 15.9.2: "the function call Date (...) is not equivalent to the object creation expression new Date (...)"
		{"15.9.2 Date() returns a string, not an object", output(`print(typeof Date(), typeof new Date())`, "string object\n")},

		// 15.9.3.1 steps 8-12: MakeDay, MakeTime, MakeDate, then TimeClip(UTC(...))
		{"15.9.3.1 new Date with seven arguments sets every component",
			output(dateJS+`var d = new Date(2001, 5, 15, 10, 20, 30, 400); print(local(d), d.getTime() == Date.UTC(2001, 5, 15, 10, 20, 30, 400))`, "2001-6-15 10:20:30.400 true\n")},
		// 15.9.3.1: "The [[Class]] property of the newly constructed object is set to "Date"." and [[Prototype]] is Date.prototype
		{"15.9.3.1 new Date creates a Date object", output(classOfJS+`Date.prototype.marker = "dp"; var d = new Date(2001, 0, 1, 0, 0, 0, 0); print(classOf(d), d.marker, typeof d)`, "[object Date] dp object\n")},
		// 15.9.3.1 steps 1-7: the arguments are converted with ToNumber in order
		{"15.9.3.1 new Date converts its arguments from left to right",
			output(objJS+`var d = new Date(obj("year", 2001), obj("month", 0), obj("date", 1), obj("hours", 0), obj("minutes", 0), obj("seconds", 0), obj("ms", 0)); print(d.getTime() == Date.UTC(2001, 0, 1))`,
				"year\nmonth\ndate\nhours\nminutes\nseconds\nms\ntrue\n")},
		// 15.9.3.1 step 8: "If Result(1) is not NaN and 0 <= ToInteger(Result(1)) <= 99, Result(8) is 1900+ToInteger(Result(1))"
		{"15.9.3.1 a year from 0 to 99 means 1900 to 1999",
			output(`print(new Date(0, 0, 1, 0, 0, 0, 0).getFullYear(), new Date(99, 0, 1, 0, 0, 0, 0).getFullYear(), new Date(99.9, 0, 1, 0, 0, 0, 0).getFullYear(), new Date(100, 0, 1, 0, 0, 0, 0).getFullYear(), new Date(-1, 0, 1, 0, 0, 0, 0).getFullYear())`, "1900 1999 1999 100 -1\n")},
		// 15.9.3.2: MakeTime(Result(4), Result(5), Result(6), 0)
		{"15.9.3.2 new Date with six arguments has 0 milliseconds", output(dateJS+`var d = new Date(2001, 5, 15, 10, 20, 30); print(local(d), new Date(1, 0, 1, 0, 0, 0).getFullYear())`, "2001-6-15 10:20:30.0 1901\n")},
		// 15.9.3.3: MakeTime(Result(4), Result(5), 0, 0)
		{"15.9.3.3 new Date with five arguments has 0 seconds", output(dateJS+`var d = new Date(2001, 5, 15, 10, 20); print(local(d), new Date(1, 0, 1, 0, 0).getFullYear())`, "2001-6-15 10:20:0.0 1901\n")},
		// 15.9.3.4: MakeTime(Result(4), 0, 0, 0)
		{"15.9.3.4 new Date with four arguments has 0 minutes", output(dateJS+`var d = new Date(2001, 5, 15, 10); print(local(d), new Date(1, 0, 1, 0).getFullYear())`, "2001-6-15 10:0:0.0 1901\n")},
		// 15.9.3.5: MakeDate(Result(5), 0)
		{"15.9.3.5 new Date with three arguments is midnight", output(dateJS+`var d = new Date(2001, 5, 15); print(local(d), new Date(1, 0, 1).getFullYear(), d.getTime() == Date.UTC(2001, 5, 15))`, "2001-6-15 0:0:0.0 1901 true\n")},
		// 15.9.3.5 steps 1-3: the arguments are converted with ToNumber
		{"15.9.3.5 new Date converts its arguments to numbers", output(`print(new Date("2001", "5", "15").getTime() == Date.UTC(2001, 5, 15), new Date(2001, 0, "x").getTime())`, "true NaN\n")},
		// 15.9.3.7 steps 3-4: "Let V be ToNumber(Result(1))." "Set the [[Value]] property ... to TimeClip(V)"
		{"15.9.3.7 new Date(number) uses the number as the time value", output(dateJS+`print(utc(new Date(86400000)), new Date(true).getTime(), new Date(null).getTime())`, "1970-1-2 0:0:0.0 1 0\n")},
		// 15.9.3.7 step 1: "Call ToPrimitive(value)." with no hint, so an object other than a Date uses valueOf
		{"15.9.3.7 new Date(object) converts the object to a primitive", output(objJS+`print(new Date(obj("value", 86400000)).getUTCDate())`, "value\n2\n")},
		// 15.9.3.7 steps 2 and 5: "If Type(Result(1)) is String" "Parse Result(1) as a date, in exactly the same manner as for the parse method"
		{"15.9.3.7 new Date(string) parses the string",
			output(dateJS+`var d = base(); d.setUTCMilliseconds(0); print(new Date(d.toString()).getTime() == d.getTime(), new Date(d.toUTCString()).getTime() == d.getTime())`, "true true\n")},
		// 15.9.3.7 step 1: ToPrimitive of a Date object calls its toString (§8.6.2.6), and the string is parsed
		{"15.9.3.7 new Date(date) converts the date through its string",
			output(`function other() { return new Date(86400000).toString() } var d = new Date(0); d.toString = other; print(new Date(d).getUTCDate())`, "2\n")},
		// 15.9.3.8: "The [[Value]] property of the newly constructed object is set to the current time (UTC)."
		{"15.9.3.8 new Date() is the current time",
			output(`var a = new Date().getTime(); var b = new Date().getTime(); print(typeof a, b >= a, a > Date.UTC(2020, 0, 1), a == Math.floor(a))`, "number true true true\n")},
		// 15.9.3.8: "The [[Class]] property of the newly constructed object is set to "Date"."
		{"15.9.3.8 new Date() creates a Date object", output(classOfJS+`print(classOf(new Date()))`, "[object Date]\n")},

		// 15.9.4: "The value of the internal [[Prototype]] property of the Date constructor is the Function prototype object."
		{"15.9.4 the Date constructor inherits from Function.prototype", output(`Function.prototype.marker = "fp"; print(Date.marker)`, "fp\n")},
		// 15.9.4: "the length property (whose value is 7)"
		{"15.9.4 Date.length is 7", output(`print(Date.length)`, "7\n")},
		// 15.9.4.1: "The initial value of Date.prototype is the built-in Date prototype object"
		{"15.9.4.1 Date.prototype provides the date methods", output(`print(new Date(0).getTime == Date.prototype.getTime)`, "true\n")},
		// 15.9.4.1: "This property shall have the attributes { DontEnum, DontDelete, ReadOnly }."
		{"15.9.4.1 Date.prototype cannot be replaced or deleted", output(`var original = Date.prototype; Date.prototype = null; print(delete Date.prototype, Date.prototype == original)`, "false true\n")},

		// 15.9.4.2: "all of the following expressions should produce the same numeric value ... x.valueOf() Date.parse(x.toString()) Date.parse(x.toGMTString())"
		{"15.9.4.2 parse reads back toString and toGMTString",
			output(`var d = new Date(Date.UTC(2001, 5, 15, 10, 20, 30)); print(Date.parse(d.toString()) == d.valueOf(), Date.parse(d.toGMTString()) == d.valueOf())`, "true true\n")},
		// 15.9.4.2: the round-trip holds for dates before 1970 and far ahead
		{"15.9.4.2 parse reads back dates before 1970 and after 2038",
			output(`var a = new Date(Date.UTC(1901, 0, 1, 23, 59, 59)), b = new Date(Date.UTC(2099, 11, 31, 1, 2, 3)); print(Date.parse(a.toString()) == a.valueOf(), Date.parse(a.toGMTString()) == a.valueOf(), Date.parse(b.toString()) == b.valueOf(), Date.parse(b.toGMTString()) == b.valueOf())`,
				"true true true true\n")},
		// 15.9.4.2: "The parse function applies the ToString operator to its argument"
		{"15.9.4.2 parse converts its argument to a string",
			output(`var d = new Date(Date.UTC(2001, 5, 15)); function s() { return d.toString() } var o = new Object(); o.toString = s; print(Date.parse(d) == d.valueOf(), Date.parse(o) == d.valueOf())`, "true true\n")},
		// 15.9.4.2: "it returns a number, the UTC time value corresponding to the date"
		{"15.9.4.2 parse returns a number", output(`print(typeof Date.parse(new Date(0).toString()), Date.parse(new Date(0).toString()))`, "number 0\n")},

		// 15.9.4.3 steps 1-11: TimeClip(MakeDate(MakeDay(...), MakeTime(...)))
		{"15.9.4.3 Date.UTC with seven arguments", output(dateJS+`var t = Date.UTC(2001, 5, 15, 10, 20, 30, 400); print(typeof t, utc(new Date(t)), Date.UTC(1970, 0, 1, 0, 0, 0, 1))`, "number 2001-6-15 10:20:30.400 1\n")},
		// 15.9.4.3 steps 1-7: the arguments are converted with ToNumber in order
		{"15.9.4.3 Date.UTC converts its arguments from left to right",
			output(objJS+`print(Date.UTC(obj("year", 1970), obj("month", 0), obj("date", 1), obj("hours", 0), obj("minutes", 0), obj("seconds", 0), obj("ms", 5)))`,
				"year\nmonth\ndate\nhours\nminutes\nseconds\nms\n5\n")},
		// 15.9.4.3 step 8: "If Result(1) is not NaN and 0 <= ToInteger(Result(1)) <= 99, Result(8) is 1900+ToInteger(Result(1))"
		{"15.9.4.3 Date.UTC reads a year from 0 to 99 as 1900 to 1999",
			output(`print(Date.UTC(0, 0, 1, 0, 0, 0, 0) == Date.UTC(1900, 0, 1, 0, 0, 0, 0), Date.UTC(99, 0, 1, 0, 0, 0, 0) == Date.UTC(1999, 0, 1, 0, 0, 0, 0), new Date(Date.UTC(100, 0, 1, 0, 0, 0, 0)).getUTCFullYear())`, "true true 100\n")},
		// 15.9.4.3: "it interprets the arguments in UTC rather than as local time"
		{"15.9.4.3 Date.UTC returns a time value, not a Date", output(`print(typeof Date.UTC(2000, 0, 1, 0, 0, 0, 0), Date.UTC(2000, 0, 1, 0, 0, 0, 0))`, "number 946684800000\n")},
		// 15.9.4.4: MakeTime(Result(4), Result(5), Result(6), 0)
		{"15.9.4.4 Date.UTC with six arguments", output(`print(Date.UTC(2001, 5, 15, 10, 20, 30) == Date.UTC(2001, 5, 15, 10, 20, 30, 0), Date.UTC(1, 0, 1, 0, 0, 0) == Date.UTC(1901, 0, 1, 0, 0, 0, 0))`, "true true\n")},
		// 15.9.4.5: MakeTime(Result(4), Result(5), 0, 0)
		{"15.9.4.5 Date.UTC with five arguments", output(`print(Date.UTC(2001, 5, 15, 10, 20) == Date.UTC(2001, 5, 15, 10, 20, 0, 0), Date.UTC(1, 0, 1, 0, 0) == Date.UTC(1901, 0, 1, 0, 0, 0, 0))`, "true true\n")},
		// 15.9.4.6: MakeTime(Result(4), 0, 0, 0)
		{"15.9.4.6 Date.UTC with four arguments", output(`print(Date.UTC(2001, 5, 15, 10) == Date.UTC(2001, 5, 15, 10, 0, 0, 0), Date.UTC(1, 0, 1, 0) == Date.UTC(1901, 0, 1, 0, 0, 0, 0))`, "true true\n")},
		// 15.9.4.7: MakeDate(Result(5), 0)
		{"15.9.4.7 Date.UTC with three arguments", output(`print(Date.UTC(2001, 5, 15) == Date.UTC(2001, 5, 15, 0, 0, 0, 0), Date.UTC(1, 0, 1) == Date.UTC(1901, 0, 1, 0, 0, 0, 0), Date.UTC(2000, 0, 1))`, "true true 946684800000\n")},

		// 15.9.5: "The Date prototype object is itself a Date object (its [[Class]] is "Date") whose value is NaN."
		{"15.9.5 Date.prototype is a Date object holding NaN", output(classOfJS+`print(Date.prototype.getTime(), classOf(Date.prototype))`, "NaN [object Date]\n")},
		// 15.9.5: "The value of the internal [[Prototype]] property of the Date prototype object is the Object prototype object"
		{"15.9.5 Date.prototype inherits from Object.prototype", output(`Object.prototype.marker = "op"; print(Date.prototype.marker)`, "op\n")},
		// 15.9.5: "it is an error if this does not refer to an object for which the value of the internal [[Class]] property is "Date""
		{"15.9.5 a getter on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Object(); o.getFullYear = Date.prototype.getFullYear; print("before"); o.getFullYear()`, "before\n")},
		// 15.9.5: the same holds for the setters
		{"15.9.5 a setter on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Number(0); o.setUTCHours = Date.prototype.setUTCHours; print("before"); o.setUTCHours(1)`, "before\n")},

		// 15.9.5.1: "The initial value of Date.prototype.constructor is the built-in Date constructor."
		{"15.9.5.1 Date.prototype.constructor is Date", output(`print(Date.prototype.constructor == Date, new Date(0).constructor == Date)`, "true true\n")},
		// 15.9.5.2: "This function returns a string value."
		{"15.9.5.2 toString returns a string that parse reads back",
			output(`var d = new Date(Date.UTC(2001, 5, 15, 10, 20, 30)); print(typeof d.toString(), Date.parse(d.toString()) == d.getTime())`, "string true\n")},
		// 15.9.5.2 with §8.6.2.6: a Date converts to a primitive through toString when there is no hint
		{"15.9.5.2 a Date in + converts through toString", output(`var d = new Date(0); print(d + "" == d.toString(), typeof (d + 0))`, "true string\n")},
		// 15.9.5.2: "it generates a runtime error if its this value is not a Date object"
		{"15.9.5.2 toString on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Object(); o.toString = Date.prototype.toString; print("before"); o.toString()`, "before\n")},
		// 15.9.5.3: "The valueOf function returns a number, which is this time value."
		{"15.9.5.3 valueOf returns the time value", output(`var d = new Date(86400000); print(d.valueOf(), typeof d.valueOf(), d - 0, new Date(NaN).valueOf())`, "86400000 number 86400000 NaN\n")},
		// 15.9.5.3: "it generates a runtime error if its this value is not a Date object"
		{"15.9.5.3 valueOf on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Number(5); o.valueOf = Date.prototype.valueOf; print("before"); o.valueOf()`, "before\n")},
		// 15.9.5.4 step 2: "Return this time value."
		{"15.9.5.4 getTime returns the time value", output(`print(new Date(1234).getTime(), new Date(NaN).getTime())`, "1234 NaN\n")},
		// 15.9.5.4 step 1: "If the this value is not an object whose [[Class]] property is "Date", generate a runtime error."
		{"15.9.5.4 getTime on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Object(); o.getTime = Date.prototype.getTime; print("before"); o.getTime()`, "before\n")},

		// 15.9.5.5 step 3: "Return YearFromTime(LocalTime(t)) - 1900."
		{"15.9.5.5 getYear is the year minus 1900",
			output(dateJS+`print(base().getYear(), new Date(Date.UTC(1950, 0, 1)).getYear(), new Date(Date.UTC(1800, 0, 1)).getYear(), new Date(NaN).getYear())`, "101 50 -100 NaN\n")},
		// 15.9.5.6: "Return YearFromTime(LocalTime(t))." "If t is NaN, return NaN."
		{"15.9.5.6 getFullYear", output(dateJS+`print(base().getFullYear(), new Date(-1).getFullYear(), new Date(NaN).getFullYear())`, "2001 1969 NaN\n")},
		// 15.9.5.7: "Return YearFromTime(t)."
		{"15.9.5.7 getUTCFullYear", output(dateJS+`print(base().getUTCFullYear(), new Date(-1).getUTCFullYear(), new Date(NaN).getUTCFullYear())`, "2001 1969 NaN\n")},
		// 15.9.5.8: "Return MonthFromTime(LocalTime(t))."
		{"15.9.5.8 getMonth", output(dateJS+`print(base().getMonth(), new Date(-1).getMonth(), new Date(NaN).getMonth())`, "5 11 NaN\n")},
		// 15.9.5.9: "Return MonthFromTime(t)."
		{"15.9.5.9 getUTCMonth", output(dateJS+`print(base().getUTCMonth(), new Date(-1).getUTCMonth(), new Date(NaN).getUTCMonth())`, "5 11 NaN\n")},
		// 15.9.5.10: "Return DateFromTime(LocalTime(t))."
		{"15.9.5.10 getDate", output(dateJS+`print(base().getDate(), new Date(-1).getDate(), new Date(NaN).getDate())`, "15 31 NaN\n")},
		// 15.9.5.11: "Return DateFromTime(t)."
		{"15.9.5.11 getUTCDate", output(dateJS+`print(base().getUTCDate(), new Date(-1).getUTCDate(), new Date(NaN).getUTCDate())`, "15 31 NaN\n")},
		// 15.9.5.12: "Return WeekDay(LocalTime(t))."
		{"15.9.5.12 getDay", output(dateJS+`print(base().getDay(), new Date(0).getDay(), new Date(NaN).getDay())`, "5 4 NaN\n")},
		// 15.9.5.13: "Return WeekDay(t)."
		{"15.9.5.13 getUTCDay", output(dateJS+`print(base().getUTCDay(), new Date(-1).getUTCDay(), new Date(NaN).getUTCDay())`, "5 3 NaN\n")},
		// 15.9.5.14: "Return HourFromTime(LocalTime(t))."
		{"15.9.5.14 getHours", output(dateJS+`print(base().getHours(), new Date(-1).getHours(), new Date(NaN).getHours())`, "10 23 NaN\n")},
		// 15.9.5.15: "Return HourFromTime(t)."
		{"15.9.5.15 getUTCHours", output(dateJS+`print(base().getUTCHours(), new Date(-1).getUTCHours(), new Date(NaN).getUTCHours())`, "10 23 NaN\n")},
		// 15.9.5.16: "Return MinFromTime(LocalTime(t))."
		{"15.9.5.16 getMinutes", output(dateJS+`print(base().getMinutes(), new Date(-1).getMinutes(), new Date(NaN).getMinutes())`, "20 59 NaN\n")},
		// 15.9.5.17: "Return MinFromTime(t)."
		{"15.9.5.17 getUTCMinutes", output(dateJS+`print(base().getUTCMinutes(), new Date(-1).getUTCMinutes(), new Date(NaN).getUTCMinutes())`, "20 59 NaN\n")},
		// 15.9.5.18: "Return SecFromTime(LocalTime(t))."
		{"15.9.5.18 getSeconds", output(dateJS+`print(base().getSeconds(), new Date(-1).getSeconds(), new Date(NaN).getSeconds())`, "30 59 NaN\n")},
		// 15.9.5.19: "Return SecFromTime(t)."
		{"15.9.5.19 getUTCSeconds", output(dateJS+`print(base().getUTCSeconds(), new Date(-1).getUTCSeconds(), new Date(NaN).getUTCSeconds())`, "30 59 NaN\n")},
		// 15.9.5.20: "Return msFromTime(LocalTime(t))."
		{"15.9.5.20 getMilliseconds", output(dateJS+`print(base().getMilliseconds(), new Date(-1).getMilliseconds(), new Date(NaN).getMilliseconds())`, "400 999 NaN\n")},
		// 15.9.5.21: "Return msFromTime(t)."
		{"15.9.5.21 getUTCMilliseconds", output(dateJS+`print(base().getUTCMilliseconds(), new Date(-1).getUTCMilliseconds(), new Date(NaN).getUTCMilliseconds())`, "400 999 NaN\n")},
		// 15.9.5.22: "Return (t - LocalTime(t)) / msPerMinute." "If t is NaN, return NaN."
		{"15.9.5.22 getTimezoneOffset is 0 under TZ=UTC", output(dateJS+`print(base().getTimezoneOffset(), new Date(0).getTimezoneOffset(), new Date(NaN).getTimezoneOffset())`, "0 0 NaN\n")},

		// 15.9.5.23 steps 2-5: ToNumber, TimeClip, set and return the value
		{"15.9.5.23 setTime sets and returns the time value",
			output(dateJS+`var d = base(); var r = d.setTime("86400000"); print(r, utc(d), d.setTime(1.9), d.setTime(8.64e15 + 1), d.getTime())`, "86400000 1970-1-2 0:0:0.0 1 NaN NaN\n")},
		// 15.9.5.23 step 1: "If the this value is not a Date object, generate a runtime error."
		{"15.9.5.23 setTime on an object that is not a Date is a runtime error",
			runtimeError(`var o = new Object(); o.setTime = Date.prototype.setTime; print("before"); o.setTime(0)`, "before\n")},
		// 15.9.5.24: MakeTime(HourFromTime(t), MinFromTime(t), SecFromTime(t), Result(2)) in local time
		{"15.9.5.24 setMilliseconds", output(dateJS+`var d = base(); var r = d.setMilliseconds(5); print(local(d), r == d.getTime()); d.setMilliseconds(1500); print(local(d))`, "2001-6-15 10:20:30.5 true\n2001-6-15 10:20:31.500\n")},
		// 15.9.5.25: MakeTime(HourFromTime(t), MinFromTime(t), SecFromTime(t), Result(2)) in UTC
		{"15.9.5.25 setUTCMilliseconds", output(dateJS+`var d = base(); var r = d.setUTCMilliseconds(5); print(utc(d), r == d.getTime()); d.setUTCMilliseconds(-1); print(utc(d))`, "2001-6-15 10:20:30.5 true\n2001-6-15 10:20:29.999\n")},
		// 15.9.5.26: "If ms is not specified, this behaves as if ms were specified with the value getMilliseconds( )."
		{"15.9.5.26 setSeconds with and without ms",
			output(dateJS+`var d = base(); var r = d.setSeconds(5); print(local(d), r == d.getTime()); d.setSeconds(6, 7); print(local(d))`, "2001-6-15 10:20:5.400 true\n2001-6-15 10:20:6.7\n")},
		// 15.9.5.27: "If ms is not specified, this behaves as if ms were specified with the value getUTCMilliseconds( )."
		{"15.9.5.27 setUTCSeconds with and without ms",
			output(dateJS+`var d = base(); var r = d.setUTCSeconds(5); print(utc(d), r == d.getTime()); d.setUTCSeconds(61, 7); print(utc(d))`, "2001-6-15 10:20:5.400 true\n2001-6-15 10:21:1.7\n")},
		// 15.9.5.28: "If sec is not specified ... getSeconds ( )." "If ms is not specified ... getMilliseconds( )."
		{"15.9.5.28 setMinutes with one, two and three arguments",
			output(dateJS+`var d = base(); var r = d.setMinutes(1); print(local(d), r == d.getTime()); d.setMinutes(2, 3); print(local(d)); d.setMinutes(4, 5, 6); print(local(d))`,
				"2001-6-15 10:1:30.400 true\n2001-6-15 10:2:3.400\n2001-6-15 10:4:5.6\n")},
		// 15.9.5.29: "If sec is not specified ... getUTCSeconds ( )." "If ms is not specified ... getUTCMilliseconds( )."
		{"15.9.5.29 setUTCMinutes with one, two and three arguments",
			output(dateJS+`var d = base(); var r = d.setUTCMinutes(1); print(utc(d), r == d.getTime()); d.setUTCMinutes(2, 3); print(utc(d)); d.setUTCMinutes(4, 5, 6); print(utc(d))`,
				"2001-6-15 10:1:30.400 true\n2001-6-15 10:2:3.400\n2001-6-15 10:4:5.6\n")},
		// 15.9.5.30: "If min is not specified ... getMinutes( )." and so on
		{"15.9.5.30 setHours with one to four arguments",
			output(dateJS+`var d = base(); var r = d.setHours(1); print(local(d), r == d.getTime()); d.setHours(2, 3); print(local(d)); d.setHours(4, 5, 6); print(local(d)); d.setHours(7, 8, 9, 10); print(local(d)); d.setHours(25); print(local(d))`,
				"2001-6-15 1:20:30.400 true\n2001-6-15 2:3:30.400\n2001-6-15 4:5:6.400\n2001-6-15 7:8:9.10\n2001-6-16 1:8:9.10\n")},
		// 15.9.5.31: "If min is not specified ... getUTCMinutes( )." and so on
		{"15.9.5.31 setUTCHours with one to four arguments",
			output(dateJS+`var d = base(); var r = d.setUTCHours(1); print(utc(d), r == d.getTime()); d.setUTCHours(2, 3); print(utc(d)); d.setUTCHours(4, 5, 6); print(utc(d)); d.setUTCHours(7, 8, 9, 10); print(utc(d)); d.setUTCHours(-1); print(utc(d))`,
				"2001-6-15 1:20:30.400 true\n2001-6-15 2:3:30.400\n2001-6-15 4:5:6.400\n2001-6-15 7:8:9.10\n2001-6-14 23:8:9.10\n")},
		// 15.9.5.32: MakeDay(YearFromTime(t), MonthFromTime(t), Result(2)) with TimeWithinDay(t), in local time
		{"15.9.5.32 setDate", output(dateJS+`var d = base(); var r = d.setDate(1); print(local(d), r == d.getTime()); d.setDate(31); print(local(d))`, "2001-6-1 10:20:30.400 true\n2001-7-1 10:20:30.400\n")},
		// 15.9.5.33: MakeDay(YearFromTime(t), MonthFromTime(t), Result(2)) with TimeWithinDay(t), in UTC
		{"15.9.5.33 setUTCDate", output(dateJS+`var d = base(); var r = d.setUTCDate(1); print(utc(d), r == d.getTime()); d.setUTCDate(0); print(utc(d))`, "2001-6-1 10:20:30.400 true\n2001-5-31 10:20:30.400\n")},
		// 15.9.5.34: "If date is not specified, this behaves as if date were specified with the value getDate( )."
		{"15.9.5.34 setMonth with and without date",
			output(dateJS+`var d = base(); var r = d.setMonth(0); print(local(d), r == d.getTime()); d.setMonth(1, 28); print(local(d)); var e = new Date(2001, 0, 31); e.setMonth(1); print(local(e))`,
				"2001-1-15 10:20:30.400 true\n2001-2-28 10:20:30.400\n2001-3-3 0:0:0.0\n")},
		// 15.9.5.35: "If date is not specified, this behaves as if date were specified with the value getUTCDate( )."
		{"15.9.5.35 setUTCMonth with and without date",
			output(dateJS+`var d = base(); var r = d.setUTCMonth(11); print(utc(d), r == d.getTime()); d.setUTCMonth(12, 2); print(utc(d))`, "2001-12-15 10:20:30.400 true\n2002-1-2 10:20:30.400\n")},
		// 15.9.5.36: "If mon is not specified ... getMonth( )." "If date is not specified ... getDate( )."
		{"15.9.5.36 setFullYear with one, two and three arguments",
			output(dateJS+`var d = base(); var r = d.setFullYear(1999); print(local(d), r == d.getTime()); d.setFullYear(1998, 0); print(local(d)); d.setFullYear(1997, 1, 2); print(local(d)); d.setFullYear(5); print(local(d))`,
				"1999-6-15 10:20:30.400 true\n1998-1-15 10:20:30.400\n1997-2-2 10:20:30.400\n5-2-2 10:20:30.400\n")},
		// 15.9.5.36 step 1: "but if this time value is NaN, let t be +0"
		{"15.9.5.36 setFullYear on an invalid date starts from time value 0", output(dateJS+`var d = new Date(NaN); d.setFullYear(2000); print(local(d))`, "2000-1-1 0:0:0.0\n")},
		// 15.9.5.37: "If mon is not specified ... getUTCMonth( )." "If date is not specified ... getUTCDate( )."
		{"15.9.5.37 setUTCFullYear with one, two and three arguments",
			output(dateJS+`var d = base(); var r = d.setUTCFullYear(1999); print(utc(d), r == d.getTime()); d.setUTCFullYear(1998, 0); print(utc(d)); d.setUTCFullYear(1997, 1, 2); print(utc(d)); var e = new Date(Date.UTC(2000, 1, 29)); e.setUTCFullYear(2001); print(utc(e))`,
				"1999-6-15 10:20:30.400 true\n1998-1-15 10:20:30.400\n1997-2-2 10:20:30.400\n2001-3-1 0:0:0.0\n")},
		// 15.9.5.37 step 1: "but if this time value is NaN, let t be +0"
		{"15.9.5.37 setUTCFullYear on an invalid date starts from time value 0", output(dateJS+`var d = new Date(NaN); d.setUTCFullYear(2000, 5); print(utc(d))`, "2000-6-1 0:0:0.0\n")},
		// 15.9.5.38 step 4: "If ... 0<=ToInteger(Result(2))<=99 then Result(4) is ToInteger(Result(2)) + 1900. Otherwise Result(4) is Result(2)."
		{"15.9.5.38 setYear reads a year from 0 to 99 as 1900 to 1999",
			output(dateJS+`var d = base(); var r = d.setYear(99); print(local(d), r == d.getTime()); d.setYear(2005); print(local(d)); d.setYear(0); print(d.getFullYear()); d.setYear(100); print(d.getFullYear())`,
				"1999-6-15 10:20:30.400 true\n2005-6-15 10:20:30.400\n1900\n100\n")},
		// 15.9.5.38 step 3: "If Result(2) is NaN, set the [[Value]] property of the this value to NaN and return NaN."
		{"15.9.5.38 setYear(NaN) makes the date invalid", output(dateJS+`var d = base(); print(d.setYear(NaN), d.getTime())`, "NaN NaN\n")},
		// 15.9.5.38 step 1: "but if this time value is NaN, let t be +0"
		{"15.9.5.38 setYear on an invalid date starts from time value 0", output(dateJS+`var d = new Date(NaN); d.setYear(99); print(local(d))`, "1999-1-1 0:0:0.0\n")},
		// 15.9.5.24-35 with §15.9.1.11-13: the other setters keep an invalid date invalid
		{"15.9.5 setters other than setFullYear and setYear keep NaN",
			output(`var d = new Date(NaN); print(d.setMilliseconds(1), d.setUTCSeconds(1), d.setMinutes(1), d.setUTCHours(1), d.setDate(1), d.setUTCMonth(1), d.getTime())`, "NaN NaN NaN NaN NaN NaN NaN\n")},
		// 15.9.5.24-38 with §15.9.1.14: a setter result outside the time range is NaN
		{"15.9.5 a setter that leaves the time range gives NaN", output(`var d = new Date(8.64e15); print(d.setUTCMilliseconds(1), d.getTime())`, "NaN NaN\n")},

		// 15.9.5.39: "This function returns a string value."
		{"15.9.5.39 toLocaleString returns a string", output(`print(typeof new Date(0).toLocaleString())`, "string\n")},
		// 15.9.5.40: "This function returns a string value." that Date.parse reads back (§15.9.4.2)
		{"15.9.5.40 toUTCString returns a string that parse reads back",
			output(`var d = new Date(Date.UTC(2001, 5, 15, 10, 20, 30)); print(typeof d.toUTCString(), Date.parse(d.toUTCString()) == d.getTime())`, "string true\n")},
		// 15.9.5.41: "the same function object that is the initial value of Date.prototype.toUTCString"
		{"15.9.5.41 toGMTString is toUTCString", output(`print(Date.prototype.toGMTString == Date.prototype.toUTCString)`, "true\n")},

		// 15.9.6: "Date instances have no special properties beyond those inherited from the Date prototype object."
		{"15.9.6 a Date object has no own properties", output(`var n = 0; for (var p in new Date(0)) n++; print(n)`, "0\n")},
	})
}
