package tests

import "testing"

// Chapter 9, Type Conversion. The conversion operators have no entry point
// of their own; each row observes one through an operator or built-in that
// applies it: ! (ToBoolean), Number and unary + (ToNumber), String
// (ToString), charAt (ToInteger), | (ToInt32), >>> (ToUint32),
// String.fromCharCode (ToUint16), property access and with (ToObject).
// JS string escapes: ES1 has no \v, so a vertical tab is written \x0B.

func TestS9_1_ToPrimitive(t *testing.T) {
	runRows(t, []row{
		// 9.1: Undefined, Null, Boolean, Number, String: "The result equals the input argument (no conversion)."
		{"9.1 a primitive value converts to itself", output(`print("1" + 2, "a" + null, "a" + true, "a" + void 0)`, "12 anull atrue aundefined\n")},
		// 9.1 Object: "Return a default value for the Object ... calling the internal [[DefaultValue]] method of the object"
		{"9.1 an object converts through [[DefaultValue]]",
			output(`function vo() { return 5 } var o = new Object(); o.valueOf = vo; print(o * 2, "" + o)`, "10 5\n")},
		// 9.1 Object: "passing the optional hint PreferredType" (String for ToString, §9.8)
		{"9.1 an object converts with the hint of its caller",
			output(`function vo() { return 5 } function ts() { return "s" } var o = new Object(); o.valueOf = vo; o.toString = ts; print(o * 2, String(o))`, "10 s\n")},
		// 9.1 Object: "If the return value is of type Object or Reference, a runtime error is generated." ([[DefaultValue]] step 9)
		{"9.1 an object with no primitive default value cannot be converted",
			runtimeError(`function self() { return this } var o = new Object(); o.valueOf = self; o.toString = self; print("before"); print(o * 2)`, "before\n")},
	})
}

func TestS9_2_ToBoolean(t *testing.T) {
	runRows(t, []row{
		// 9.2 Undefined: false
		{"9.2 undefined converts to false", output(`print(!!void 0)`, "false\n")},
		// 9.2 Null: false
		{"9.2 null converts to false", output(`print(!!null)`, "false\n")},
		// 9.2 Boolean: "The result equals the input argument (no conversion)."
		{"9.2 a boolean converts to itself", output(`print(!!true, !!false)`, "true false\n")},
		// 9.2 Number: "The result is false if the argument is +0, -0, or NaN"
		{"9.2 zero and NaN convert to false", output(`print(!!0, !!-0, !!NaN)`, "false false false\n")},
		// 9.2 Number: "otherwise the result is true"
		{"9.2 any other number converts to true", output(`print(!!1, !!-1, !!0.5, !!Infinity, !!-Infinity, !!5e-324)`, "true true true true true true\n")},
		// 9.2 String: "The result is false if the argument is the empty string (its length is zero)"
		{"9.2 the empty string converts to false", output(`print(!!"")`, "false\n")},
		// 9.2 String: "otherwise the result is true"
		{"9.2 any other string converts to true", output(`print(!!"0", !!" ", !!"false", !!"\0")`, "true true true true\n")},
		// 9.2 Object: true
		{"9.2 every object converts to true, even a wrapper of a false value",
			output(`print(!!new Object(), !!new Boolean(false), !!new Number(0), !!new String(""))`, "true true true true\n")},
	})
}

func TestS9_3_ToNumber(t *testing.T) {
	runRows(t, []row{
		// 9.3 Undefined: NaN
		{"9.3 undefined converts to NaN", output(`print(+void 0)`, "NaN\n")},
		// 9.3 Null: +0
		{"9.3 null converts to +0", output(`print(+null, 1 / +null)`, "0 Infinity\n")},
		// 9.3 Boolean: "The result is 1 if the argument is true. The result is +0 if the argument is false."
		{"9.3 true converts to 1 and false to +0", output(`print(+true, +false, 1 / +false)`, "1 0 Infinity\n")},
		// 9.3 Number: "The result equals the input argument (no conversion)."
		{"9.3 a number converts to itself", output(`print(+1.5, +-2, 1 / +-0)`, "1.5 -2 -Infinity\n")},
		// 9.3 Object: "Call ToPrimitive(input argument, hint Number). Call ToNumber(Result(1))."
		{"9.3 an object converts through its primitive value", output(`print(+new Number(7), +new String(" 12 "), +new Boolean(true))`, "7 12 1\n")},
		// 9.3 Object: a primitive string from the object is then converted with §9.3.1
		{"9.3 an object whose default value is a non-numeric string converts to NaN", output(`print(+new Object())`, "NaN\n")},
	})
}

