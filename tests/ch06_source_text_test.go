package tests

import "testing"

// Chapter 6, Source Text. Only ASCII appears in the sources below: the README
// does not pin the encoding of source files or of stdout, so non-ASCII
// characters are written as \u escapes and observed through charCodeAt.

func TestS6_SourceText(t *testing.T) {
	runRows(t, []row{
		// 6: "In string literals, any Unicode character may also be expressed as a Unicode escape sequence"
		{"6 a Unicode escape in a string literal stands for one character",
			output(`var s = "\u00e9\u4e2d"; print(s.length, s.charCodeAt(0), s.charCodeAt(1))`, "2 233 20013\n")},
		// 6: "Within a comment, such an escape sequence is effectively ignored as part of the comment."
		{"6 a Unicode escape for a line feed does not end a single-line comment",
			output("print(1) // \\u000A print(2)\nprint(3)", "1\n3\n")},
		// 6: "a Unicode escape sequence occurring within a comment is never interpreted and therefore cannot contribute to termination of the comment"
		{"6 a Unicode escape for */ does not end a multi-line comment",
			output(`print(1); /* \u002A/ print(2); */ print(3)`, "1\n3\n")},
		// 6: "always contributes a character ... never interpreted as a line terminator or as a quote mark"
		{"6 a Unicode escape for a quote does not end a string literal",
			output(`print("a\u0022b", 'c\u0027d')`, "a\"b c'd\n")},
		// 6: "never interpreted as a line terminator" inside a string literal
		{"6 a Unicode escape for a line feed is allowed in a string literal",
			output(`print("a\u000Ab".length)`, "3\n")},
		// 6: "Non-ASCII Unicode characters may appear only within comments and string literals."
		// The comment holds UTF-8 bytes, which are never line terminators.
		{"6 a comment may hold non-ASCII characters",
			output("print(1) // caf\u00e9 \u4e2d\nprint(2) /* \u00e9 */", "1\n2\n")},
	})
}
