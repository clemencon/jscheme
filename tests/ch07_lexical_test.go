package tests

import "testing"

// Chapter 7, Lexical Conventions. Sources that need a control character
// (TAB, VT, FF, CR, LF) inside the program text use interpreted Go strings;
// a JS escape such as \n inside a JS string literal stays in a raw string.

func TestS7_LexicalConventions(t *testing.T) {
	runRows(t, []row{
		// 7: "repeatedly taking the longest possible sequence of characters as the next token"
		{"7 a+++b is read as a ++ + b", output(`var a = 1, b = 10; print(a+++b, a, b)`, "11 2 10\n")},
		// 7: "taking the longest possible sequence of characters as the next token"
		{"7 a---b is read as a -- - b", output(`var a = 5, b = 1; print(a---b, a, b)`, "4 4 1\n")},
		// 7: "the longest possible sequence": >>>= is one token, not >> followed by >=
		{"7 >>>= is read as one token", output(`var a = -8; a >>>= 28; print(a)`, "15\n")},
		// 7: "the longest possible sequence": a ++ a has no parse even though a + +a would
		{"7 tokens are not split to make a program parse", syntaxError(`var a = 1; print(a ++ a)`)},
	})
}

func TestS7_1_WhiteSpace(t *testing.T) {
	runRows(t, []row{
		// 7.1: "White space characters are used ... to separate tokens": <TAB>, <VT>, <FF>, <SP>
		{"7.1 tab, vertical tab, form feed and space separate tokens",
			output("var\ta\v=\f1 ;print(\ta\v+\f1 )", "2\n")},
		// 7.1: "White space may occur between any two tokens"
		{"7.1 white space separates a + + a into two plus tokens",
			output(`var a = 1; print(a + + a)`, "2\n")},
		// 7.1: "may occur within strings (where they are considered significant characters ...)"
		{"7.1 white space inside a string literal is part of its value",
			output("print(\"a\t\v\f b\".length)", "6\n")},
		// 7.1: "but cannot appear within any other kind of token"
		{"7.1 white space inside a numeric literal splits it into two tokens",
			syntaxError(`print(12 34)`)},
		// 7.1: "but cannot appear within any other kind of token"
		{"7.1 white space inside an identifier splits it into two tokens",
			syntaxError(`var ab = 1; print(a b)`)},
		// 7.1: "but cannot appear within any other kind of token"
		{"7.1 white space inside a punctuator splits it into two tokens",
			syntaxError(`var a = 1; print(a = = 1)`)},
	})
}

func TestS7_2_LineTerminators(t *testing.T) {
	runRows(t, []row{
		// 7.2: <LF> is a LineTerminator, so it separates tokens (and ends a statement here, §7.8.1)
		{"7.2 a line feed is a line terminator",
			output("var a = 1\nprint(a)", "1\n")},
		// 7.2: <CR> is a LineTerminator
		{"7.2 a carriage return is a line terminator",
			output("var a = 1\rprint(a)", "1\n")},
		// 7.2: <CR> <LF> is two line terminators, which are insignificant between tokens
		{"7.2 a CR LF pair separates statements",
			output("var a = 1\r\nprint(a)\r\n", "1\n")},
		// 7.2: "line terminators may occur between any two tokens"
		{"7.2 line terminators may separate the tokens of an expression",
			output("print(\n1\n+\r2\n)", "3\n")},
		// 7.2: "A line terminator cannot occur within any token, not even a string."
		{"7.2 a line feed inside a string literal is a syntax error",
			syntaxError("print(\"a\nb\")")},
		// 7.2: "A line terminator cannot occur within any token, not even a string."
		{"7.2 a carriage return inside a string literal is a syntax error",
			syntaxError("print('a\rb')")},
	})
}