func TestS9_3_1_ToNumberAppliedToTheStringType(t *testing.T) {
	runRows(t, []row{
		// 9.3.1: "The MV of StringNumericLiteral ::: (an empty character sequence) is 0."
		{"9.3.1 the empty string converts to 0", output(`print(Number(""), 1 / Number(""))`, "0 Infinity\n")},
		// 9.3.1: "The MV of StringNumericLiteral ::: StrWhiteSpace is 0."
		{"9.3.1 a string of only white space converts to 0", output(`print(Number("   "), Number("\t\n\r"))`, "0 0\n")},
		// 9.3.1: StrWhiteSpaceChar ::: <TAB> <SP> <FF> <VT> <CR> <LF>
		{"9.3.1 TAB, SP, FF, VT, CR and LF may surround a number", output(`print(Number("\t \f\x0B\r\n12\t \f\x0B\r\n"))`, "12\n")},
		// 9.3.1: StrWhiteSpaceChar is only those six characters, so a no-break space is not white space
		{"9.3.1 a no-break space is not white space", output(`print(Number("\u00A05"), Number("5\u00A0"))`, "NaN NaN\n")},
		// 9.3.1: StrWhiteSpaceChar does not include the Unicode line separator
		{"9.3.1 a line separator is not white space", output(`print(Number("\u20285"))`, "NaN\n")},
		// 9.3.1: "If the grammar cannot interpret the string as an expansion of StringNumericLiteral, then the result of ToNumber is NaN."
		{"9.3.1 a string that is not a numeric literal converts to NaN", output(`print(Number("abc"), Number("12px"), Number("1 2"), Number("1.2.3"))`, "NaN NaN NaN NaN\n")},
		// 9.3.1: "The MV of StrNumericLiteral ::: + StrDecimalLiteral is the MV of StrDecimalLiteral."
		{"9.3.1 a decimal string may have a + sign", output(`print(Number("+12"), Number(" +1.5 "))`, "12 1.5\n")},
		// 9.3.1: "The MV of StrNumericLiteral ::: - StrDecimalLiteral is the negative of the MV of StrDecimalLiteral."
		{"9.3.1 a decimal string may have a - sign", output(`print(Number("-12"), Number(" -1.5 "))`, "-12 -1.5\n")},
		// 9.3.1: "If the MV is 0, then the rounded value is +0 unless the first non-whitespace character ... is '-', in which case ... -0"
		{"9.3.1 a string for zero with a - sign converts to -0", output(`print(1 / Number("-0"), 1 / Number(" -0.0 "), 1 / Number("0"), 1 / Number("+0"))`, "-Infinity -Infinity Infinity Infinity\n")},
		// 9.3.1: only one sign is allowed
		{"9.3.1 a string with two signs converts to NaN", output(`print(Number("--1"), Number("+-1"))`, "NaN NaN\n")},
		// 9.3.1: "The MV of StrDecimalLiteral ::: Infinity is 10^10000 (a value so large that it will round to +Infinity)."
		{"9.3.1 the string Infinity converts to Infinity", output(`print(Number("Infinity"), Number("+Infinity"), Number("-Infinity"), Number(" Infinity "))`, "Infinity Infinity -Infinity Infinity\n")},
		// 9.3.1: Infinity must be spelled exactly
		{"9.3.1 Infinity is case sensitive", output(`print(Number("infinity"), Number("INFINITY"), Number("Inf"))`, "NaN NaN NaN\n")},
		// 9.3.1: "StrDecimalLiteral ::: DecimalDigits . is the MV of DecimalDigits"
		{"9.3.1 a decimal string may end with a point", output(`print(Number("12."))`, "12\n")},
		// 9.3.1: "StrDecimalLiteral ::: DecimalDigits . DecimalDigits is the MV of the first DecimalDigits plus (... times 10^-n)"
		{"9.3.1 a decimal string with a fraction", output(`print(Number("3.25"))`, "3.25\n")},
		// 9.3.1: "StrDecimalLiteral ::: DecimalDigits . ExponentPart is the MV of DecimalDigits times 10^e"
		{"9.3.1 a point may be followed directly by an exponent", output(`print(Number("2.e3"))`, "2000\n")},
		// 9.3.1: "StrDecimalLiteral ::: DecimalDigits . DecimalDigits ExponentPart"
		{"9.3.1 a fraction with an exponent", output(`print(Number("1.5e3"), Number("2.5E-2"))`, "1500 0.025\n")},
		// 9.3.1: "StrDecimalLiteral ::: . DecimalDigits is the MV of DecimalDigits times 10^-n"
		{"9.3.1 a decimal string may start with a point", output(`print(Number(".5"), Number("-.5"))`, "0.5 -0.5\n")},
		// 9.3.1: "StrDecimalLiteral ::: . DecimalDigits ExponentPart is the MV of DecimalDigits times 10^(e-n)"
		{"9.3.1 a leading point with an exponent", output(`print(Number(".5e1"))`, "5\n")},
		// 9.3.1: "StrDecimalLiteral ::: DecimalDigits ExponentPart is the MV of DecimalDigits times 10^e"
		{"9.3.1 an integer string with an exponent", output(`print(Number("1e3"), Number("1E+2"), Number("1e-2"))`, "1000 100 0.01\n")},
		// 9.3.1: a lone point, or an exponent without digits, is not a StrDecimalLiteral
		{"9.3.1 a point or an exponent without digits converts to NaN", output(`print(Number("."), Number("1e"), Number("e5"), Number("1e+"))`, "NaN NaN NaN NaN\n")},
		// 9.3.1: "A StringNumericLiteral that is decimal may have any number of leading 0 digits."
		{"9.3.1 leading zeros are allowed", output(`print(Number("007"), Number("000.5"))`, "7 0.5\n")},
		// 9.3.1: "A StringNumericLiteral may not use octal notation."
		{"9.3.1 a leading zero does not make a string octal", output(`print(Number("010"), Number("0777"))`, "10 777\n")},
		// 9.3.1: "The MV of HexIntegerLiteral ::: 0x HexDigit" ... "(the MV of HexIntegerLiteral times 16) plus the MV of HexDigit"
		{"9.3.1 a hexadecimal string", output(`print(Number("0x1F"), Number("0X1f"), Number(" 0xff "))`, "31 31 255\n")},
		// 9.3.1: HexIntegerLiteral needs a digit and takes no sign (only StrDecimalLiteral may be signed)
		{"9.3.1 a hexadecimal string without digits or with a sign converts to NaN", output(`print(Number("0x"), Number("-0x10"), Number("+0x10"), Number("0xG"))`, "NaN NaN NaN NaN\n")},
		// 9.3.1: StrNumericLiteral is decimal or hexadecimal only, so 0b and 0o have no meaning
		{"9.3.1 binary and octal prefixes are not numeric", output(`print(Number("0b11"), Number("0o17"))`, "NaN NaN\n")},
		// 9.3.1: "the rounded value must be the number value for the MV (in the sense defined in section 8.4)"
		{"9.3.1 the value is rounded to the nearest number", output(`print(Number("9007199254740993"), Number("0.1"))`, "9007199254740992 0.1\n")},
		// 9.3.1: "more than 20 significant digits, in which case the number value may be either ..."
		{"9.3.1 digits after the 20th significant digit may be ignored", output(`print(Number("1.000000000000000000000001"))`, "1\n")},
	})
}

