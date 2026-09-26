package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func requiresFixture(t *testing.T, manifest string, onPath []string) (string, int) {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "pi", "packages.json"), manifest)
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, onPath, nil)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkPiPackageRequirements(sys, &Config{DotfilesDir: t.TempDir()}, rep)
	return buf.String(), rep.Failures()
}

// AC6: pi-memory's state, a declared package whose retrieval dependency was
// never provisioned, is a FAIL naming the package and what it needs.
func TestPiPackageRequirements_AMissingRequirementFails(t *testing.T) {
	out, fails := requiresFixture(t, `{"packages":[{"source":"npm:pi-memory@0.4.2","why":"x","requires":["qmd"]}]}`, nil)
	if fails != 1 || !strings.Contains(out, "npm:pi-memory@0.4.2 needs qmd") {
		t.Fatalf("want one FAIL naming the package and its requirement:\n%s", out)
	}
}

func TestPiPackageRequirements_AResolvedRequirementPasses(t *testing.T) {
	out, fails := requiresFixture(t, `{"packages":[{"source":"npm:m@1","why":"x","requires":["qmd"]}]}`, []string{"qmd"})
	if fails != 0 || !strings.Contains(out, "(1 checked)") {
		t.Fatalf("want a pass counting the requirement:\n%s", out)
	}
}

// AC6: the shipped manifest passes on any machine, because no package it
// declares after the purge requires anything.
func TestPiPackageRequirements_TheShippedManifestPasses(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkPiPackageRequirements(sys, &Config{DotfilesDir: t.TempDir()}, rep)
	if rep.Failures() != 0 {
		t.Fatalf("the shipped manifest must pass:\n%s", buf.String())
	}
}

func TestPiPackageRequirements_AnUnreadableManifestWarnsNotPasses(t *testing.T) {
	out, fails := requiresFixture(t, `{"packages":[`, nil)
	if fails != 0 || !strings.Contains(out, "unreadable") {
		t.Fatalf("want a WARN, never a pass:\n%s", out)
	}
}
