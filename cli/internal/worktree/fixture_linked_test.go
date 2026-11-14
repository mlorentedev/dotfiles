package worktree

import (
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// fakeAdd stands in for `git worktree add` in a MockAddRunner stub: the
// target gets the .git pointer and admin dir git would leave, so the lease
// SaveMetadata writes has somewhere to live (ADR-062).
func fakeAdd(repo, target string) error {
	_, err := gittest.LinkWorktree(repo, target)
	return err
}

// linkFake makes an already-planned fixture directory a linked worktree of
// repo; the fixtures that drive list and sweep through a mock porcelain use it
// where they used to make a bare directory.
func linkFake(t *testing.T, repo, wt string) {
	t.Helper()
	if err := fakeAdd(repo, wt); err != nil {
		t.Fatal(err)
	}
}

// linkedTempDir is t.TempDir() for a fixture that stands for a worktree.
func linkedTempDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	wt := filepath.Join(tmp, "wt")
	linkFake(t, filepath.Join(tmp, "repo"), wt)
	return wt
}