func TestS9_4_ToInteger(t *testing.T) {
	runRows(t, []row{
		// 9.4 step 2: "If Result(1) is NaN, return +0."
		{"9.4 NaN converts to 0", output(`print("abc".charAt(NaN), "abc".charAt("x"))`, "a a\n")},
		// 9.4 step 4: "Compute sign(Result(1)) * floor(abs(Result(1)))."
		{"9.4 a positive fraction is rounded down", output(`print("abc".charAt(1.9), "abc".charAt(2.1))`, "b c\n")},
		// 9.4 step 4: sign * floor(abs): a negative fraction is rounded towards zero
		{"9.4 a negative fraction is rounded towards zero", output(`print("abc".charAt(-0.9), "abc".charAt(-1.5) == "")`, "a true\n")},
		// 9.4 step 3: "If Result(1) is +0, -0, +Infinity, or -Infinity, return Result(1)."
		{"9.4 infinities are kept", output(`print("abc".charAt(Infinity) == "", "abc".charAt(-Infinity) == "", "abc".substring(Infinity), "abc".substring(-Infinity))`, "true true  abc\n")},
		// 9.4 step 1: "Call ToNumber on the input argument."
		{"9.4 the argument is first converted to a number", output(`print("abc".charAt("2"), "abc".charAt(true))`, "c b\n")},
	})
}

