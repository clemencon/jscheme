package main

import (
	"strings"
	"testing"
)

const pkg = "jscheme/tests"

// events turns compact "action test" lines into go test -json output for pkg.
func events(lines ...string) string {
	var b strings.Builder
	for _, l := range lines {
		action, test, _ := strings.Cut(l, " ")
		b.WriteString(`{"Action":"` + action + `","Package":"` + pkg + `","Test":"` + test + `"}` + "\n")
	}
	return b.String()
}

func TestRun(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name     string
		input    string
		wantCode int
		wantOut  []string // substrings of the report
	}{
		{
			"every leaf failed",
			events("run TestA", "run TestA/x", "fail TestA/x", "fail TestA"),
			0, []string{"1 leaf tests, all failed"},
		},
		{
			"a passing leaf is reported",
			events("run TestA", "run TestA/x", "pass TestA/x", "run TestA/y", "fail TestA/y", "fail TestA"),
			1, []string{"pass TestA/x"},
		},
		{
			"a skipped leaf is reported",
			events("run TestA", "skip TestA"),
			1, []string{"skip TestA"},
		},
		{
			"a leaf with no result is reported",
			events("run TestA"),
			1, []string{"none TestA"},
		},
		{
			"no tests at all is an error",
			"",
			1, []string{"no tests ran"},
		},
		{
			"other packages are ignored",
			`{"Action":"pass","Package":"jscheme/tests/internal/coverage","Test":"TestX"}` + "\n" +
				events("run TestA", "fail TestA"),
			0, []string{"1 leaf tests, all failed"},
		},
		{
			"non-JSON lines are ignored",
			"FAIL\n" + events("run TestA", "fail TestA"),
			0, []string{"1 leaf tests, all failed"},
		},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			var out strings.Builder
			code := run(strings.NewReader(r.input), &out, pkg)
			if code != r.wantCode {
				t.Errorf("exit code %d, want %d; output:\n%s", code, r.wantCode, out.String())
			}
			for _, w := range r.wantOut {
				if !strings.Contains(out.String(), w) {
					t.Errorf("output does not contain %q:\n%s", w, out.String())
				}
			}
		})
	}
}
