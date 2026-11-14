package evals

import (
	"reflect"
	"strings"
	"testing"
)

const tableHeader = "### Findings\n\n" +
	"| Severity | Reality | Area | Finding | Evidence | Test | Fix location |\n" +
	"|----------|---------|------|---------|----------|------|--------------|\n"

func answer(verdict string, rows ...string) string {
	return "---\nspec: \"X\"\nverdict: \"" + verdict + "\"\nreviewed_sha: \"" + strings.Repeat("a", 40) +
		"\"\nreviewer: \"nan/x\"\n---\n\n## Adversarial review\n\n" + tableHeader +
		strings.Join(rows, "\n") + "\n\n### Evaluator rubric\n\n" +
		"| Dimension | Grade (A-D) | Rationale |\n|---|---|---|\n| Correctness | B | fine |\n\n### Verdict\n" + verdict + "\n"
}

func TestParseFindingsReadsOnlyTheFindingsTable(t *testing.T) {
	md := answer("FAIL",
		"| **Major** | REAL | tests | the `a | b` pipe in a code span | ev | UNTESTED | tests |",
		"| Minor | THEORETICAL | code | second\r",
	)
	format, rows := ParseFindings(md)
	if format != FormatOK {
		t.Fatalf("format = %q, want ok", format)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (the rubric table is not findings): %+v", len(rows), rows)
	}
	if rows[0].Severity != "Major" || rows[1].Severity != "Minor" {
		t.Fatalf("severities = %q, %q; want Major, Minor", rows[0].Severity, rows[1].Severity)
	}
	if !strings.Contains(rows[0].Text, "a | b") {
		t.Fatalf("row text must be the raw line, pipes in code spans included: %q", rows[0].Text)
	}
	if strings.Contains(rows[1].Text, "\r") {
		t.Fatalf("a CRLF checkout must not leave \\r in the row text: %q", rows[1].Text)
	}
}

func TestParseFindingsTellsAMissingTableFromAnEmptyOne(t *testing.T) {
	if f, _ := ParseFindings("## Adversarial review\n\nNo table here.\n"); f != FormatNoTable {
		t.Fatalf("no table: format = %q, want no-table", f)
	}
	if f, rows := ParseFindings(answer("PASS")); f != FormatEmptyTable || len(rows) != 0 {
		t.Fatalf("header only: format = %q with %d rows, want empty-table", f, len(rows))
	}
}

func finding(id, floor string, all []string, anyOf ...string) Finding {
	return Finding{ID: id, SeverityFloor: floor, AnchorsAll: all, AnchorsAny: anyOf}
}

// The AI-055 round-2 shape: two rows name the same symbol and two expected
// findings anchor on it. A greedy first-fit gives the first row to whichever
// finding is listed first and can strand the other; the assignment must be
// one-to-one and maximal.
func TestGradeAssignsRowsOneToOneWhenFindingsShareASymbol(t *testing.T) {
	c := Case{
		ExpectedVerdict: "FAIL",
		Expected: []Finding{
			finding("false-verification-claim", "Major", []string{"AFileWithoutAClip"}, "does not exist", "delet"),
			finding("deleted-clip-test", "Major", []string{"AFileWithoutAClip"}, "removed"),
		},
	}
	md := answer("FAIL",
		"| Major | REAL | tests | `7d0c2b33` deleted the test: it removed `AFileWithoutAClipFailsClosed` | ev | UNTESTED | tests |",
		"| Major | REAL | spec | verification.md names `AFileWithoutAClipFailsClosed`, which does not exist | ev | n/a | spec |",
	)
	g := GradeAnswer(c, md)
	if len(g.Matched) != 2 || len(g.Missed) != 0 {
		t.Fatalf("matched %v, missed %v; want both findings matched", g.Matched, g.Missed)
	}
	if g.Recall != 1 || !g.VerdictMatch {
		t.Fatalf("recall = %v, verdict match = %v; want 1, true", g.Recall, g.VerdictMatch)
	}
	if len(g.Unadjudicated) != 0 {
		t.Fatalf("unadjudicated = %v, want none", g.Unadjudicated)
	}
}