func TestS9_5_ToInt32(t *testing.T) {
	runRows(t, []row{
		// 9.5 step 2: "If Result(1) is NaN, +0, -0, +Infinity, or -Infinity, return +0."
		{"9.5 NaN, zeros and infinities convert to +0", output(`print(NaN | 0, Infinity | 0, -Infinity | 0, 1 / (-0 | 0))`, "0 0 0 Infinity\n")},
		// 9.5 step 3: "Compute sign(Result(1)) * floor(abs(Result(1)))."
		{"9.5 fractions are truncated towards zero", output(`print(1.9 | 0, -1.9 | 0)`, "1 -1\n")},
		// 9.5 steps 4-5: values in range are kept
		{"9.5 values in range are unchanged", output(`print(2147483647 | 0, -2147483648 | 0)`, "2147483647 -2147483648\n")},
		// 9.5 step 5: "If Result(4) is greater than or equal to 2^31, return Result(5) - 2^32"
		{"9.5 values from 2^31 wrap around to negative", output(`print(2147483648 | 0, 4294967295 | 0)`, "-2147483648 -1\n")},
		// 9.5 step 4: "Compute Result(3) modulo 2^32"
		{"9.5 values are taken modulo 2^32", output(`print(4294967296 | 0, 4294967297 | 0, -2147483649 | 0, -4294967297 | 0)`, "0 1 2147483647 -1\n")},
		// 9.5 steps 3-4: truncation happens before the modulo
		{"9.5 large fractional values are truncated then wrapped", output(`print(4294967297.9 | 0, -4294967297.9 | 0)`, "1 -1\n")},
		// 9.5: "ToInt32(ToUint32(x)) is equal to ToInt32(x) for all values of x"
		{"9.5 converting through ToUint32 first changes nothing", output(`print((-1 >>> 0) | 0, (3000000000 >>> 0) | 0, 3000000000 | 0)`, "-1 -1294967296 -1294967296\n")},
	})
}

