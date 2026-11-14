package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sha = "06f3100b7efafdfdf77d250553b23994eefc7bd2"

func sourceReview(verdict, reviewer string, rows ...string) string {
	return strings.Replace(answer(verdict, rows...), `reviewer: "nan/x"`, `reviewer: "`+reviewer+`"`, 1)
}

const goodCase = `{
  "id": "x-r1", "repo": "o/r", "kind": "replay", "role": "spec-reviewer",
  "spec": "X", "source_review": "specs/archive/X/review.md", "key_author": "nan/x",
  "reviewed_sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "base_sha": "` + sha + `",
  "base_derivation": "stated in the review's Sources line",
  "expected_verdict": "FAIL",
  "expected": [{"id": "deleted-test", "severity_floor": "Major", "anchors_all": ["ClipFailsClosed"], "anchors_any": ["deleted"], "confirmed_by": "restored in abc"}],
  "optional": [{"id": "wording", "anchors_all": ["wording"]}],
  "decoys": [{"id": "bash32", "anchors_all": ["bash 3.2"], "reason": "false: bash 3.2 handles it", "sample": "| Major | REAL | sh | breaks on bash 3.2 |"}],
  "noise": []
}`

var goodRows = []string{
	"| Major | REAL | tests | the commit deleted `ClipFailsClosed` | e | t | f |",
	"| Minor | THEORETICAL | code | message wording misleads | e | t | f |",
}

// writeTree lays out a repository with one case and its source review.
func writeTree(t *testing.T, caseJSON, caseName, review string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(".dotf/evals/cases/"+caseName, caseJSON)
	if review != "" {
		write("specs/archive/X/review.md", review)
	}
	return root
}

func problemsOf(t *testing.T, root string) []string {
	t.Helper()
	results, err := Check(root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	return results[0].Problems
}

func TestCheckAcceptsASelfConsistentCase(t *testing.T) {
	root := writeTree(t, goodCase, "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...))
	if p := problemsOf(t, root); len(p) != 0 {
		t.Fatalf("a self-consistent case reported problems: %v", p)
	}
}

// Each mutation breaks one property `check` exists to hold. A source review
// never contains a decoy, so a decoy's anchors are only exercised by its
// recorded sample; without that sample a dead decoy would pass forever.
func TestCheckReportsEachBrokenProperty(t *testing.T) {
	cases := []struct {
		name     string
		caseJSON string
		file     string
		review   string
		want     string
	}{
		{"file name differs from id", goodCase, "other.json", sourceReview("FAIL", "nan/x", goodRows...), "file name"},
		{"source review absent", goodCase, "x-r1.json", "", "source_review"},
		{"schema violation", strings.Replace(goodCase, `"kind": "replay"`, `"kind": "x"`, 1), "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...), "schema"},
		{"reviewed_sha differs from the source", strings.Replace(goodCase, strings.Repeat("a", 40), sha, 1), "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...), "reviewed_sha"},
		{"verdict differs from the source", goodCase, "x-r1.json", sourceReview("PASS", "nan/x", goodRows...), "verdict"},
		{"key_author differs from the reviewer", goodCase, "x-r1.json", sourceReview("FAIL", "nan/y", goodRows...), "key_author"},
		{"expected anchor rotted", strings.Replace(goodCase, `"ClipFailsClosed"`, `"ClipFailsOpen"`, 1), "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...), "deleted-test"},
		{"optional entry unmatched", strings.Replace(goodCase, `["wording"]`, `["phrasing"]`, 1), "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...), "wording"},
		{"decoy sample does not match its anchors", strings.Replace(goodCase, `breaks on bash 3.2`, `breaks on zsh`, 1), "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...), "bash32"},
		{"source carries a decoy", goodCase, "x-r1.json", sourceReview("FAIL", "nan/x", append([]string{"| Minor | REAL | sh | fails on bash 3.2 | e | t | f |"}, goodRows...)...), "bash32"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.caseJSON, tc.file, tc.review)
			p := problemsOf(t, root)
			if !strings.Contains(strings.Join(p, "\n"), tc.want) {
				t.Fatalf("problems %v do not mention %q", p, tc.want)
			}
		})
	}
}

func TestCheckFindsNothingInARepositoryWithoutCases(t *testing.T) {
	results, err := Check(t.TempDir())
	if err != nil || len(results) != 0 {
		t.Fatalf("results %v, err %v; want none and no error", results, err)
	}
}
