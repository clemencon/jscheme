// Package tests is the jscheme conformance suite for ECMA-262, 1st edition.
//
// Every test runs the interpreter as a separate process and observes only
// what the host contract exposes: stdout, the exit code, and whether stderr
// is empty.
//
// This file holds the harness: how the binary is run, the assertion helpers
// and the rows.
package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// envBin names the binary under test. Default: ../bin/jscheme.
const envBin = "JSCHEME_BIN"

// timeout bounds every run of the binary. A hang fails the test.
const timeout = 10 * time.Second

// config is what the harness needs to run the binary under test.
type config struct {
	bin     string   // absolute path of the binary under test
	env     []string // the whole process environment of every run
	pathDir string   // the empty directory that makes up PATH; TestMain removes it
}

var (
	configOnce sync.Once
	cfg        config
	cfgErr     error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if cfg.pathDir != "" {
		os.RemoveAll(cfg.pathDir)
	}
	os.Exit(code)
}

// harness returns the configuration, reading it on first use. It is read
// from inside a test, never from TestMain, so that go test's result cache
// sees the environment variable and the binary it depends on.
func harness(tb testing.TB) config {
	tb.Helper()
	configOnce.Do(func() { cfg, cfgErr = readConfig() })
	if cfgErr != nil {
		tb.Fatalf("harness: %v", cfgErr)
	}
	return cfg
}

// readConfig resolves the binary under test and builds the process
// environment.
func readConfig() (config, error) {
	var c config
	path := os.Getenv(envBin)
	if path == "" {
		path = filepath.Join("..", "bin", "jscheme")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return c, err
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return c, fmt.Errorf("%s is not an executable file (set %s, or build bin/jscheme)", abs, envBin)
	}
	c.bin = abs

	// PATH holds a single empty directory, so no program can be found by
	// name: in particular no JavaScript runtime.
	c.pathDir, err = os.MkdirTemp("", "jscheme-path-")
	if err != nil {
		return c, err
	}
	c.env = []string{"TZ=UTC", "PATH=" + c.pathDir}
	return c, nil
}

// result is everything a test can observe about one run.
type result struct {
	exit   int
	stdout string
	stderr string // shown in failure messages only, never compared
}

func (r result) String() string {
	return fmt.Sprintf("exit code: %d\nstdout: %q\nstderr: %q", r.exit, r.stdout, r.stderr)
}

// run runs the binary once with args and the given stdin ("" for none).
// Failing to run it at all, or a timeout, fails the test outright.
func run(tb testing.TB, stdin string, args ...string) result {
	tb.Helper()
	c := harness(tb)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Env = c.env
	cmd.WaitDelay = time.Second
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := result{stdout: stdout.String(), stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() != nil:
		tb.Fatalf("%s did not finish within %v\nargs: %q", c.bin, timeout, args)
	case errors.As(err, &exitErr) && exitErr.Exited():
		res.exit = exitErr.ExitCode()
	case err != nil:
		tb.Fatalf("running %s: %v\nargs: %q", c.bin, err, args)
	}
	return res
}

// runSource runs src as a Program with `-e`.
func runSource(tb testing.TB, src string) result {
	tb.Helper()
	return run(tb, "", "-e", src)
}

// Assertion helpers. Each one is a whole Act and Assert: it runs the binary
// once and reports every mismatch with Errorf, never with Fatal, so that one run
// shows every way the outcome differs.

// outcome is what a row expects from one run.
type outcome struct {
	exit           int
	stdout         string
	stderrNotEmpty bool // true: stderr must not be empty; false: stderr is not checked
}

func checkOutcome(tb testing.TB, what string, got result, want outcome) {
	tb.Helper()
	if got.exit != want.exit || got.stdout != want.stdout || (want.stderrNotEmpty && got.stderr == "") {
		tb.Errorf("%s\nwant: exit code %d, stdout %q, stderr not empty: %v\ngot:\n%v",
			what, want.exit, want.stdout, want.stderrNotEmpty, got)
	}
}

// expectOutput asserts exit code 0 and that stdout is exactly want.
func expectOutput(tb testing.TB, src, want string) {
	tb.Helper()
	checkOutcome(tb, "source: "+src, runSource(tb, src), outcome{exit: 0, stdout: want})
}

// expectRuntimeError asserts exit code 1, that stdout is exactly what the
// program printed before the error, and that stderr is not empty.
func expectRuntimeError(tb testing.TB, src, stdoutBefore string) {
	tb.Helper()
	checkOutcome(tb, "source: "+src, runSource(tb, src), outcome{exit: 1, stdout: stdoutBefore, stderrNotEmpty: true})
}