func TestS7_3_Comments(t *testing.T) {
	runRows(t, []row{
		// 7.3: SingleLineComment :: // SingleLineCommentChars(opt)
		{"7.3 a single-line comment runs to the end of the line",
			output("print(1) // print(2)\nprint(3)", "1\n3\n")},
		// 7.3: MultiLineComment :: /* MultiLineCommentChars(opt) */
		{"7.3 a multi-line comment is ignored",
			output(`print(1 /* + 100 */ + 2)`, "3\n")},
		// 7.3: MultiLineComment may contain line terminators ("single or multi-line")
		{"7.3 a multi-line comment may span several lines",
			output("print(1);/* print(2);\nprint(3);\n*/print(4)", "1\n4\n")},
		// 7.3: "Multi-line comments cannot nest."
		{"7.3 a multi-line comment ends at the first */",
			output(`/* outer /* inner */ print(1) /* */`, "1\n")},
		// 7.3: "Multi-line comments cannot nest." The second */ is left over.
		{"7.3 a nested comment leaves its last */ outside any comment",
			syntaxError(`/* outer /* inner */ */ print(1)`)},
		// 7.3: MultiLineCommentChars may contain * and / that do not form */
		{"7.3 asterisks and slashes inside a multi-line comment do not end it",
			output(`print(1 /* a * / b ** c / d **/ + 2)`, "3\n")},
		// 7.3: MultiLineCommentChars is optional
		{"7.3 the empty multi-line comment /**/ is a comment",
			output(`print(1/**/+/***/2)`, "3\n")},
		// 7.3: MultiLineComment must end with */
		{"7.3 an unterminated multi-line comment is a syntax error",
			syntaxError(`print(1) /* never closed`)},
		// 7.3: "the LineTerminator at the end of the line is not considered to be part of the single-line comment"
		{"7.3 the line terminator after a single-line comment still ends the statement",
			output("var a = 1 // comment\nprint(a)", "1\n")},
		// 7.3: a single-line comment may contain any character but a LineTerminator, including */ and /*
		{"7.3 a single-line comment may contain comment delimiters",
			output("print(1) // /* */ print(2)\nprint(3)", "1\n3\n")},
		// 7.3: a single-line comment is ended by <CR> as well as <LF>
		{"7.3 a carriage return ends a single-line comment",
			output("print(1) // comment\rprint(2)", "1\n2\n")},
		// 7.3 and 7.6: ES1 has no HTML-like comments, so <!-- is the tokens <, !, --
		{"7.3 <!-- is not a comment opener but the operators < ! --",
			output(`var a = 1, b = 2; print(a <!--b, b)`, "false 1\n")},
	})
}

func TestS7_4_Tokens(t *testing.T) {
	runRows(t, []row{
		// 7.4: Token :: ReservedWord | Identifier | Punctuator | Literal
		{"7.4 a program is made of reserved words, identifiers, punctuators and literals",
			output(`var x = "s"; if (x) print(x, 1, true, null)`, "s 1 true null\n")},

		// 7.4.1: "Reserved words cannot be used as identifiers." ReservedWord includes NullLiteral.
		{"7.4.1 null cannot be declared as a variable", syntaxError(`var null = 1;`)},
		// 7.4.1: ReservedWord includes BooleanLiteral
		{"7.4.1 true cannot be declared as a variable", syntaxError(`var true = 1;`)},
		// 7.4.1: ReservedWord includes BooleanLiteral
		{"7.4.1 false cannot be declared as a variable", syntaxError(`var false = 1;`)},
		// 7.4.1: "Reserved words cannot be used as identifiers." A function name is an Identifier (§13).
		{"7.4.1 a reserved word cannot name a function", syntaxError(`function true() {}`)},
		// 7.4.1: "Reserved words cannot be used as identifiers." A parameter is an Identifier (§13).
		{"7.4.1 a reserved word cannot name a parameter", syntaxError(`function f(null) {}`)},
		// 7.4.1: a word is reserved only as a whole token; a longer IdentifierName is an Identifier
		{"7.4.1 an identifier may start with a reserved word",
			output(`var nullable = 1, trueish = 2, iffy = 3, newer = 4; print(nullable, trueish, iffy, newer)`, "1 2 3 4\n")},

		// 7.4.2: "may not be used as identifiers": break
		{"7.4.2 break is a keyword", syntaxError(`var break = 1;`)},
		// 7.4.2: continue
		{"7.4.2 continue is a keyword", syntaxError(`var continue = 1;`)},
		// 7.4.2: delete
		{"7.4.2 delete is a keyword", syntaxError(`var delete = 1;`)},
		// 7.4.2: else
		{"7.4.2 else is a keyword", syntaxError(`var else = 1;`)},
		// 7.4.2: for
		{"7.4.2 for is a keyword", syntaxError(`var for = 1;`)},
		// 7.4.2: function
		{"7.4.2 function is a keyword", syntaxError(`var function = 1;`)},
		// 7.4.2: if
		{"7.4.2 if is a keyword", syntaxError(`var if = 1;`)},
		// 7.4.2: in
		{"7.4.2 in is a keyword", syntaxError(`var in = 1;`)},
		// 7.4.2: new
		{"7.4.2 new is a keyword", syntaxError(`var new = 1;`)},
		// 7.4.2: return
		{"7.4.2 return is a keyword", syntaxError(`var return = 1;`)},
		// 7.4.2: this
		{"7.4.2 this is a keyword", syntaxError(`var this = 1;`)},
		// 7.4.2: typeof
		{"7.4.2 typeof is a keyword", syntaxError(`var typeof = 1;`)},
		// 7.4.2: var
		{"7.4.2 var is a keyword", syntaxError(`var var = 1;`)},
		// 7.4.2: void
		{"7.4.2 void is a keyword", syntaxError(`var void = 1;`)},
		// 7.4.2: while
		{"7.4.2 while is a keyword", syntaxError(`var while = 1;`)},
		// 7.4.2: with
		{"7.4.2 with is a keyword", syntaxError(`var with = 1;`)},
		// 7.4.2: keywords are case sensitive like identifiers (§7.5), so If and VAR are identifiers
		{"7.4.2 a keyword spelled with other letter case is an identifier",
			output(`var If = 1, VAR = 2, Function = 3; print(If, VAR, Function)`, "1 2 3\n")},
		// 7.4.2: the keywords are exactly those listed; instanceof was reserved only later
		{"7.4.2 instanceof is not a keyword",
			output(`var instanceof = 1; print(instanceof)`, "1\n")},
		// 7.4.2 and 7.4.3: words reserved by later editions only in strict mode are identifiers
		{"7.4.2 let, static, yield and implements are not reserved",
			output(`var let = 1, static = 2, yield = 3, implements = 4; print(let, static, yield, implements)`, "1 2 3 4\n")},

		// 7.4.3: FutureReservedWord. §2 lets an implementation give these words a meaning,
		// so their use is not asserted to fail. They are reserved only as whole tokens.
		{"7.4.3 an identifier may contain a future reserved word",
			output(`var classes = 1, doit = 2, trying = 3, constant = 4, imports = 5; print(classes, doit, trying, constant, imports)`, "1 2 3 4 5\n")},
	})
}

