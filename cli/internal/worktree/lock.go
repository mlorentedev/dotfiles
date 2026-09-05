package worktree

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrLocked = errors.New("worktree operation already in progress by another session")

func DefaultLockPath() string {
	return filepath.Join(os.TempDir(), "dotf-worktree.lock")
}
