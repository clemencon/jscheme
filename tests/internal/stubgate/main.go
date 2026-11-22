// Command stubgate checks the stub gate: when the
// suite runs against a binary that prints nothing and exits 0, every leaf test
// must fail. A leaf test that passes, is skipped or has no result is reported.
//
// It reads `go test -json` output on stdin:
//
//	JSCHEME_BIN=testdata/stub go test -json . | go run ./internal/stubgate
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"jscheme/tests/internal/testjson"
)

func main() {
	pkg := flag.String("package", "jscheme/tests", "import path of the conformance suite")
	flag.Parse()
	os.Exit(run(os.Stdin, os.Stdout, *pkg))
}

// run reads the events of pkg from in, writes a report to out and returns the
// exit code: 0 if every leaf test failed, 1 otherwise.
func run(in io.Reader, out io.Writer, pkg string) int {
	events, err := testjson.Read(in, pkg)
	if err != nil {
		fmt.Fprintf(out, "stubgate: reading go test output: %v\n", err)
		return 1
	}

	var order []string
	result := map[string]string{}
	for _, e := range events {
		switch e.Action {
		case "run":
			if _, seen := result[e.Test]; !seen {
				order = append(order, e.Test)
				result[e.Test] = "none"
			}
		case "pass", "fail", "skip":
			result[e.Test] = e.Action
		}
	}

	parent := map[string]bool{}
	for _, name := range order {
		if i := strings.LastIndex(name, "/"); i >= 0 {
			parent[name[:i]] = true
		}
	}

	leaves, bad := 0, 0
	for _, name := range order {
		if parent[name] {
			continue
		}
		leaves++
		if result[name] != "fail" {
			bad++
			fmt.Fprintf(out, "%s %s\n", result[name], name)
		}
	}

	switch {
	case leaves == 0:
		fmt.Fprintln(out, "stubgate: no tests ran")
		return 1
	case bad > 0:
		fmt.Fprintf(out, "stubgate: %d of %d leaf tests did not fail against the stub (vacuous tests)\n", bad, leaves)
		return 1
	}
	fmt.Fprintf(out, "stubgate: %d leaf tests, all failed\n", leaves)
	return 0
}
