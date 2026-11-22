package tests

import "testing"

// Chapter 16, Errors. §16 fixes the last moment an error may be reported
// and lets an implementation report it sooner. Rows that print before the
// error use constructs whose failure depends on run-time values, so no
// implementation can report them early; constructs that fail whatever the
// values use earlyOrRuntimeError.

func TestS16_Errors(t *testing.T) {
	runRows(t, []row{
		// 16: "This specification specifies the last possible moment an error occurs."
		{"16 a runtime error stops the program where it occurs",
			runtimeError(`function get(o) { return o.x } print("before"); get(null); print("after")`, "before\n")},
		// 16: the last possible moment for an error in a function body is when the function runs
		{"16 an error in a function body is reported when the function is called",
			runtimeError(`function f(o) { return o.x } print("declared"); var ok = f(new Object()); print("called once"); f(void 0); print("after")`, "declared\ncalled once\n")},
		// 16: an error that depends on run-time values is not reported for code that does not run
		{"16 code that never runs reports no runtime error",
			output(`function g() { return undeclaredName } if (false) { undeclaredName } print("ok")`, "ok\n")},
		// 16: "A given implementation may generate errors sooner (e.g. at compile-time)."
		{"16 an error provable from the source may be reported before the program runs", earlyOrRuntimeError(`5()`)},
		// 16: "if a compiler can prove that a construct cannot execute without error under any circumstances, then it may issue a compile-time error"
		{"16 a construct that always fails may be reported before it runs", earlyOrRuntimeError(`var x = new 1`)},
		// 16: "An ECMAScript compiler should detect errors at compile time in all code presented to it, even code that detailed analysis might prove to be dead"
		{"16 a syntax error in code that never runs stops the program before it starts",
			syntaxError(`print("before"); if (false) { x = ( }`)},
		// 16: code given to eval is presented to the compiler only when eval runs
		{"16 a syntax error in an eval string is reported when eval runs",
			runtimeError(`var src = "1 +"; print("before"); eval(src); print("after")`, "before\n")},
	})
}
