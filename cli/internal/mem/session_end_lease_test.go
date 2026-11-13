package mem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// The lease lives in the worktree's git admin dir (ADR-062): session_end
// reads it there first, from any depth inside the worktree, and falls back to
// the in-tree file only when the admin dir holds none.
func TestIsWorktreeParkedReadsTheAdminDirLease(t *testing.T) {
	tmp := t.TempDir()
	wt := filepath.Join(tmp, "wt")
	admin, err := gittest.LinkWorktree(filepath.Join(tmp, "repo"), wt)
	if err != nil {
		t.Fatal(err)
	}
	adminLease := filepath.Join(admin, "dotf-worktree.json")
	treeLease := filepath.Join(wt, ".dotf-worktree.json")
	sub := filepath.Join(wt, "cli", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, state string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(`{"lane":"feat-lane-primary","state":"`+state+`","reap_ok":false}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if isWorktreeParked(wt) {
		t.Error("no lease anywhere read as parked")
	}
	write(adminLease, "PARKED")
	if !isWorktreeParked(wt) || !isWorktreeParked(sub) {
		t.Error("a PARKED lease in the admin dir alone was not read, from the top or from a subdirectory")
	}
	write(adminLease, "ACTIVE")
	write(treeLease, "PARKED")
	if isWorktreeParked(wt) {
		t.Error("the in-tree file outranked the admin-dir lease")
	}
	if err := os.Remove(adminLease); err != nil {
		t.Fatal(err)
	}
	if !isWorktreeParked(sub) {
		t.Error("a legacy in-tree lease was not read once the admin dir held none")
	}
}
