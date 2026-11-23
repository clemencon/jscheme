// Command coverage checks the coverage gate: every
// numbered heading of the Markdown spec must be named by a test, excluded as
// editorial, or have all of its children covered.
//
// It reads `go test -json` output on stdin, takes the
// leading section number of every row (the first subtest level), and diffs
// those against the `sec-*` anchors of the spec minus the exclusions. Every
// row that ran counts, whether it passed, failed or was skipped.
//
//	go test -json ./... | go run ./internal/coverage
//
// It prints the uncovered sections, the rows whose names carry no valid
// section number, and any malformed exclusion, and exits 1 while any remain.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"jsrust/tests/internal/testjson"
)

func main() {
	specPath := flag.String("spec", "../spec/ECMA-262_1st_edition_june_1997.md", "Markdown spec")
	exclPath := flag.String("exclusions", "coverage-exclusions.txt", "sections excluded from coverage")
	pkg := flag.String("package", "jsrust/tests", "import path of the conformance suite")
	flag.Parse()

	spec, err := os.Open(*specPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
	defer spec.Close()
	excl, err := os.Open(*exclPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
	defer excl.Close()
	os.Exit(run(os.Stdin, spec, excl, os.Stdout, *pkg))
}

// section is one numbered heading of the spec.
type section struct {
	number string // "15.5.4.3"
	title  string // "String.prototype.valueOf()"
}

var (
	// anchor matches a spec heading: `### <a id="sec-7.1"></a>7.1 White Space`.
	anchor = regexp.MustCompile(`^#+ <a id="sec-([0-9.]+)"></a>\S+\s+(.*)$`)
	// leadingSection matches the start of a row name, after Go turned its
	// spaces into underscores: "11.9.3_NaN_is_not_equal_to_itself".
	leadingSection = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*)_.`)
)

// Prefixes of rows that test the host contract, not a section.
var unnumberedPrefixes = []string{"host:"}

// run writes the coverage report for the events of pkg to out and returns the
// exit code: 0 when every section is covered and every input is well formed.
func run(events, spec, exclusions io.Reader, out io.Writer, pkg string) int {
	sections, err := readSpec(spec)
	if err != nil {
		fmt.Fprintln(out, "coverage: reading spec:", err)
		return 1
	}
	known := map[string]bool{}
	for _, s := range sections {
		known[s.number] = true
	}

	ok := true
	excluded, problems := readExclusions(exclusions, known)
	for _, p := range problems {
		fmt.Fprintln(out, p)
		ok = false
	}

	evs, err := testjson.Read(events, pkg)
	if err != nil {
		fmt.Fprintln(out, "coverage: reading go test output:", err)
		return 1
	}
	named := map[string]bool{}
	rows := 0
	for _, e := range evs {
		if e.Action != "run" {
			continue
		}
		rows++
		parts := strings.Split(e.Test, "/")
		if len(parts) < 2 {
			continue // a Test function, not a row
		}
		row := parts[1]
		if hasUnnumberedPrefix(row) {
			continue
		}
		m := leadingSection.FindStringSubmatch(row)
		switch {
		case m == nil:
			fmt.Fprintln(out, "no section number:", e.Test)
			ok = false
		case !known[m[1]]:
			fmt.Fprintln(out, "unknown section:", e.Test)
			ok = false
		default:
			named[m[1]] = true
		}
	}
	if rows == 0 {
		fmt.Fprintln(out, "coverage: no tests ran")
		return 1
	}

	children := map[string][]string{}
	for _, s := range sections {
		if p := parentOf(s.number); p != "" {
			children[p] = append(children[p], s.number)
		}
	}
	memo := map[string]bool{}
	var covered func(string) bool
	covered = func(n string) bool {
		if c, seen := memo[n]; seen {
			return c
		}
		c := named[n] || excluded[n]
		if !c && len(children[n]) > 0 {
			c = true
			for _, child := range children[n] {
				if !covered(child) {
					c = false
				}
			}
		}
		memo[n] = c
		return c
	}

	uncovered := 0
	for _, s := range sections {
		if !covered(s.number) {
			fmt.Fprintf(out, "uncovered: %s %s\n", s.number, s.title)
			uncovered++
		}
	}

	excludedNote := ""
	if len(excluded) > 0 {
		excludedNote = fmt.Sprintf(" (%d excluded)", len(excluded))
	}
	if uncovered > 0 {
		fmt.Fprintf(out, "coverage: %d of %d sections covered%s; %d uncovered\n",
			len(sections)-uncovered, len(sections), excludedNote, uncovered)
		return 1
	}
	fmt.Fprintf(out, "coverage: all %d sections covered%s\n", len(sections), excludedNote)
	if !ok {
		return 1
	}
	return 0
}

func hasUnnumberedPrefix(row string) bool {
	for _, p := range unnumberedPrefixes {
		if strings.HasPrefix(row, p) {
			return true
		}
	}
	return false
}

// parentOf returns "15.5.4" for "15.5.4.3", and "" for a chapter.
func parentOf(n string) string {
	i := strings.LastIndex(n, ".")
	if i < 0 {
		return ""
	}
	return n[:i]
}

// readSpec returns the numbered headings of the spec, in document order.
func readSpec(r io.Reader) ([]section, error) {
	var sections []section
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if m := anchor.FindStringSubmatch(sc.Text()); m != nil {
			sections = append(sections, section{number: m[1], title: m[2]})
		}
	}
	return sections, sc.Err()
}

// readExclusions parses lines of the form "<section> <reason>". Blank lines
// and lines starting with "#" are ignored. Every section must exist in the
// spec and every exclusion must give a reason.
func readExclusions(r io.Reader, known map[string]bool) (map[string]bool, []string) {
	excluded := map[string]bool{}
	var problems []string
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		number, reason := text, ""
		if i := strings.IndexAny(text, " \t"); i >= 0 {
			number, reason = text[:i], text[i+1:]
		}
		switch {
		case !known[number]:
			problems = append(problems, fmt.Sprintf("exclusions line %d: no section %s in the spec", line, number))
		case strings.TrimSpace(reason) == "":
			problems = append(problems, fmt.Sprintf("exclusions line %d: no reason given for %s", line, number))
		default:
			excluded[number] = true
		}
	}
	if err := sc.Err(); err != nil {
		problems = append(problems, fmt.Sprintf("exclusions: %v", err))
	}
	return excluded, problems
}
