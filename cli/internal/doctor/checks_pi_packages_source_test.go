package doctor

import (
	"path/filepath"
	"testing"
)

// The pi-packages count reads the checkout first (ADR-030 precedence). The
// deploy mirror never carried ai/pi/, so on Windows the shadow check reported
// "0 packages declared" as a PASS — a wrong number under a green tag
// (WIN-007/#1288).
func TestPiPackagesManifest_ReadsTheCheckoutBeforeTheMirror(t *testing.T) {
	repo, mirror := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "pi", "packages.json"), `{"packages":[{"name":"a"},{"name":"b"}]}`)
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)
	cfg := &Config{DotfilesDir: mirror}

	if got := piPackagesManifest(sys, cfg); got != 2 {
		t.Errorf("checkout declares 2 packages, got %d", got)
	}

	// A checkout without the file: the mirror copy is the fallback, not zero.
	// (An unresolvable DOTFILES_REPO_DIR would walk up from the test cwd into
	// the real checkout, so the fixture uses an empty one.)
	writeFile(t, filepath.Join(mirror, "ai", "pi", "packages.json"), `{"packages":[{"name":"a"}]}`)
	sys = newSys(map[string]string{"DOTFILES_REPO_DIR": t.TempDir()}, nil, nil)
	if got := piPackagesManifest(sys, cfg); got != 1 {
		t.Errorf("mirror fallback: want 1, got %d", got)
	}
}