func TestS9_6_ToUint32(t *testing.T) {
	runRows(t, []row{
		// 9.6 step 2: "If Result(1) is NaN, +0, -0, +Infinity, or -Infinity, return +0."
		{"9.6 NaN, zeros and infinities convert to +0", output(`print(NaN >>> 0, Infinity >>> 0, -Infinity >>> 0, 1 / (-0 >>> 0))`, "0 0 0 Infinity\n")},
		// 9.6 step 3: "Compute sign(Result(1)) * floor(abs(Result(1)))."
		{"9.6 fractions are truncated towards zero", output(`print(1.9 >>> 0, -1.9 >>> 0)`, "1 4294967295\n")},
		// 9.6 step 4: "Compute Result(3) modulo 2^32"
		{"9.6 values are taken modulo 2^32", output(`print(4294967295 >>> 0, 4294967296 >>> 0, 4294967297 >>> 0, 2147483648 >>> 0)`, "4294967295 0 1 2147483648\n")},
		// 9.6 step 4: "a finite integer value k of Number type with positive sign"
		{"9.6 negative values wrap around to positive", output(`print(-1 >>> 0, -2147483648 >>> 0, -4294967296 >>> 0)`, "4294967295 2147483648 0\n")},
	})
}

func TestS9_7_ToUint16(t *testing.T) {
	runRows(t, []row{
		// 9.7 step 2: "If Result(1) is NaN, +0, -0, +Infinity, or -Infinity, return +0."
		{"9.7 NaN and infinities convert to 0",
			output(`print(String.fromCharCode(NaN).charCodeAt(0), String.fromCharCode(Infinity).charCodeAt(0), String.fromCharCode(-Infinity).charCodeAt(0))`, "0 0 0\n")},
		// 9.7 step 3: "Compute sign(Result(1)) * floor(abs(Result(1)))."
		{"9.7 fractions are truncated", output(`print(String.fromCharCode(65.9).charCodeAt(0))`, "65\n")},
		// 9.7 step 4: "Compute Result(3) modulo 2^16"
		{"9.7 values are taken modulo 2^16",
			output(`print(String.fromCharCode(65536 + 65).charCodeAt(0), String.fromCharCode(-1).charCodeAt(0), String.fromCharCode(65535).charCodeAt(0))`, "65 65535 65535\n")},
	})
}

func TestS9_8_ToString(t *testing.T) {
	runRows(t, []row{
		// 9.8 Undefined: "undefined"
		{"9.8 undefined converts to the string undefined", output(`print(String(void 0) + "!")`, "undefined!\n")},
		// 9.8 Null: "null"
		{"9.8 null converts to the string null", output(`print(String(null) + "!")`, "null!\n")},
		// 9.8 Boolean: "If the argument is true, then the result is "true". If the argument is false, then the result is "false"."
		{"9.8 booleans convert to true and false", output(`print(String(true) + "!", String(false) + "!")`, "true! false!\n")},
		// 9.8 String: "Return the input argument (no conversion)"
		{"9.8 a string converts to itself", output(`print(String("abc") + "!", String("").length)`, "abc! 0\n")},
		// 9.8 Object: "Call ToPrimitive(input argument, hint String). Call ToString(Result(1))."
		{"9.8 an object converts through toString first",
			output(`function ts() { return 12 } function vo() { return 99 } var o = new Object(); o.toString = ts; o.valueOf = vo; print(String(o) + "!")`, "12!\n")},
	})
}

