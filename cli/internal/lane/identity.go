package lane

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LaneIdentityFile is the record `lane init` writes inside a lane's git admin
// dir (`<repo>/.git/worktrees/<id>/`): the lane's name and nothing else. Git
// owns that directory, so the record moves with `git worktree move`, survives
// `git gc` and `git clean -fdx`, and disappears with `git worktree remove` or
// `prune`. The pre-commit guard reads the same file by the same name.
const LaneIdentityFile = "dotf-lane"

// Identity answers whether path is a persistent named lane and, if so, which
// one. It reports ("", false, nil) for a worktree without a record, for a main
// checkout (which has no worktree admin dir) and for a path outside any
// repository: none of those is an error, they are simply not lanes.
func Identity(path string) (string, bool, error) {
	gitDir, commonDir, err := worktreeGitDirs(path)
	if err != nil {
		return "", false, err
	}
	// A main checkout's git dir is the common dir, which is shared by every
	// worktree and where `dotf lane next` keeps its prompt DIRECTORY under
	// this same name. Only a linked worktree's admin dir can hold the record.
	if gitDir == "" || gitDir == commonDir {
		return "", false, nil
	}
	record := filepath.Join(gitDir, LaneIdentityFile)
	if st, serr := os.Stat(record); serr != nil || st.IsDir() {
		return "", false, nil
	}
	raw, err := os.ReadFile(record)
	if err != nil {
		return "", false, fmt.Errorf("lane identity: reading %s: %w", filepath.Join(gitDir, LaneIdentityFile), err)
	}
	// The first line is the name, as the guard reads it; anything after it is
	// residue no writer of ours produces.
	first, _, _ := strings.Cut(string(raw), "\n")
	name := strings.TrimSpace(first)
	if name == "" {
		return "", false, nil
	}
	return name, true, nil
}

// writeIdentity records name as the lane identity of the worktree at path.
// It writes only when the record is absent or differs, and reports whether it
// wrote, so `lane init` can answer changed=0 on a re-run. A main checkout is
// refused: its git dir is the common dir, shared by every worktree, so a record
// there would name every worktree at once.
func writeIdentity(path, name string) (bool, error) {
	gitDir, commonDir, err := worktreeGitDirs(path)
	if err != nil {
		return false, err
	}
	if gitDir == "" {
		return false, fmt.Errorf("lane identity: %s is not inside a git repository", path)
	}
	if gitDir == commonDir {
		return false, fmt.Errorf("lane identity: %s is a main checkout, not a linked worktree", path)
	}
	record := filepath.Join(gitDir, LaneIdentityFile)
	want := []byte(name + "\n")
	if have, err := os.ReadFile(record); err == nil && bytes.Equal(have, want) {
		return false, nil
	}
	if err := os.WriteFile(record, want, 0o644); err != nil {
		return false, fmt.Errorf("lane identity: writing %s: %w", record, err)
	}
	return true, nil
}

// worktreeGitDirs resolves the git dir of the worktree containing path and the
// repository's common dir, both absolute. A path outside any repository
// answers two empty strings and no error; only git itself failing to run is an
// error.
func worktreeGitDirs(path string) (gitDir, commonDir string, err error) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("lane identity: running git in %s: %w", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[0] == "" || lines[1] == "" {
		return "", "", fmt.Errorf("lane identity: unexpected git rev-parse output for %s: %q", path, string(out))
	}
	return filepath.Clean(lines[0]), filepath.Clean(lines[1]), nil
}