func TestGradeOutcomes(t *testing.T) {
	base := Case{
		ExpectedVerdict: "FAIL",
		Expected:        []Finding{finding("count-exit", "Major", []string{"count++"}, "set -e")},
		Optional:        []Finding{finding("style", "", []string{"printf"})},
		Decoys:          []Finding{finding("bash32", "", []string{"bash 3.2"})},
		Noise:           []Finding{finding("tool-drift", "", []string{"golangci-lint v9"})},
	}
	cases := []struct {
		name          string
		verdict       string
		rows          []string
		matched       []string
		missed        []string
		optional      []string
		decoys        []string
		noise         []string
		unadjudicated int
		verdictMatch  bool
	}{
		{
			name:    "a matched finding",
			verdict: "FAIL",
			rows:    []string{"| Major | REAL | sh | `((count++))` exits under set -e when count is 0 | e | t | f |"},
			matched: []string{"count-exit"}, verdictMatch: true,
		},
		{
			name:    "a row below the severity floor does not satisfy the finding",
			verdict: "FAIL",
			rows:    []string{"| Minor | REAL | sh | `((count++))` exits under set -e | e | t | f |"},
			missed:  []string{"count-exit"}, unadjudicated: 1, verdictMatch: true,
		},
		{
			name:    "a stronger severity than the floor still satisfies it",
			verdict: "FAIL",
			rows:    []string{"| Blocker | REAL | sh | `((count++))` exits under set -e | e | t | f |"},
			matched: []string{"count-exit"}, verdictMatch: true,
		},
		{
			name:    "anchors_any needs at least one hit",
			verdict: "PASS",
			rows:    []string{"| Major | REAL | sh | `((count++))` is unusual | e | t | f |"},
			missed:  []string{"count-exit"}, unadjudicated: 1,
		},
		{
			name:    "a decoy row is a false positive even when it also names the real symbol",
			verdict: "FAIL",
			rows:    []string{"| Major | REAL | sh | `((count++))` under set -e breaks on Bash 3.2 | e | t | f |"},
			missed:  []string{"count-exit"}, decoys: []string{"bash32"}, verdictMatch: true,
		},
		{
			name:    "noise is discarded, optional earns credit, the rest is unadjudicated",
			verdict: "FAIL",
			rows: []string{
				"| Minor | REAL | lint | golangci-lint v9 reports a new check | e | t | f |",
				"| Minor | REAL | sh | prefer printf over echo | e | t | f |",
				"| Minor | REAL | docs | a typo in the runbook | e | t | f |",
			},
			missed: []string{"count-exit"}, noise: []string{"tool-drift"}, optional: []string{"style"},
			unadjudicated: 1, verdictMatch: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := GradeAnswer(base, answer(tc.verdict, tc.rows...))
			check := func(what string, got []Match, want []string) {
				t.Helper()
				ids := []string{}
				for _, m := range got {
					ids = append(ids, m.ID)
				}
				if want == nil {
					want = []string{}
				}
				if !reflect.DeepEqual(ids, want) {
					t.Errorf("%s = %v, want %v", what, ids, want)
				}
			}
			check("matched", g.Matched, tc.matched)
			check("optional", g.OptionalMatched, tc.optional)
			check("decoys", g.DecoyHits, tc.decoys)
			check("noise", g.Noise, tc.noise)
			missed := g.Missed
			if missed == nil {
				missed = []string{}
			}
			wantMissed := tc.missed
			if wantMissed == nil {
				wantMissed = []string{}
			}
			if !reflect.DeepEqual(missed, wantMissed) {
				t.Errorf("missed = %v, want %v", missed, wantMissed)
			}
			if len(g.Unadjudicated) != tc.unadjudicated {
				t.Errorf("unadjudicated = %d, want %d", len(g.Unadjudicated), tc.unadjudicated)
			}
			if g.VerdictMatch != tc.verdictMatch {
				t.Errorf("verdict match = %v, want %v", g.VerdictMatch, tc.verdictMatch)
			}
		})
	}
}

// An answer whose table cannot be found is a format failure. It must not read
// as a review that found nothing, which on a PASS case would score perfectly.
func TestGradeReportsAMissingTableAsAFormatFailure(t *testing.T) {
	c := Case{ExpectedVerdict: "FAIL", Expected: []Finding{finding("x", "Minor", []string{"x"})}}
	g := GradeAnswer(c, "---\nverdict: FAIL\nreviewed_sha: \""+strings.Repeat("a", 40)+"\"\n---\nprose only\n")
	if g.Format != FormatNoTable {
		t.Fatalf("format = %q, want no-table", g.Format)
	}
}

func TestGradeWeightsRecallBySeverityFloor(t *testing.T) {
	c := Case{
		ExpectedVerdict: "FAIL",
		Expected: []Finding{
			finding("blocker", "Blocker", []string{"alpha"}),
			finding("minor", "Minor", []string{"beta"}),
		},
	}
	g := GradeAnswer(c, answer("FAIL", "| Minor | REAL | x | beta | e | t | f |"))
	if g.Recall != 0.5 {
		t.Fatalf("recall = %v, want 0.5", g.Recall)
	}
	if g.WeightedRecall != 0.25 { // minor weighs 1 of the 3+1 total
		t.Fatalf("weighted recall = %v, want 0.25", g.WeightedRecall)
	}
}
