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

// A replay's source is a model's review, and a false claim it made is exactly
// where a decoy comes from, so a decoy may match its own source. A decoy too
// broad to be safe still fails the check: decoys are graded before expected
// entries, so one that swallows a true row leaves that entry unmatched.
func TestCheckLetsAReplaySourceCarryItsOwnDecoy(t *testing.T) {
	review := sourceReview("FAIL", "nan/x", append([]string{"| Minor | REAL | sh | fails on bash 3.2 | e | t | f |"}, goodRows...)...)
	if p := problemsOf(t, writeTree(t, goodCase, "x-r1.json", review)); len(p) != 0 {
		t.Fatalf("a replay source carrying its own decoy was refused: %v", p)
	}
	tooBroad := strings.Replace(goodCase, `["bash 3.2"]`, `["ClipFailsClosed"]`, 1)
	tooBroad = strings.Replace(tooBroad, `breaks on bash 3.2`, `ClipFailsClosed`, 1)
	p := problemsOf(t, writeTree(t, tooBroad, "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...)))
	if !strings.Contains(strings.Join(p, "\n"), "deleted-test") {
		t.Fatalf("a decoy swallowing an expected row passed: %v", p)
	}
}

const plantedCase = `{
  "id": "p1", "repo": "o/r", "kind": "planted", "role": "spec-reviewer",
  "prompt": "harness/evals/planted/p1/prompt.md", "reference_answer": "harness/evals/planted/p1/reference.md",
  "key_author": "human", "expected_verdict": "FAIL",
  "expected": [{"id": "count", "severity_floor": "Major", "anchors_all": ["count++"], "confirmed_by": "planted"}],
  "optional": [], "noise": [],
  "decoys": [{"id": "bash32", "anchors_all": ["bash 3.2"], "reason": "false", "sample": "fails on bash 3.2"}]
}`

func writePlanted(t *testing.T, reference string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "harness", "evals", "planted", "p1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"case.json": plantedCase, "prompt.md": "review this", "reference.md": reference} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// A planted reference answer has no commit to name, so it carries a verdict and
// no reviewed_sha; and being the ideal answer, it must not carry a decoy.
func TestCheckHoldsAPlantedReferenceToTheIdealAnswer(t *testing.T) {
	ideal := "---\nverdict: FAIL\n---\n\n" + tableHeader + "| Major | REAL | sh | `((count++))` exits under set -e | e | t | f |\n"
	if p := problemsOf(t, writePlanted(t, ideal)); len(p) != 0 {
		t.Fatalf("an ideal planted reference was refused: %v", p)
	}
	withDecoy := ideal + "| Minor | REAL | sh | fails on bash 3.2 | e | t | f |\n"
	if p := problemsOf(t, writePlanted(t, withDecoy)); !strings.Contains(strings.Join(p, "\n"), "bash32") {
		t.Fatalf("a planted reference carrying a decoy passed: %v", p)
	}
}