func TestS7_5_Identifiers(t *testing.T) {
	runRows(t, []row{
		// 7.5: "each character ... must be a letter, a decimal digit, an underscore (_) character, or a dollar sign ($)"
		{"7.5 identifiers are made of letters, digits, _ and $",
			output(`var a1 = 1, _ = 2, $ = 3, $_x9Z = 4; print(a1, _, $, $_x9Z)`, "1 2 3 4\n")},
		// 7.5: "the first character may not be a decimal digit"
		{"7.5 an identifier cannot start with a digit", syntaxError(`var 1a = 1;`)},
		// 7.5: "identifiers whose characters differ in any way, even if only in case, are considered to be distinct"
		{"7.5 identifiers are case sensitive",
			output(`var a = 1, A = 2; print(a, A)`, "1 2\n")},
		// 7.5: "an identifier is a character sequence of unlimited length"
		{"7.5 a long identifier is distinct from its prefix",
			output(`var abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghij1 = 1, abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghij2 = 2; print(abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghij1, abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghij2)`, "1 2\n")},
		// 7.5: a character that is not a letter, digit, _ or $ cannot be part of an identifier
		{"7.5 # is not an identifier character", syntaxError(`var a#b = 1;`)},
	})
}

func TestS7_6_Punctuators(t *testing.T) {
	runRows(t, []row{
		// 7.6: = == != < > <= >= ! ~ ? : , are punctuators
		{"7.6 the comparison and conditional punctuators are tokens",
			output(`var a = 1; print(a == 1, a != 1, a < 2, a > 2, a <= 1, a >= 2, !a, ~a, a ? "y" : "n")`,
				"true false true false true false false -2 y\n")},
		// 7.6: && || + - * / % & | ^ << >> >>> are punctuators
		{"7.6 the binary operator punctuators are tokens",
			output(`print(1 && 2, 0 || 3, 7 + 2, 7 - 2, 7 * 2, 7 / 2, 7 % 2, 6 & 3, 6 | 3, 6 ^ 3, 1 << 3, -16 >> 2, -1 >>> 28)`,
				"2 3 9 5 14 3.5 1 2 7 5 8 -4 15\n")},
		// 7.6: ++ -- and the compound assignments += -= *= /= %= &= |= ^= <<= >>= >>>= are punctuators
		{"7.6 the increment and assignment punctuators are tokens",
			output(`var a = 1; a++; ++a; a--; a += 10; a -= 1; a *= 4; a /= 2; a %= 7; a <<= 3; a >>= 1; a |= 1; a &= 13; a ^= 3; a >>>= 1; print(a)`,
				"3\n")},
		// 7.6: . ( ) { } [ ] ; are punctuators
		{"7.6 the bracketing and separator punctuators are tokens",
			output(`var s = "abc"; { print(s.length, s["length"], (s).charAt(1)); }`, "3 3 b\n")},
	})
}

