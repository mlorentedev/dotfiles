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
