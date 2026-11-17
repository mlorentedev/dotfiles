package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

func TestHarnessGateStickyPersonaAndMeeseekResolution(t *testing.T) {
	root := repoRootForTest(t)

	// 1. DOTF_WORKER_ID=meeseek-reviewer-avocado resolves to reviewer persona
	t.Setenv("DOTF_WORKER_ID", "meeseek-reviewer-avocado")
	gotRole := effectiveRole("", "")
	if gotRole != "meeseek-reviewer-avocado" {
		t.Fatalf("effectiveRole with DOTF_WORKER_ID = %q, want meeseek-reviewer-avocado", gotRole)
	}

	var warn bytes.Buffer
	p, res := loadGatePersona(&warn, root, gotRole, nil, false)
	if res != roleResolved || p == nil || p.Name != "reviewer" {
		t.Fatalf("loadGatePersona(%q) = (%v, %v), want reviewer, roleResolved (stderr: %s)", gotRole, p, res, warn.String())
	}

	// 2. Sticky persona bound in stateDir is used when payload and env carry no role
	t.Setenv("DOTF_WORKER_ID", "")
	stateDir := t.TempDir()
	pPath := harness.PersonaStatePath(stateDir, "sess-sticky")
	if err := harness.BindStickyPersona(pPath, "builder", ""); err != nil {
		t.Fatalf("BindStickyPersona: %v", err)
	}

	stickyRole := effectiveRoleWithSticky("", "", stateDir, "sess-sticky", "")
	if stickyRole != "builder" {
		t.Fatalf("effectiveRoleWithSticky = %q, want builder", stickyRole)
	}
}

// A plain checkout carries no .dotf-worktree.json, and the sticky persona still
// has to be invalidated when the operator switches branch there. Reading the
// branch only inside the worktree-metadata branch left it empty, which made
// LoadStickyPersona blind to the switch (#HARNESS-186 review, Major).
func TestCurrentGateWorktreeContextReadsBranchOutsideWorktreeMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	head := filepath.Join(dir, ".git", "HEAD")
	if err := os.WriteFile(head, []byte("ref: refs/heads/feat-lane-primary\n"), 0o600); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}

	branch, wtPersona := currentGateWorktreeContext(dir)
	if branch != "feat-lane-primary" {
		t.Fatalf("branch = %q, want feat-lane-primary (no worktree metadata present)", branch)
	}
	if wtPersona != "" {
		t.Fatalf("wtPersona = %q, want empty (no worktree metadata present)", wtPersona)
	}
}
