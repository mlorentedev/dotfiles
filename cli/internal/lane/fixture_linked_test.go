package lane

import (
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// linkedTempDir is t.TempDir() for a fixture that stands for a worktree the
// fake backend created: it carries the .git pointer and admin dir git would
// leave, so the lease `lane next` writes has somewhere to live (ADR-062).
func linkedTempDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	wt := filepath.Join(tmp, "wt")
	if _, err := gittest.LinkWorktree(filepath.Join(tmp, "repo"), wt); err != nil {
		t.Fatal(err)
	}
	return wt
}
