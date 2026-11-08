package cmd

import (
	"bytes"
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