func TestS9_8_1_ToStringAppliedToTheNumberType(t *testing.T) {
	runRows(t, []row{
		// 9.8.1 step 1: "If m is NaN, return the string "NaN"."
		{"9.8.1 NaN converts to NaN", output(`print(String(NaN) + "!")`, "NaN!\n")},
		// 9.8.1 step 2: "If m is +0 or -0, return the string "0"."
		{"9.8.1 both zeros convert to 0", output(`print(String(0) + "!", String(-0) + "!")`, "0! 0!\n")},
		// 9.8.1 step 3: "If m is less than zero, return the string concatenation of the string "-" and ToString(-m)."
		{"9.8.1 a negative number gets a minus sign", output(`print(String(-1.5), String(-1e21), String(-Infinity))`, "-1.5 -1e+21 -Infinity\n")},
		// 9.8.1 step 4: "If m is infinity, return the string "Infinity"."
		{"9.8.1 infinity converts to Infinity", output(`print(String(Infinity) + "!")`, "Infinity!\n")},
		// 9.8.1 step 6: "If k <= n <= 21, return ... the k digits ... followed by n-k occurences of the character '0'."
		{"9.8.1 an integer of up to 21 digits is written out", output(`print(String(123), String(1500), String(1e20), String(123456789012345680000))`, "123 1500 100000000000000000000 123456789012345680000\n")},
		// 9.8.1 step 7: "If 0 < n <= 21, return ... the most significant n digits ..., followed by a decimal point '.', followed by the remaining k-n digits"
		{"9.8.1 a number at least 1 with a fraction is written with a point", output(`print(String(1.5), String(123.456), String(100.25))`, "1.5 123.456 100.25\n")},
		// 9.8.1 step 8: "If -6 < n <= 0, return ... '0', followed by a decimal point '.', followed by -n occurences of the character '0', followed by the k digits"
		{"9.8.1 a number down to 1e-6 is written with leading zeros", output(`print(String(0.5), String(0.001), String(0.000001), String(0.0000015))`, "0.5 0.001 0.000001 0.0000015\n")},
		// 9.8.1 step 9: "if k = 1, return ... the single digit of s, followed by ... 'e', followed by a plus sign '+' or minus sign ... followed by ... abs(n-1)"
		{"9.8.1 a one-digit number outside that range uses an exponent", output(`print(String(1e21), String(1e-7), String(5e300), String(2e-300))`, "1e+21 1e-7 5e+300 2e-300\n")},
		// 9.8.1 step 10: "the most significant digit ..., followed by a decimal point '.', followed by the remaining k-1 digits ..., followed by ... 'e' ..."
		{"9.8.1 a several-digit number outside that range uses a point and an exponent", output(`print(String(1.5e21), String(1.2345e-7), String(-6.02e23))`, "1.5e+21 1.2345e-7 -6.02e+23\n")},
		// 9.8.1 step 5: "k is as small as possible", pinned by the README as the shortest digit string that round-trips
		{"9.8.1 the shortest digit string that round-trips is used", output(`print(0.1 + 0.2, 1 / 3)`, "0.30000000000000004 0.3333333333333333\n")},
		// 9.8.1: "if x is any number value other than -0, then ToNumber(ToString(x)) is exactly the same number value as x"
		{"9.8.1 converting to a string and back gives the same number",
			output(`var a = 0.1 + 0.2, b = 1 / 3, c = 5e-324, d = 1.7976931348623157e308; print(Number(String(a)) == a, Number(String(b)) == b, Number(String(c)) == c, Number(String(d)) == d)`, "true true true true\n")},
	})
}

func TestS9_9_ToObject(t *testing.T) {
	runRows(t, []row{
		// 9.9 Undefined: "Generate a runtime error." Property access calls ToObject (§11.2.1).
		{"9.9 undefined cannot be converted to an object", runtimeError(`var u; print("before"); u.x`, "before\n")},
		// 9.9 Null: "Generate a runtime error."
		{"9.9 null cannot be converted to an object", runtimeError(`var n = null; print("before"); n.x`, "before\n")},
		// 9.9 Boolean: "Create a new Boolean object whose default value is the value of the boolean."
		{"9.9 a boolean converts to a Boolean object", output(`print(true.toString(), false.valueOf())`, "true false\n")},
		// 9.9 Number: "Create a new Number object whose default value is the value of the number."
		{"9.9 a number converts to a Number object", output(`var n = 255; print(n.toString(16), n.valueOf())`, "ff 255\n")},
		// 9.9 String: "Create a new String object whose default value is the value of the string."
		{"9.9 a string converts to a String object", output(`print("abc".length, "abc".charAt(1))`, "3 b\n")},
		// 9.9: "Create a new ... object": each conversion makes a new object, so a property put on it is lost
		{"9.9 each conversion of a primitive makes a new object", output(`var s = "abc"; s.extra = 1; print(s.extra)`, "undefined\n")},
		// 9.9 Object: "The result is the input argument (no conversion)."
		{"9.9 an object converts to itself", output(`var o = new Object(); o.x = "same"; with (o) x = "changed"; print(o.x)`, "changed\n")},
	})
}
