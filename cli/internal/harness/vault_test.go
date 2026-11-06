package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vaultPair is a checkout with one skill record and one agent record, each
// stamped from a vault source, and the vault they were refreshed from.
func vaultPair(t *testing.T) (repo, vault string) {
	t.Helper()
	repo, vault = t.TempDir(), t.TempDir()
	skill := "---\nname: demo\n---\nbody\n"
	agent := "---\nname: builder\n---\nmandate\n"
	writeFile(t, filepath.Join(vault, "00_meta", "skills", "demo", "SKILL.md"), skill)
	writeFile(t, filepath.Join(vault, "00_meta", "agents", "builder", "AGENT.md"), agent)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "SKILL.md"),
		"---\ngenerated: true\ngenerated_from: 00_meta/skills/demo/SKILL.md\ngenerated_sha: "+sha16(skill)+"\nname: demo\n---\nbody\n")
	writeFile(t, filepath.Join(repo, "harness", "agents", "builder", "AGENT.md"),
		"---\ngenerated: true\ngenerated_from: 00_meta/agents/builder/AGENT.md\ngenerated_sha: "+sha16(agent)+"\nname: builder\n---\nmandate\n")
	return repo, vault
}

func TestCompareWithVault_ARefreshedCheckoutMatches(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", ".DS_Store"), "finder\n")
	d := CompareWithVault(repo, vault)
	if d.Checked != 2 || d.HasDrift() {
		t.Fatalf("want 2 records level with the vault, got %+v", d)
	}
}

// AC6 (#1545): a vault source edited and never refreshed is drift.
func TestCompareWithVault_AnUnrefreshedVaultEditIsStale(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(vault, "00_meta", "skills", "demo", "SKILL.md"), "---\nname: demo\n---\nbody, edited\n")
	d := CompareWithVault(repo, vault)
	if !d.HasDrift() || strings.Join(d.Stale, ",") != "skills/demo" {
		t.Fatalf("want skills/demo stale, got %+v", d)
	}
	if lines := d.Lines(); !HasDrift(lines) || !strings.Contains(lines[0].Text, "skills/demo") {
		t.Errorf("want a problem line naming the record, got %+v", lines)
	}
}

func TestCompareWithVault_AMissingSourceAndAnUnstampedRecordAreDrift(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "skills", "plain", "SKILL.md"), "---\nname: plain\n---\n")
	if err := os.RemoveAll(filepath.Join(vault, "00_meta", "agents")); err != nil {
		t.Fatal(err)
	}
	d := CompareWithVault(repo, vault)
	if strings.Join(d.Stale, ",") != "agents/builder (source gone)" || strings.Join(d.Unstamped, ",") != "skills/plain" {
		t.Fatalf("got %+v", d)
	}
}

// #1545: a record edited in the checkout while its vault source stayed put
// carries the right sha, so only rendering the source finds the edit.
func TestCompareWithVault_ARecordEditedWithoutItsSourceIsEdited(t *testing.T) {
	repo, vault := vaultPair(t)
	rec := filepath.Join(repo, "harness", "skills", "demo", "SKILL.md")
	b, err := os.ReadFile(rec)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, rec, string(b)+"an edit made only here\n")
	writeFile(t, filepath.Join(repo, "harness", "agents", "builder", "notes.md"), "a file the vault does not have\n")
	d := CompareWithVault(repo, vault)
	if strings.Join(d.Edited, ",") != "agents/builder,skills/demo" || len(d.Stale) != 0 {
		t.Fatalf("want both records edited and none stale, got %+v", d)
	}
	if lines := d.Lines(); !HasDrift(lines) || !strings.Contains(lines[0].Text, "[EDITED] harness/agents/builder") {
		t.Errorf("want an EDITED line naming the record, got %+v", lines)
	}
}

// A vault checked out with CRLF renders the same records: line endings are not an edit.
func TestCompareWithVault_ACRLFVaultIsNotAnEdit(t *testing.T) {
	repo, vault := t.TempDir(), t.TempDir()
	src := "---\r\nname: demo\r\n---\r\nbody\r\n"
	writeFile(t, filepath.Join(vault, "00_meta", "skills", "demo", "SKILL.md"), src)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "SKILL.md"),
		"---\ngenerated: true\ngenerated_from: 00_meta/skills/demo/SKILL.md\ngenerated_sha: "+sha16(src)+"\nname: demo\n---\nbody\n")
	if d := CompareWithVault(repo, vault); d.HasDrift() {
		t.Fatalf("want no drift, got %+v", d)
	}
}