func TestS7_7_Literals(t *testing.T) {
	runRows(t, []row{
		// 7.7: Literal :: NullLiteral | BooleanLiteral | NumericLiteral | StringLiteral
		{"7.7 each kind of literal has a value of its own type",
			output(`print(typeof null, typeof true, typeof 1, typeof "s")`, "object boolean number string\n")},

		// 7.7.1: "The value of the null literal null is the sole value of the Null type, namely null."
		{"7.7.1 the null literal is the value null", output(`print(null)`, "null\n")},

		// 7.7.2: "The value of the Boolean literal true is a value of the Boolean type, namely true."
		{"7.7.2 the literal true is the value true", output(`print(true, typeof true)`, "true boolean\n")},
		// 7.7.2: "The value of the Boolean literal false is a value of the Boolean type, namely false."
		{"7.7.2 the literal false is the value false", output(`print(false, typeof false)`, "false boolean\n")},

		// 7.7.3: "The MV of DecimalIntegerLiteral :: 0 is 0." and "If the MV is 0, then the rounded value is +0"
		{"7.7.3 the literal 0 is +0", output(`print(0, 1/0)`, "0 Infinity\n")},
		// 7.7.3: "The MV of DecimalIntegerLiteral :: NonZeroDigit DecimalDigits is (the MV of NonZeroDigit times 10^n) plus the MV of DecimalDigits"
		{"7.7.3 a decimal integer literal", output(`print(7, 42, 1234567890)`, "7 42 1234567890\n")},
		// 7.7.3: "The MV of DecimalLiteral :: DecimalIntegerLiteral . is the MV of DecimalIntegerLiteral."
		{"7.7.3 a decimal literal may end with a point", output(`print(5.)`, "5\n")},
		// 7.7.3: "DecimalIntegerLiteral . DecimalDigits is the MV of DecimalIntegerLiteral plus (the MV of DecimalDigits times 10^-n)"
		{"7.7.3 a decimal literal with a fraction", output(`print(3.25, 0.5, 0.0625)`, "3.25 0.5 0.0625\n")},
		// 7.7.3: "DecimalIntegerLiteral . ExponentPart is the MV of DecimalIntegerLiteral times 10^e"
		{"7.7.3 a point may be followed directly by an exponent", output(`print(2.e3)`, "2000\n")},
		// 7.7.3: "DecimalIntegerLiteral . DecimalDigits ExponentPart is (...) times 10^e"
		{"7.7.3 a fraction with an exponent", output(`print(1.5e3, 2.5E-2)`, "1500 0.025\n")},
		// 7.7.3: "DecimalLiteral :: . DecimalDigits is the MV of DecimalDigits times 10^-n"
		{"7.7.3 a decimal literal may start with a point", output(`print(.5, .125)`, "0.5 0.125\n")},
		// 7.7.3: "DecimalLiteral :: . DecimalDigits ExponentPart is the MV of DecimalDigits times 10^(e-n)"
		{"7.7.3 a leading point with an exponent", output(`print(.5e1, .25e2)`, "5 25\n")},
		// 7.7.3: "DecimalIntegerLiteral ExponentPart is the MV of DecimalIntegerLiteral times 10^e"
		{"7.7.3 an integer with an exponent", output(`print(1e3, 1E3, 5e0)`, "1000 1000 5\n")},
		// 7.7.3: "SignedInteger :: + DecimalDigits is the MV of DecimalDigits" and "- DecimalDigits is the negative"
		{"7.7.3 an exponent may carry a sign", output(`print(1e+2, 1e-2, 4E-1)`, "100 0.01 0.4\n")},
		// 7.7.3: an ExponentPart needs digits after the indicator
		{"7.7.3 an exponent without digits is a syntax error", syntaxError(`print(1e)`)},
		// 7.7.3: "The MV of HexIntegerLiteral :: 0x HexDigit" ... "(the MV of HexIntegerLiteral times 16) plus the MV of HexDigit"
		{"7.7.3 a hexadecimal literal with 0x", output(`print(0x0, 0x1F, 0x100)`, "0 31 256\n")},
		// 7.7.3: "The MV of HexIntegerLiteral :: 0X HexDigit is the MV of HexDigit."
		{"7.7.3 a hexadecimal literal with 0X", output(`print(0XA)`, "10\n")},
		// 7.7.3: "The MV of HexDigit :: a or of HexDigit :: A is 10" ... "f or F is 15"
		{"7.7.3 hexadecimal digits are case insensitive", output(`print(0xabcdef, 0xABCDEF, 0xaBcDeF)`, "11259375 11259375 11259375\n")},
		// 7.7.3: HexIntegerLiteral :: 0x HexDigit needs at least one digit
		{"7.7.3 0x without a digit is a syntax error", syntaxError(`print(0x)`)},
		// 7.7.3: "The MV of OctalIntegerLiteral :: 0 OctalDigit" ... "(the MV of OctalIntegerLiteral times 8) plus the MV of OctalDigit"
		{"7.7.3 a literal with a leading 0 is octal", output(`print(07, 010, 0777, 00)`, "7 8 511 0\n")},
		// 7.7.3: "the rounded value must be the number value for the MV (in the sense defined in section 8.4)"
		{"7.7.3 a literal is rounded to the nearest number value", output(`print(9007199254740993, 0.1)`, "9007199254740992 0.1\n")},
		// 7.7.3: rounding to "the number value for the MV": a very large MV rounds to +Infinity (§8.5)
		{"7.7.3 a literal too large for a number is Infinity", output(`print(1e400, -1e400)`, "Infinity -Infinity\n")},
		// 7.7.3: "If the MV is 0, then the rounded value is +0" also when the literal is written with an exponent or a fraction
		{"7.7.3 any literal of MV 0 is +0", output(`print(1/0.0, 1/0e5, 1/0x0, 1/.0)`, "Infinity Infinity Infinity Infinity\n")},
		// 7.7.3: "unless the literal ... has more than 20 significant digits, in which case the number value may be either ..."
		{"7.7.3 digits after the 20th significant digit may be ignored",
			output(`print(1.000000000000000000000001, 12345678901234567890123)`, "1 1.2345678901234568e+22\n")},
	})
}

