package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func digestOf(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return ContractDigests(dir)
}

func TestContractDigestCoversEveryContractFile(t *testing.T) {
	got := digestOf(t, map[string]string{"proposal.md": "p\n", "tasks.md": "t\n"})
	if len(got) != len(contractFiles) {
		t.Fatalf("digests = %v, want one entry per contract file %v", got, contractFiles)
	}
	if got["proposal.md"] == "" || got["tasks.md"] == "" {
		t.Errorf("present files must have a digest: %v", got)
	}
	if got["features.json"] != "" {
		t.Errorf("an absent file digests to \"\", got %q", got["features.json"])
	}
}

// Ticking a box is progress, not a new criterion (#998 part 2): a review whose
// own finding was "tick the boxes" must not invalidate itself when applied.
func TestContractDigestIgnoresCheckboxTicks(t *testing.T) {
	before := "## Acceptance criteria\n\n- [ ] **AC1** — refuses\n  - [ ] nested step\n* [ ] star item\n"
	after := "## Acceptance criteria\n\n- [x] **AC1** — refuses\n  - [X] nested step\n* [x] star item\n"
	a := digestOf(t, map[string]string{"proposal.md": before, "tasks.md": before})
	b := digestOf(t, map[string]string{"proposal.md": after, "tasks.md": after})
	if a["proposal.md"] != b["proposal.md"] || a["tasks.md"] != b["tasks.md"] {
		t.Fatal("ticking checkboxes changed the contract digest")
	}
}

// Ordered-list checkboxes are bookkeeping too: 9 occurrences across specs/
// on 2026-09-23 (SDD-042 review, finding 1). Ticking one must not read as
// contract drift and refuse an archive.
func TestContractDigestIgnoresOrderedListTicks(t *testing.T) {
	a := digestOf(t, map[string]string{"tasks.md": "1. [ ] first\n2) [ ] second\n  10. [ ] nested\n"})
	b := digestOf(t, map[string]string{"tasks.md": "1. [x] first\n2) [X] second\n  10. [x] nested\n"})
	if a["tasks.md"] != b["tasks.md"] {
		t.Fatal("ticking an ordered-list checkbox changed the contract digest")
	}
	c := digestOf(t, map[string]string{"tasks.md": "1. [ ] first, reworded\n2) [ ] second\n  10. [ ] nested\n"})
	if a["tasks.md"] == c["tasks.md"] {
		t.Fatal("rewording an ordered-list task must still change the digest")
	}
}

// The fold is for list checkboxes only: a bracketed x in running text is text.
func TestContractDigestKeepsBracketsInProse(t *testing.T) {
	a := digestOf(t, map[string]string{"proposal.md": "the flag [ ] means off\n"})
	b := digestOf(t, map[string]string{"proposal.md": "the flag [x] means off\n"})
	if a["proposal.md"] == b["proposal.md"] {
		t.Fatal("a bracket in prose is not a checkbox and must stay significant")
	}
}

func TestContractDigestSeesCriterionText(t *testing.T) {
	a := digestOf(t, map[string]string{"proposal.md": "- [ ] **AC1** — refuses a closed issue\n"})
	b := digestOf(t, map[string]string{"proposal.md": "- [x] **AC1** — warns on a closed issue\n"})
	if a["proposal.md"] == b["proposal.md"] {
		t.Fatal("rewording an acceptance criterion must change the digest")
	}
}

func TestContractDigestIgnoresCRLF(t *testing.T) {
	a := digestOf(t, map[string]string{"proposal.md": "line one\nline two\n"})
	b := digestOf(t, map[string]string{"proposal.md": "line one\r\nline two\r\n"})
	if a["proposal.md"] != b["proposal.md"] {
		t.Fatal("a Windows checkout's line endings changed the digest")
	}
}

// state and evidence are written by the harness after the review; the rest of
// a feature is the contract the reviewer read.
func TestContractDigestIgnoresHarnessFeatureFields(t *testing.T) {
	pending := `[{"id":"f1","behavior":"refuses","verification":"go test","state":"pending","evidence":""}]`
	passing := `[
  {"id":"f1","behavior":"refuses","verification":"go test","state":"passing","evidence":"ok 0.1s"}
]`
	changed := `[{"id":"f1","behavior":"warns","verification":"go test","state":"pending","evidence":""}]`
	a := digestOf(t, map[string]string{"features.json": pending})
	b := digestOf(t, map[string]string{"features.json": passing})
	c := digestOf(t, map[string]string{"features.json": changed})
	if a["features.json"] != b["features.json"] {
		t.Error("the harness recording state/evidence changed the contract digest")
	}
	if a["features.json"] == c["features.json"] {
		t.Error("changing a feature's behavior must change the digest")
	}
}

func TestContractDigestMalformedFeaturesStillDigests(t *testing.T) {
	a := digestOf(t, map[string]string{"features.json": "{not json"})
	b := digestOf(t, map[string]string{"features.json": "{not json either"})
	if a["features.json"] == "" || a["features.json"] == b["features.json"] {
		t.Fatalf("malformed features.json must still digest by its bytes: %q vs %q", a["features.json"], b["features.json"])
	}
}