// expectSyntaxError asserts exit code 2, an empty stdout (nothing ran) and a
// non-empty stderr.
func expectSyntaxError(tb testing.TB, src string) {
	tb.Helper()
	checkOutcome(tb, "source: "+src, runSource(tb, src), outcome{exit: 2, stdout: "", stderrNotEmpty: true})
}

// expectEarlyOrRuntimeError is for errors provable from the source text
// alone, which §16 lets an implementation report before execution. It
// asserts exit code 1 or 2, an empty stdout and a non-empty stderr.
func expectEarlyOrRuntimeError(tb testing.TB, src string) {
	tb.Helper()
	got := runSource(tb, src)
	if (got.exit != 1 && got.exit != 2) || got.stdout != "" || got.stderr == "" {
		tb.Errorf("source: %s\nwant: exit code 1 or 2, empty stdout, non-empty stderr\ngot:\n%v", src, got)
	}
}

// expectApprox asserts exit code 0 and that stdout is one number and a
// newline, within |got − want| ≤ 1e-15 × max(1, |want|) of want.
// Use it only where the spec leaves the exact value open.
func expectApprox(tb testing.TB, src string, want float64) {
	tb.Helper()
	got := runSource(tb, src)
	text, isLine := strings.CutSuffix(got.stdout, "\n")
	value, err := strconv.ParseFloat(text, 64)
	tolerance := 1e-15 * math.Max(1, math.Abs(want))
	if got.exit != 0 || !isLine || strings.Contains(text, "\n") || err != nil || !(math.Abs(value-want) <= tolerance) {
		tb.Errorf("source: %s\nwant: exit code 0, stdout one number within %g of %v\ngot:\n%v",
			src, tolerance, want, got)
	}
}

// expectLinesUnordered asserts exit code 0 and that the lines of stdout are
// the lines of want in any order, as a multiset. Use it only where the spec
// leaves the order open.
func expectLinesUnordered(tb testing.TB, src, want string) {
	tb.Helper()
	got := runSource(tb, src)
	gotLines, wantLines := sortedLines(got.stdout), sortedLines(want)
	if got.exit != 0 || !slices.Equal(gotLines, wantLines) {
		tb.Errorf("source: %s\nwant: exit code 0, stdout lines %q in any order\ngot:\n%v", src, wantLines, got)
	}
}

func sortedLines(s string) []string {
	lines := strings.Split(s, "\n")
	slices.Sort(lines)
	return lines
}

// Host-contract helpers. Only host_contract_test.go uses them.

// runFile runs `jscheme testdata/<name>` and checks the outcome.
func runFile(tb testing.TB, name string, want outcome) {
	tb.Helper()
	path := filepath.Join("testdata", name)
	checkOutcome(tb, "file: "+path, run(tb, "", path), want)
}

// runRepl runs `jscheme` with no arguments and input piped to stdin, and
// checks the outcome.
func runRepl(tb testing.TB, input string, want outcome) {
	tb.Helper()
	checkOutcome(tb, fmt.Sprintf("REPL input: %q", input), run(tb, input), want)
}

// expectUsageError runs `jscheme args...` and asserts exit code 3, an empty
// stdout and a non-empty stderr.
func expectUsageError(tb testing.TB, args ...string) {
	tb.Helper()
	checkOutcome(tb, fmt.Sprintf("args: %q", args), run(tb, "", args...), outcome{exit: 3, stdout: "", stderrNotEmpty: true})
}

// Rows. A chapter test is a table of rows, each a name and one check (a
// whole Act and Assert). runRows runs them as parallel subtests; it is the
// one loop over test data the suite has.

// check is the Act and Assert of one row.
type check func(testing.TB)

// row is one subtest: a name that starts with a section number
// and its check.
type row struct {
	name  string
	check check
}

// runRows runs every row as a parallel subtest of t.
func runRows(t *testing.T, rows []row) {
	t.Helper()
	t.Parallel()
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.check(t)
		})
	}
}

// The check constructors below wrap the assertion helpers, so that a row
// reads as one line: {"7.1 ...", output(`print(1)`, "1\n")}.

func output(src, want string) check {
	return func(tb testing.TB) { expectOutput(tb, src, want) }
}

func runtimeError(src, stdoutBefore string) check {
	return func(tb testing.TB) { expectRuntimeError(tb, src, stdoutBefore) }
}

func syntaxError(src string) check {
	return func(tb testing.TB) { expectSyntaxError(tb, src) }
}

func earlyOrRuntimeError(src string) check {
	return func(tb testing.TB) { expectEarlyOrRuntimeError(tb, src) }
}

func approx(src string, want float64) check {
	return func(tb testing.TB) { expectApprox(tb, src, want) }
}

func linesUnordered(src, want string) check {
	return func(tb testing.TB) { expectLinesUnordered(tb, src, want) }
}