func TestS7_7_4_StringLiterals(t *testing.T) {
	runRows(t, []row{
		// 7.7.4: "The SV of StringLiteral :: "" is the empty character sequence."
		{"7.7.4 an empty double-quoted string is empty", output(`print("".length)`, "0\n")},
		// 7.7.4: "The SV of StringLiteral :: '' is the empty character sequence."
		{"7.7.4 an empty single-quoted string is empty", output(`print(''.length)`, "0\n")},
		// 7.7.4: "a string literal is zero or more characters enclosed in single or double quotes"
		{"7.7.4 double and single quotes delimit strings alike", output(`print("abc", 'abc', "abc" == 'abc')`, "abc abc true\n")},
		// 7.7.4: DoubleStringCharacter is any SourceCharacter but not " or \ or LineTerminator
		{"7.7.4 a single quote needs no escape inside double quotes", output(`print("it's")`, "it's\n")},
		// 7.7.4: SingleStringCharacter is any SourceCharacter but not ' or \ or LineTerminator
		{"7.7.4 a double quote needs no escape inside single quotes", output(`print('say "hi"')`, "say \"hi\"\n")},
		// 7.7.4: a string literal must be closed by its quote on the same line
		{"7.7.4 an unterminated string literal is a syntax error", syntaxError(`print("abc)`)},
		// 7.7.4: a string literal is closed only by the quote that opened it
		{"7.7.4 a string is not closed by the other kind of quote", syntaxError(`print("abc')`)},
		// 7.7.4 table: \b is \u0008, \t is \u0009, \n is \u000A, \f is \u000C, \r is \u000D
		{"7.7.4 the control character escapes",
			output(`var s = "\b\t\n\f\r"; print(s.length, s.charCodeAt(0), s.charCodeAt(1), s.charCodeAt(2), s.charCodeAt(3), s.charCodeAt(4))`,
				"5 8 9 10 12 13\n")},
		// 7.7.4 table: \" is ", \' is ', \\ is \
		{"7.7.4 the quote and backslash escapes", output(`print("\"", '\'', "\'", '\"', "\\")`, "\" ' ' \" \\\n")},
		// 7.7.4: "The CV of CharacterEscapeSequence :: \ NonEscapeCharacter is the CV of the NonEscapeCharacter."
		{"7.7.4 a backslash before an ordinary character is dropped", output(`print("\q\a\c\z\ \$\9".length, "\q\a\c\z\$")`, "7 qacz$\n")},
		// 7.7.4: "The CV of HexEscapeSequence :: \x HexDigit HexDigit is ... (16 times the MV of the first HexDigit) plus the MV of the second"
		{"7.7.4 a hexadecimal escape", output(`print("\x41\x62", "\xff".charCodeAt(0), "\xFF".charCodeAt(0))`, "Ab 255 255\n")},
		// 7.7.4: \x is an EscapeCharacter, so it is not a NonEscapeCharacter and needs two hex digits
		{"7.7.4 a hexadecimal escape with a non-hex digit is a syntax error", syntaxError(`print("\xg1")`)},
		// 7.7.4: "The CV of OctalEscapeSequence :: \ OctalDigit is the Unicode character whose code is the MV of the OctalDigit."
		{"7.7.4 a one-digit octal escape", output(`print("\0".charCodeAt(0), "\7".charCodeAt(0), "\0".length)`, "0 7 1\n")},
		// 7.7.4: "OctalEscapeSequence :: \ OctalDigit OctalDigit is ... (8 times the MV of the first OctalDigit) plus the MV of the second"
		{"7.7.4 a two-digit octal escape", output(`print("\101", "\77".charCodeAt(0))`, "A 63\n")},
		// 7.7.4: "\ ZeroToThree OctalDigit OctalDigit is ... (64 times the MV of the ZeroToThree) plus (8 times ...) plus ..."
		{"7.7.4 a three-digit octal escape starts with 0 to 3", output(`print("\101\102", "\377".charCodeAt(0))`, "AB 255\n")},
		// 7.7.4: OctalEscapeSequence has three digits only after ZeroToThree, so \400 is \40 then "0"
		{"7.7.4 an octal escape starting with 4 to 7 takes at most two digits", output(`print("\400".length, "\400".charCodeAt(0), "\400".charAt(1))`, "2 32 0\n")},
		// 7.7.4: the longest OctalEscapeSequence is three digits, so a fourth digit is a character of its own
		{"7.7.4 an octal escape takes at most three digits", output(`print("\1011")`, "A1\n")},
		// 7.7.4: "The CV of UnicodeEscapeSequence :: \u HexDigit HexDigit HexDigit HexDigit is the Unicode character whose code is ..."
		{"7.7.4 a Unicode escape", output(`print("\u0041\u00620", "\uABCD".charCodeAt(0), "\uabcd".charCodeAt(0))`, "Ab0 43981 43981\n")},
		// 7.7.4: \u is an EscapeCharacter, so it needs four hex digits
		{"7.7.4 a Unicode escape with too few hex digits is a syntax error", syntaxError(`print("\u004")`)},
		// 7.7.4: "a LineTerminator character cannot appear in a string literal" "The correct way ... is to use an escape sequence such as \n or \u000A"
		{"7.7.4 a line feed enters a string through an escape", output(`print("a\nb", "a\u000Ab" == "a\nb")`, "a\nb true\n")},
	})
}

