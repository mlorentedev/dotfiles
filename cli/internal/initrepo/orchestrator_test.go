package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitOrchestratesFullScaffold(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myproj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := Init(InitOptions{
		Root:       root,
		Stack:      "go",
		Date:       "2026-06-14",
		SkipGithub: true, // no remote on a fresh repo anyway
		// VaultPath "" => vault step skips
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	wantFiles := []string{
		"AGENTS.md", "CLAUDE.md", ".gitignore", ".pre-commit-config.yaml",
		"env-contract.json", filepath.Join("docs", "lessons.md"),
		filepath.Join(".github", "workflows", "ci.yml"),
		"go.mod", "Makefile", ".git",
	}
	for _, f := range wantFiles {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("expected %s after Init: %v", f, err)
		}
	}

	agents := readFile(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(agents, "## Spec-Driven Development") {
		t.Error("AGENTS.md should carry the SDD section")
	}
	if strings.Contains(agents, "$VAULT_PATH") {
		t.Error("AGENTS.md leaks $VAULT_PATH")
	}

	if len(report.Steps) == 0 {
		t.Error("report should list the steps taken")
	}
	// The vault step skips (no VaultPath given).
	if s := stepStatus(report, "vault"); s != "skipped" {
		t.Errorf("vault step status = %q, want skipped", s)
	}
}

func TestInitSkipAgents(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(InitOptions{Root: root, Stack: "none", SkipAgents: true, SkipGithub: true, Date: "2026-06-14"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("--skip-agents should not write AGENTS.md")
	}
	// Structure still scaffolded.
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err != nil {
		t.Error("structure should still be scaffolded with --skip-agents")
	}
}

// TestInitWritesVaultEntryWhenVaultPresent is the parity oracle for #395: with a
// vault present, the orchestrator must drive vault.WriteProjectEntry and produce
// the full 10_projects/<repo>/ entry — the same output the inlined
// initrepo.WriteVaultEntry produced before the extraction.
func TestInitWritesVaultEntryWhenVaultPresent(t *testing.T) {
	vaultDir := t.TempDir()
	root := filepath.Join(t.TempDir(), "myproj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := Init(InitOptions{
		Root:       root,
		Stack:      "go",
		Date:       "2026-06-14",
		SkipGithub: true,
		VaultPath:  vaultDir,
		// ClaudeProjectsDir "" => no symlink attempt (keeps HOME untouched).
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if s := stepStatus(report, "vault"); s != "ok" {
		t.Errorf("vault step status = %q, want ok", s)
	}

	entry := filepath.Join(vaultDir, "10_projects", "myproj")
	for _, f := range []string{"context.md", "roadmap.md", filepath.Join("memory", "MEMORY.md")} {
		if _, err := os.Stat(filepath.Join(entry, f)); err != nil {
			t.Errorf("expected vault entry file %s after Init: %v", f, err)
		}
	}
}

func stepStatus(r InitReport, name string) string {
	for _, s := range r.Steps {
		if s.Name == name {
			return s.Status
		}
	}
	return ""
}
