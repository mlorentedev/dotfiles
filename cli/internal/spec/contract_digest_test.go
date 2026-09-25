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

// HARNESS-151: `dotf spec archive` writes `status: archived` AFTER the freshness
// check passes, so if the digest read that line, every archived proposal would
// stop matching the review that permitted its archive, and a later reader could
// not re-verify it (CodeRabbit flagged HARNESS-145's archive PR as stale for
// exactly this). The frontmatter status is lifecycle, not contract: it folds
// like a checkbox tick, through the same setStatus the archive uses.
func TestContractDigestIgnoresTheLifecycleStatus(t *testing.T) {
	proposal := "---\nid: \"X-001-y\"\ntype: spec\nstatus: implementing # draft | implementing | verifying | archived\n---\n\n# X\n\n- [ ] **AC1** — refuses\n"
	before := digestOf(t, map[string]string{"proposal.md": proposal})
	for _, status := range []string{"archived", "abandoned", "verifying"} {
		after := digestOf(t, map[string]string{"proposal.md": setStatus(proposal, status)})
		if before["proposal.md"] != after["proposal.md"] {
			t.Errorf("setting status %q changed the contract digest", status)
		}
	}
}

// The fold is the frontmatter's status line only: a `status:` line in the body
// is prose, and changing it is a contract change like any other.
func TestContractDigestSeesAStatusLineInTheBody(t *testing.T) {
	fm := "---\nid: \"X-001-y\"\nstatus: draft\n---\n\n"
	a := digestOf(t, map[string]string{"proposal.md": fm + "status: the endpoint returns 200\n"})
	b := digestOf(t, map[string]string{"proposal.md": fm + "status: the endpoint returns 404\n"})
	if a["proposal.md"] == b["proposal.md"] {
		t.Error("a status line in the body was folded away")
	}
}

// A review launched before HARNESS-151 recorded its digests with the status line
// in them. An unchanged spec must stay fresh after the upgrade instead of
// demanding a re-review, a review recorded in the new form must survive the
// archive's own status write, and a real edit must still stale either.
func TestReviewFreshnessAcceptsBothDigestForms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proposal.md")
	proposal := "---\nid: \"X-001-y\"\nstatus: implementing\n---\n\n- [ ] **AC1** — refuses\n"
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(proposal)
	legacy, current := legacyContractDigests(dir), ContractDigests(dir)
	if moved := changedContracts(dir, legacy); len(moved) != 0 {
		t.Errorf("a review recorded in the legacy form went stale on an unchanged spec: %v", moved)
	}

	write(setStatus(proposal, "archived"))
	if moved := changedContracts(dir, current); len(moved) != 0 {
		t.Errorf("the archive's own status write staled a review recorded in the new form: %v", moved)
	}

	write(setStatus(proposal, "archived") + "- [ ] **AC2** — a criterion added after the review\n")
	for form, recorded := range map[string]map[string]string{"legacy": legacy, "new": current} {
		if moved := changedContracts(dir, recorded); len(moved) != 1 || moved[0] != "proposal.md" {
			t.Errorf("a new criterion must stale a review recorded in the %s form, got %v", form, moved)
		}
	}
}