func TestS7_8_AutomaticSemicolonInsertion(t *testing.T) {
	runRows(t, []row{
		// 7.8: "Such a semicolon may always appear explicitly in the source text."
		{"7.8 statements may end with explicit semicolons", output(`var a = 1; a = a + 1; print(a);;`, "2\n")},

		// 7.8.1 rule 1.1: "The offending token is separated from the previous token by at least one LineTerminator."
		{"7.8.1 a semicolon is inserted before a token on a new line that cannot continue the statement",
			output("var a = 1\nvar b = 2\nprint(a, b)", "1 2\n")},
		// 7.8.1 rule 1.2: "The offending token is }."
		{"7.8.1 a semicolon is inserted before }",
			output(`function f() { return 1 } { print(f()) }`, "1\n")},
		// 7.8.1: an offending token on the same line that is not } gets no semicolon
		{"7.8.1 no semicolon is inserted between tokens on one line", syntaxError(`var a = 1 var b = 2`)},
		// 7.8.1 rule 2: "the end of the input stream of tokens is encountered ... a semicolon is automatically inserted at the end"
		{"7.8.1 a semicolon is inserted at the end of the input", output(`var a = 3; print(a)`, "3\n")},
		// 7.8.1: "a semicolon is never inserted automatically if the semicolon would then be parsed as an empty statement"
		{"7.8.1 no semicolon is inserted to make an empty statement", syntaxError("if (true)\nprint(1)\nelse\n")},
		// 7.8.1: "the parser is not currently parsing the header of a for statement"
		{"7.8.1 no semicolon is inserted in a for header", syntaxError("for (var i = 0\ni < 3; i++) print(i)")},
		// 7.8.1 restricted production: PostfixExpression :: LeftHandSideExpression [no LineTerminator here] ++
		{"7.8.1 a ++ on a new line is not postfix", output("var a = 1, b = 1\na\n++b\nprint(a, b)", "1 2\n")},
		// 7.8.1 restricted production: PostfixExpression :: LeftHandSideExpression [no LineTerminator here] --
		{"7.8.1 a -- on a new line is not postfix", output("var a = 1, b = 1\na\n--b\nprint(a, b)", "1 0\n")},
		// 7.8.1 restricted production: ReturnStatement :: return [no LineTerminator here] Expression(opt) ;
		{"7.8.1 a line terminator after return ends the return statement",
			output("function f() {\nreturn\n42\n}\nprint(f())", "undefined\n")},
		// 7.8.1: the restricted token is separated by a LineTerminator in a for header: "it is a syntax error"
		{"7.8.1 a ++ on a new line in a for header is a syntax error",
			syntaxError("for (var i = 0; i < 1; i\n++) print(i)")},
		// 7.8.1: a LineTerminator allowed by the grammar inserts nothing, so a statement continues on the next line
		{"7.8.1 a statement continues on the next line when the grammar allows it",
			output("var a = 1 +\n2\nprint(a)", "3\n")},
		// 7.8.1: "the presence or absence of single-line comments does not affect the process of automatic semicolon insertion" (§7.3)
		{"7.8.1 a single-line comment before the line terminator still lets a semicolon be inserted",
			output("var a = 1 // one\nvar b = 2 // two\nprint(a + b)", "3\n")},

		// 7.8.2: "{ 1 2 } 3 is not a valid sentence in the ECMAScript grammar, even with the automatic semicolon insertion rules"
		{"7.8.2 { 1 2 } 3 is a syntax error", syntaxError(`{ 1 2 } 3`)},
		// 7.8.2: "{ 1\n2 } 3 ... is transformed by automatic semicolon insertion into ... { 1\n;2 ;} 3;"
		{"7.8.2 { 1 newline 2 } 3 is a valid program", output(`print(eval("{ 1\n2 } 3"))`, "3\n")},
		// 7.8.2: "for (a; b\n) ... is not altered by automatic semicolon insertion"
		{"7.8.2 for (a; b newline ) is a syntax error", syntaxError("var a, b\nfor (a; b\n)")},
		// 7.8.2: "return\na + b is transformed ... into return;\na + b;"
		{"7.8.2 return newline a + b returns undefined",
			output("var a = 1, b = 2\nfunction f() {\nreturn\na + b\n}\nprint(f())", "undefined\n")},
		// 7.8.2: "a = b\n++c is transformed ... into a = b;\n++c;"
		{"7.8.2 a = b newline ++c increments c, not b",
			output("var a, b = 1, c = 1\na = b\n++c\nprint(a, b, c)", "1 1 2\n")},
		// 7.8.2: "if (a > b)\nelse c = d is not a valid ECMAScript sentence"
		{"7.8.2 if (a > b) newline else c = d is a syntax error", syntaxError("var a, b, c, d\nif (a > b)\nelse c = d")},
		// 7.8.2: "a = b + c\n(d + e).print() is not transformed ... a = b + c(d + e).print()"
		{"7.8.2 a parenthesis on the next line continues the expression as a call",
			output("function P(v) { this.v = v; this.print = show }\n"+
				"function show() { print(\"show\", this.v); return \"!\" }\n"+
				"function c(x) { return new P(x) }\n"+
				"var a, b = \"b\", d = 1, e = 2\n"+
				"a = b + c\n(d + e).print()\nprint(a)", "show 3\nb!\n")},
	})
}
