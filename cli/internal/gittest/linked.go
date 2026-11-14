package gittest

import (
	"fmt"
	"os"
	"path/filepath"
)

// LinkWorktree makes dir look like a linked worktree of repo to any code that
// reads the `.git` pointer file, without running git: it creates the admin dir
// `<repo>/.git/worktrees/<base(dir)>/` and writes `<dir>/.git` pointing at it,
// which is what `git worktree add` leaves behind. Fixtures that fake worktrees
// with a mock runner use it so the lease has the admin dir it lives in
// (ADR-062); a fixture that only makes the directory is a plain directory,
// and `worktree.SaveMetadata` refuses those. It returns the admin dir.
func LinkWorktree(repo, dir string) (string, error) {
	admin := filepath.Join(repo, ".git", "worktrees", filepath.Base(dir))
	if err := os.MkdirAll(admin, 0o755); err != nil {
		return "", fmt.Errorf("gittest: creating the admin dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("gittest: creating the worktree: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: "+admin+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("gittest: writing the .git pointer: %w", err)
	}
	return admin, nil
}
