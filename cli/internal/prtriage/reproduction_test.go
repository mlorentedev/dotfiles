package prtriage

import (
	"strings"
	"testing"
	"time"
)

var (
	findingAt = time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	triageAt  = findingAt.Add(time.Hour)
)

func finding(class, path string, line int) Comment {
	return Comment{Author: "github-actions[bot]", Body: "**A claim**\n\nClass: " + class + "\nwhy", Path: path, Line: line, CreatedAt: findingAt}
}

func triagedWith(table string) PR {
	return PR{Number: 7, Comments: []Comment{{
		Author: "mlorentedev", AuthorAssociation: "OWNER", CreatedAt: triageAt,
		Body: "## Review triage\n\n| Item | Disposition | Reason |\n|---|---|---|\n" + table,
	}}}
}

// TOOL-017 AC4 and AC6 under the owner's ruling on #1062 (option A, and the
// gate refuses): a REAL finding is applied only with a recorded reproduction,
// and refuted only with a recorded failure to reproduce. Never skipped.
func TestUnreproducedFindings(t *testing.T) {
	const (
		reproduced = "Reproduction: reproduced (exit 1) on 0123456789ab: `go test ./x`"
		refuted    = "Reproduction: did-not-reproduce (exit 0) on 0123456789ab: `go test ./x`"
		notRun     = "Reproduction: could-not-run (exit 127) on 0123456789ab: `nope`"
	)
	for _, tc := range []struct {
		name    string
		finding Comment
		table   string
		want    string // a fragment of the problem, or "" for none
	}{
		{"applied with a reproduction", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | apply | " + reproduced + " |", ""},
		{"applied with a bold word and prose", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | **Applied** in abc: fixed | " + reproduced + " |", ""},
		{"applied without one", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | apply | looks right |", "reproduced"},
		{"applied on a check that could not run", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | apply | " + notRun + " |", "reproduced"},
		{"applied on a refutation", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | apply | " + refuted + " |", "reproduced"},
		{"refuted with its evidence", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | refuted | " + refuted + " |", ""},
		{"refuted without it", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | refuted | the claim is wrong |", "did-not-reproduce"},
		{"skipped", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | skip | wrong |", "never skipped"},
		{"declined", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | Declined | wrong |", "never skipped"},
		{"deferred to a ticket", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | defer | filed #9 |", ""},
		{"no row", finding("REAL", "cli/a.go", 4), "| `cli/b.go:4` | apply | " + reproduced + " |", "no row"},
		{"a longer line number is another row", finding("REAL", "cli/a.go", 4), "| `cli/a.go:40` | apply | " + reproduced + " |", "no row"},
		{"a row with no disposition", finding("REAL", "cli/a.go", 4), "| `cli/a.go:4` | see above | x |", "no disposition"},
		{"named by its base name", finding("REAL", "cli/internal/a.go", 4), "| `a.go:4` | apply | " + reproduced + " |", ""},
		{"a disposition word in the item cell is not the disposition", finding("REAL", "cli/a.go", 4), "| Skipped lines in `cli/a.go:4` | defer | filed #9 |", ""},
		{"a THEORETICAL finding needs no reproduction", finding("THEORETICAL", "cli/a.go", 4), "| `cli/a.go:4` | apply | fine |", ""},
		{"an unclassified finding is not gated here", finding("REAL?", "cli/a.go", 4), "| `cli/a.go:4` | apply | fine |", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := UnreproducedFindings(triagedWith(tc.table), []Comment{tc.finding}, classifyingRegistry())
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("got %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0].Problem, tc.want) {
				t.Fatalf("got %+v, want one problem naming %q", got, tc.want)
			}
		})
	}
}

// Only the triage record counts, and only for findings it could have seen:
// a finding newer than the record already makes the PR pending, and a
// record from an author without standing is no record (#2870).
func TestUnreproducedFindingsReadsOnlyTheStandingRecord(t *testing.T) {
	f := finding("REAL", "cli/a.go", 4)
	pr := triagedWith("| `cli/a.go:4` | apply | no reproduction |")
	pr.Comments = append(pr.Comments, Comment{
		Author: "drive-by", AuthorAssociation: "NONE", CreatedAt: triageAt.Add(time.Minute),
		Body: "## Review triage\n\n| `cli/a.go:4` | apply | Reproduction: reproduced (exit 1) on x: `y` |",
	})
	if got := UnreproducedFindings(pr, []Comment{f}, classifyingRegistry()); len(got) != 1 {
		t.Fatalf("a record without standing cleared the finding: %+v", got)
	}
	if got := UnreproducedFindings(PR{Number: 7}, []Comment{f}, classifyingRegistry()); len(got) != 0 {
		t.Fatalf("a PR with no record was judged: %+v", got)
	}
}

// An inline finding added or edited after the triage can arrive without a new
// counted review, so nothing else makes the PR pending: it is judged like any
// other, and a record that cannot have seen it has no row for it (#2996).
func TestUnreproducedFindingsJudgesAFindingNewerThanTheRecord(t *testing.T) {
	late := finding("REAL", "cli/a.go", 4)
	late.CreatedAt = triageAt.Add(time.Hour)
	got := UnreproducedFindings(triagedWith("| `cli/b.go:1` | defer | filed #9 |"), []Comment{late}, classifyingRegistry())
	if len(got) != 1 || !strings.Contains(got[0].Problem, "newer than the triage record") {
		t.Fatalf("a late REAL finding went unjudged: %+v", got)
	}
	edited := finding("REAL", "cli/a.go", 4)
	edited.UpdatedAt = triageAt.Add(time.Hour)
	got = UnreproducedFindings(triagedWith("| `cli/a.go:4` | apply | looks right |"), []Comment{edited}, classifyingRegistry())
	if len(got) != 1 || !strings.Contains(got[0].Problem, "reproduced") {
		t.Fatalf("a finding edited after the triage escaped its row's check: %+v", got)
	}
}

// Two findings at one line of same-named files must not read each other's
// row: when a base name is shared, only the full path names a finding (#2996).
func TestUnreproducedFindingsTellsSameNamedFilesApart(t *testing.T) {
	a, b := finding("REAL", "a/util.go", 4), finding("REAL", "b/util.go", 4)
	const reproduced = "Reproduction: reproduced (exit 1) on 0123456789ab: `go test ./x`"
	pr := triagedWith("| `b/util.go:4` | apply | " + reproduced + " |\n| `a/util.go:4` | apply | looks right |")
	got := UnreproducedFindings(pr, []Comment{a, b}, classifyingRegistry())
	if len(got) != 1 || got[0].Finding.Path != "a/util.go" {
		t.Fatalf("got %+v, want only a/util.go held", got)
	}
	pr = triagedWith("| `util.go:4` | apply | " + reproduced + " |")
	if got := UnreproducedFindings(pr, []Comment{a, b}, classifyingRegistry()); len(got) != 2 {
		t.Fatalf("a shared base name cleared both findings: %+v", got)
	}
	pr = triagedWith("| `x/a/util.go:4` | apply | " + reproduced + " |\n| `b/util.go:4` | defer | filed #9 |")
	if got := UnreproducedFindings(pr, []Comment{a, b}, classifyingRegistry()); len(got) != 1 || got[0].Finding.Path != "a/util.go" {
		t.Fatalf("a longer path cleared a/util.go: %+v", got)
	}
	// The look-alike need not be a finding: a row naming another directory
	// never names this one, whatever else the PR carries.
	pr = triagedWith("| `b/util.go:4` | apply | " + reproduced + " |")
	if got := UnreproducedFindings(pr, []Comment{a}, classifyingRegistry()); len(got) != 1 {
		t.Fatalf("a row for b/util.go cleared the only finding, a/util.go: %+v", got)
	}
	pr = triagedWith("| `internal/a.go:4` | apply | " + reproduced + " |")
	if got := UnreproducedFindings(pr, []Comment{finding("REAL", "cli/internal/a.go", 4)}, classifyingRegistry()); len(got) != 0 {
		t.Fatalf("a row naming the path's tail did not clear it: %+v", got)
	}
}
