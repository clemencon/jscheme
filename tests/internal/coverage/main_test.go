package main

import (
	"slices"
	"strings"
	"testing"
)

const pkg = "jsrust/tests"

const spec = `<!-- pdf-page 1 / printed none -->
# <a id="sec-1"></a>1 Scope

Text.

# <a id="sec-7"></a>7 Lexical Conventions
## <a id="sec-7.1"></a>7.1 White Space
## <a id="sec-7.2"></a>7.2 Line Terminators
# <a id="sec-8"></a>8 Types
`

// runs turns test names into "run" events of pkg, as go test -json writes them.
func runs(names ...string) string {
	var b strings.Builder
	for _, n := range names {
		b.WriteString(`{"Action":"run","Package":"` + pkg + `","Test":"` + n + `"}` + "\n")
	}
	return b.String()
}

func TestRun(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name       string
		events     string
		exclusions string
		wantCode   int
		want       []string // lines the report must contain
		notWant    []string // lines it must not contain
	}{
		{
			name:     "a section named by a subtest is covered, the others are listed",
			events:   runs("TestS7", "TestS7/7.1_a_tab_is_white_space"),
			wantCode: 1,
			want: []string{
				"uncovered: 1 Scope", "uncovered: 7 Lexical Conventions",
				"uncovered: 7.2 Line Terminators", "uncovered: 8 Types",
			},
			notWant: []string{"uncovered: 7.1 White Space"},
		},
		{
			name: "a parent is covered when all its children are",
			events: runs("TestS7/7.1_a_tab_is_white_space", "TestS7/7.2_LF_ends_a_line",
				"TestS8/8_x", "TestS1/1_x"),
			wantCode: 0,
			want:     []string{"coverage: all 5 sections covered"},
		},
		{
			name:     "a parent named by a subtest does not cover its children",
			events:   runs("TestS7/7_x", "TestS8/8_x", "TestS1/1_x"),
			wantCode: 1,
			want:     []string{"uncovered: 7.1 White Space", "uncovered: 7.2 Line Terminators"},
			notWant:  []string{"uncovered: 7 Lexical Conventions"},
		},
		{
			name:       "an excluded section counts towards its parent",
			events:     runs("TestS7/7.1_x", "TestS8/8_x", "TestS1/1_x"),
			exclusions: "# comment\n\n7.2 editorial\n",
			wantCode:   0,
			want:       []string{"coverage: all 5 sections covered (1 excluded)"},
		},
		{
			name:       "an exclusion without a reason is an error",
			events:     runs("TestS7/7.1_x"),
			exclusions: "7.2\n",
			wantCode:   1,
			want:       []string{"exclusions line 1: no reason given for 7.2"},
		},
		{
			name:       "an exclusion of an unknown section is an error",
			events:     runs("TestS7/7.1_x"),
			exclusions: "7.9 typo\n",
			wantCode:   1,
			want:       []string{"exclusions line 1: no section 7.9 in the spec"},
		},
		{
			name:     "a subtest naming an unknown section is reported",
			events:   runs("TestS7/7.9_does_not_exist"),
			wantCode: 1,
			want:     []string{"unknown section: TestS7/7.9_does_not_exist"},
		},
		{
			name:     "a subtest without a section number is reported",
			events:   runs("TestS7/tab_is_white_space", "TestS7/7.1"),
			wantCode: 1,
			want: []string{
				"no section number: TestS7/tab_is_white_space",
				"no section number: TestS7/7.1",
			},
		},
		{
			name: "host subtests need no section number",
			events: runs("TestHostREPL/host:REPL_x",
				"TestS7/7.1_x", "TestS7/7.2_x", "TestS8/8_x", "TestS1/1_x"),
			wantCode: 0,
			want:     []string{"coverage: all 5 sections covered"},
		},
		{
			name:     "only the first subtest level carries the section",
			events:   runs("TestS7/7.1_x/nested_free_text", "TestS7/7.2_x", "TestS8/8_x", "TestS1/1_x"),
			wantCode: 0,
		},
		{
			name: "other packages and non-run events are ignored",
			events: `{"Action":"run","Package":"other","Test":"TestS8/8_x"}` + "\n" +
				`{"Action":"output","Package":"` + pkg + `","Test":"TestS1/1_x"}` + "\n" +
				"not json\n" + runs("TestS7/7.1_x", "TestS7/7.2_x"),
			wantCode: 1,
			want:     []string{"uncovered: 1 Scope", "uncovered: 8 Types"},
		},
		{
			name:     "no tests at all is an error",
			wantCode: 1,
			want:     []string{"coverage: no tests ran"},
		},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			var out strings.Builder
			code := run(strings.NewReader(r.events), strings.NewReader(spec),
				strings.NewReader(r.exclusions), &out, pkg)
			report := out.String()
			if code != r.wantCode {
				t.Errorf("exit code %d, want %d; report:\n%s", code, r.wantCode, report)
			}
			lines := strings.Split(report, "\n")
			for _, w := range r.want {
				if !slices.Contains(lines, w) {
					t.Errorf("report lacks line %q:\n%s", w, report)
				}
			}
			for _, w := range r.notWant {
				if slices.Contains(lines, w) {
					t.Errorf("report has line %q:\n%s", w, report)
				}
			}
		})
	}
}