// #3098: a vault record a refresh would write, absent from the checkout, is
// drift; one the manifest excludes is not, and refresh skips it too.
func TestCompareWithVault_AVaultRecordNeverRefreshedIsMissingUnlessExcluded(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "agent-frontmatter.schema.json"), `{"required":["name"]}`)
	writeFile(t, filepath.Join(repo, "harness", "skill-frontmatter.schema.json"), `{"required":["name"]}`)
	manifest := func(exclude string) {
		writeFile(t, filepath.Join(repo, filepath.FromSlash(ManifestFile)), `{"targets":[],
			"skills":{"vault_subpath":"00_meta/skills","record_dir":"harness/skills"},
			"agents":{"vault_subpath":"00_meta/agents","record_dir":"harness/agents"`+exclude+`}}`)
	}
	manifest("")
	writeFile(t, filepath.Join(vault, "00_meta", "agents", "newcomer", "AGENT.md"), "---\nname: newcomer\n---\n")
	writeFile(t, filepath.Join(vault, "00_meta", "agents", "notes", "README.md"), "not a record\n")

	d := CompareWithVault(repo, vault)
	if strings.Join(d.Missing, ",") != "agents/newcomer" {
		t.Fatalf("want agents/newcomer missing, got %+v", d)
	}
	if lines := d.Lines(); !HasDrift(lines) || !strings.Contains(lines[0].Text, "[MISSING] harness/agents/newcomer") {
		t.Errorf("want a MISSING line naming the record, got %+v", lines)
	}

	manifest(`,"exclude":{"newcomer":"decided in kubelab#1933"}`)
	if d := CompareWithVault(repo, vault); d.HasDrift() {
		t.Fatalf("want an excluded record not missing, got %+v", d)
	}
	var w refreshWrites
	m, err := loadCheckManifest(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.records(repo, vault, m.Agents, "AGENT.md"); err != nil {
		t.Fatal(err)
	}
	for _, op := range w.ops {
		if strings.Contains(op.dst, "newcomer") {
			t.Errorf("refresh planned an excluded record: %+v", op)
		}
	}

	// An excluded record still committed is one the next refresh drops.
	writeFile(t, filepath.Join(repo, "harness", "agents", "newcomer", "AGENT.md"), "---\nname: newcomer\n---\n")
	d = CompareWithVault(repo, vault)
	if strings.Join(d.Excluded, ",") != "agents/newcomer" {
		t.Fatalf("want the committed excluded record reported, got %+v", d)
	}
	if lines := d.Lines(); !strings.Contains(lines[0].Text, "[EXCLUDED] harness/agents/newcomer") {
		t.Errorf("want an EXCLUDED line, got %+v", lines)
	}
}

// An exclusion whose vault dir is gone has outlived its reason.
func TestCompareWithVault_AnExclusionTheVaultNoLongerHasIsUnused(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "agent-frontmatter.schema.json"), `{"required":["name"]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(ManifestFile)), `{"targets":[],
		"agents":{"vault_subpath":"00_meta/agents","record_dir":"harness/agents","exclude":{"gone":"was decided elsewhere"}}}`)
	if d := CompareWithVault(repo, vault); strings.Join(d.Unused, ",") != "agents/gone" {
		t.Fatalf("want agents/gone unused, got %+v", d)
	}
}

// A vault record a refresh would refuse is not MISSING: committing cannot clear it.
func TestCompareWithVault_AVaultRecordRefreshRefusesIsInvalidNotMissing(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "agent-frontmatter.schema.json"), `{"required":["name"]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(ManifestFile)), `{"targets":[],
		"agents":{"vault_subpath":"00_meta/agents","record_dir":"harness/agents"}}`)
	writeFile(t, filepath.Join(vault, "00_meta", "agents", "nameless", "AGENT.md"), "---\ndescription: x\n---\n")
	d := CompareWithVault(repo, vault)
	if len(d.Missing) != 0 || len(d.Invalid) != 1 || !strings.Contains(d.Invalid[0], `"name"`) {
		t.Fatalf("want one invalid record naming the key and none missing, got %+v", d)
	}
}

// An exclusion carries its reason, like an enforced opt_out.
func TestRecords_AnExclusionWithoutAReasonIsRefused(t *testing.T) {
	repo, vault := vaultPair(t)
	writeFile(t, filepath.Join(repo, "harness", "agent-frontmatter.schema.json"), `{"required":["name"]}`)
	k := &recordKind{VaultSubpath: "00_meta/agents", RecordDir: "harness/agents", Exclude: map[string]string{"builder": " "}}
	var w refreshWrites
	if err := w.records(repo, vault, k, "AGENT.md"); err == nil || !strings.Contains(err.Error(), "builder") {
		t.Fatalf("want a refusal naming builder, got %v", err)
	}
}

// Enforced records carry no stamp: each is held to the vault section it is cut from.
func TestCompareWithVault_EnforcedRecordsAreHeldToTheirSection(t *testing.T) {
	repo, vault := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, filepath.FromSlash(ManifestFile)),
		`{"vault_subpath":"00_meta","enforced":[{"id":"rule","source":"patterns/p.md#the-rule"},{"id":"gone","source":"patterns/p.md#no-such-section"}],"targets":[]}`)
	pattern := filepath.Join(vault, "00_meta", "patterns", "p.md")
	writeFile(t, pattern, "# P\n\n## The Rule\n\nDo the thing.\n\n## Other\n\nnot this\n")
	want, err := sectionOf(pattern, "the-rule")
	if err != nil {
		t.Fatal(err)
	}
	// The record is cut from an LF checkout; the vault below is CRLF and cuts the same section.
	writeFile(t, pattern, "# P\r\n\r\n## The Rule\r\n\r\nDo the thing.\r\n\r\n## Other\r\n\r\nnot this\r\n")
	rec := filepath.Join(repo, filepath.FromSlash(EnforcedDir), "rule.md")
	writeFile(t, rec, want)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(EnforcedDir), "gone.md"), "whatever\n")

	d := CompareWithVault(repo, vault)
	if d.Checked != 2 || len(d.Edited) != 0 || strings.Join(d.Stale, ",") != "enforced/gone (source gone)" {
		t.Fatalf("want rule level and gone stale, got %+v", d)
	}
	writeFile(t, rec, want+"an edit made only here\n")
	if d := CompareWithVault(repo, vault); strings.Join(d.Edited, ",") != "enforced/rule" {
		t.Fatalf("want enforced/rule edited, got %+v", d)
	}
}
