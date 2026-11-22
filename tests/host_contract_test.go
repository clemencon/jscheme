package tests

import "testing"

// Tests of the host contract (README.md, "Host contract"): how jscheme is
// invoked, its exit codes, the print function and the REPL. Row names start
// with "host:" and the README subsection instead of a spec section number.

func TestHostInvocation(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name string
		act  func(testing.TB)
	}{
		// "jscheme -e <source>: Evaluate <source> as an ECMAScript Program."
		{"host:Invocation -e evaluates its argument as a Program", func(t testing.TB) {
			expectOutput(t, `print("a"); print("b")`, "a\nb\n")
		}},
		// "jscheme <file.js>: Read the file and evaluate it exactly like -e with its contents."
		{"host:Invocation a file is read and evaluated as a Program", func(t testing.TB) {
			runFile(t, "hello.js", outcome{exit: 0, stdout: "hello world\n"})
		}},
		// "evaluate it exactly like -e": a syntax error is found before anything runs (exit code 2).
		{"host:Invocation a file with a syntax error runs none of it", func(t testing.TB) {
			runFile(t, "syntax-error.js", outcome{exit: 2, stdout: "", stderrNotEmpty: true})
		}},
		// "evaluate it exactly like -e": a runtime error ends the program (exit code 1).
		{"host:Invocation a file stops at its first runtime error", func(t testing.TB) {
			runFile(t, "runtime-error.js", outcome{exit: 1, stdout: "before\n", stderrNotEmpty: true})
		}},
		// "jscheme: Start the REPL on stdin."
		{"host:Invocation with no arguments the REPL reads stdin", func(t testing.TB) {
			runRepl(t, "print(\"from stdin\")\n", outcome{exit: 0, stdout: "from stdin\n"})
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.act(t)
		})
	}
}

func TestHostExitCodes(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name string
		act  func(testing.TB)
	}{
		// "0: Program executed successfully."
		{"host:Exit codes 0 when the program completes", func(t testing.TB) {
			expectOutput(t, `print("done")`, "done\n")
		}},
		// "1: Runtime error (spec §16)." Reading an undeclared variable fails in GetValue (§8.7.3).
		{"host:Exit codes 1 for a runtime error, after what ran before it", func(t testing.TB) {
			expectRuntimeError(t, `print("before"); undeclared; print("after")`, "before\n")
		}},
		// "2: Syntax error, detected before execution (spec §16)."
		{"host:Exit codes 2 for a syntax error, before anything runs", func(t testing.TB) {
			expectSyntaxError(t, `print("never"); )`)
		}},
		// "For such errors the tests accept code 1 or 2." (e.g. `1 = 2;`)
		{"host:Exit codes 1 or 2 for an error provable before execution", func(t testing.TB) {
			expectEarlyOrRuntimeError(t, `1 = 2;`)
		}},
		// "3: Usage or I/O error: bad arguments, ..."
		{"host:Exit codes 3 for an unknown option", func(t testing.TB) {
			expectUsageError(t, "--no-such-option")
		}},
		// "3: Usage or I/O error: bad arguments, ..." (`-e` needs a <source>)
		{"host:Exit codes 3 for -e without a source", func(t testing.TB) {
			expectUsageError(t, "-e")
		}},
		// "3: Usage or I/O error: ... missing ... file."
		{"host:Exit codes 3 for a missing file", func(t testing.TB) {
			expectUsageError(t, "testdata/no-such-file.js")
		}},
		// "3: Usage or I/O error: ... unreadable file." A directory cannot be read as a file.
		{"host:Exit codes 3 for a path that cannot be read as a file", func(t testing.TB) {
			expectUsageError(t, "testdata")
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.act(t)
		})
	}
}

func TestHostPrint(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name string
		src  string
		want string
	}{
		// "converts each argument with ToString"
		{"host:The print function converts each argument with ToString",
			`print(1.5, "s", true, null, void 0)`, "1.5 s true null undefined\n"},
		// "converts each argument with ToString": for an object, ToPrimitive with hint String calls toString (§9.8).
		{"host:The print function converts an object through its toString method",
			`function f() { return "custom"; } var o = new Object(); o.toString = f; print(o)`, "custom\n"},
		// "joins them with a single space"
		{"host:The print function joins its arguments with a single space",
			`print("a", "", "b")`, "a  b\n"},
		// "writes the result plus \"\n\" to stdout"
		{"host:The print function ends every call with a newline",
			`print("a"); print(); print("b")`, "a\n\nb\n"},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			expectOutput(t, r.src, r.want)
		})
	}
}

func TestHostREPL(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name  string
		input string
		want  outcome
	}{
		// "All inputs are evaluated in one shared global environment."
		{"host:REPL a variable declared in one input is visible in the next",
			"var x = 40;\nprint(x + 2);\n", outcome{exit: 0, stdout: "42\n"}},
		// "All inputs are evaluated in one shared global environment."
		{"host:REPL a function declared in one input can be called in the next",
			"function f() { return 7; }\nprint(f());\n", outcome{exit: 0, stdout: "7\n"}},
		// "An input may span multiple lines; evaluation happens once a complete statement has been read."
		{"host:REPL an expression may span several lines",
			"print(1 +\n2);\n", outcome{exit: 0, stdout: "3\n"}},
		// "An input may span multiple lines; evaluation happens once a complete statement has been read."
		{"host:REPL a function declaration may span several lines",
			"function f() {\n  return 5;\n}\nprint(f());\n", outcome{exit: 0, stdout: "5\n"}},
		// "the completion value is printed with ToString, unless it is undefined"
		{"host:REPL the completion value is echoed with ToString",
			"1 + 2\n\"a\" + \"b\"\n", outcome{exit: 0, stdout: "3\nab\n"}},
		// "the completion value is printed with ToString, unless it is undefined"
		{"host:REPL an undefined completion value is not echoed",
			"void 0\nprint(\"after\")\n", outcome{exit: 0, stdout: "after\n"}},
		// "Errors are written to stderr and the session continues."
		{"host:REPL a runtime error is reported and the session continues",
			"print(1);\nundeclared;\nprint(2);\n", outcome{exit: 0, stdout: "1\n2\n", stderrNotEmpty: true}},
		// "Errors are written to stderr and the session continues."
		{"host:REPL a syntax error is reported and the session continues",
			"print(1);\n)\nprint(2);\n", outcome{exit: 0, stdout: "1\n2\n", stderrNotEmpty: true}},
		// "The prompt > is shown only when stdin is a TTY"
		{"host:REPL piped input sees no prompt",
			"print(\"a\");\n", outcome{exit: 0, stdout: "a\n"}},
		// "On EOF the REPL exits with code 0."
		{"host:REPL a last line without a newline is evaluated before exit 0 at EOF",
			"print(\"last\")", outcome{exit: 0, stdout: "last\n"}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			runRepl(t, r.input, r.want)
		})
	}
}
