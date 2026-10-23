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
	late := f
	late.CreatedAt = triageAt.Add(time.Hour)
	if got := UnreproducedFindings(triagedWith("| x | apply | y |"), []Comment{late}, classifyingRegistry()); len(got) != 0 {
		t.Fatalf("a finding newer than the record was judged against it: %+v", got)
	}
	if got := UnreproducedFindings(PR{Number: 7}, []Comment{f}, classifyingRegistry()); len(got) != 0 {
		t.Fatalf("a PR with no record was judged: %+v", got)
	}
}
