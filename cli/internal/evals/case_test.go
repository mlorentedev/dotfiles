package evals

import (
	"strings"
	"testing"
)

const validReplay = `{
  "id": "ai-055-r2",
  "repo": "mlorentedev/dotfiles",
  "kind": "replay",
  "role": "spec-reviewer",
  "spec": "AI-055-budget-guard-readable-reserve",
  "source_review": "specs/archive/AI-055-budget-guard-readable-reserve/review-round2.md",
  "key_author": "nan/glm5.3-flash",
  "reviewed_sha": "06f3100b7efafdfdf77d250553b23994eefc7bd2",
  "base_sha": "2aa3ad1693bd95b19de67fd37aea1df8e798b6f1",
  "base_derivation": "the launch request's stated base",
  "fetch_ref": "refs/pull/3349/head",
  "expected_verdict": "FAIL",
` + expectedBlock + `
  "optional": [], "decoys": [], "noise": []
}`

const expectedBlock = `"expected": [{
    "id": "deleted-clip-test",
    "severity_floor": "Major",
    "anchors_all": ["AFileWithoutAClipFailsClosed"],
    "anchors_any": ["delet"],
    "confirmed_by": "restored in 1234abcd"
  }],`

func TestValidateCaseAcceptsAWellFormedReplay(t *testing.T) {
	if err := ValidateCase([]byte(validReplay)); err != nil {
		t.Fatalf("a well-formed replay case was rejected: %v", err)
	}
}

// Each mutation is one way a case can be wrong while still being valid JSON.
// The schema is the only thing between a mistyped key and a check that passes
// because the field it would have constrained is absent.
func TestValidateCaseRejectsMalformedCases(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
	}{
		{"replay without reviewed_sha", `"reviewed_sha": "06f3100b7efafdfdf77d250553b23994eefc7bd2",`, ``},
		{"replay without base_sha", `"base_sha": "2aa3ad1693bd95b19de67fd37aea1df8e798b6f1",`, ``},
		{"replay without source_review", `"source_review": "specs/archive/AI-055-budget-guard-readable-reserve/review-round2.md",`, ``},
		{"unknown kind", `"kind": "replay"`, `"kind": "replayed"`},
		{"unknown role", `"role": "spec-reviewer"`, `"role": "reviewer"`},
		{"short sha", `"reviewed_sha": "06f3100b7efafdfdf77d250553b23994eefc7bd2"`, `"reviewed_sha": "06f3100b"`},
		{"unknown verdict", `"expected_verdict": "FAIL"`, `"expected_verdict": "FAILED"`},
		{"replay with no confirmed finding", expectedBlock, `"expected": [],`},
		{"empty anchors_all", `"anchors_all": ["AFileWithoutAClipFailsClosed"]`, `"anchors_all": []`},
		{"blank anchor", `"anchors_any": ["delet"]`, `"anchors_any": [" "]`},
		{"unknown severity floor", `"severity_floor": "Major"`, `"severity_floor": "Huge"`},
		{"misspelled finding key", `"anchors_any"`, `"anchor_any"`},
		{"replay without base_derivation", `"base_derivation": "the launch request's stated base",`, ``},
		{"decoy without a sample", `"decoys": []`, `"decoys": [{"id": "d", "anchors_all": ["x"], "reason": "false"}]`},
		{"planted without a reference answer", `"kind": "replay"`, `"kind": "planted", "prompt": "harness/evals/planted/p/prompt.md"`},
		{"absolute source path", `"source_review": "specs/`, `"source_review": "/specs/`},
		{"misspelled case key", `"key_author"`, `"key_authr"`},
		{"repo not owner/name", `"repo": "mlorentedev/dotfiles"`, `"repo": "dotfiles"`},
		{"missing confirmed_by", `"confirmed_by": "restored in 1234abcd"`, `"note": "x"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(validReplay, tc.from) {
				t.Fatalf("mutation anchor %q is not in the fixture", tc.from)
			}
			doc := strings.Replace(validReplay, tc.from, tc.to, 1)
			if err := ValidateCase([]byte(doc)); err == nil {
				t.Fatalf("the schema accepted a case with %s", tc.name)
			}
		})
	}
}
