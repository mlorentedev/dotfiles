package worktree

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

var ErrLocked = errors.New("worktree operation already in progress by another session")

func DefaultLockPath() string {
	return filepath.Join(os.TempDir(), "dotf-worktree.lock")
}

// TryLockFile takes the worktree lock without waiting, through filelock, and
// reports a held lock as this package's ErrLocked.
func TryLockFile(path string) (func(), error) {
	unlock, err := filelock.TryLock(path)
	if errors.Is(err, filelock.ErrLocked) {
		return nil, ErrLocked
	}
	return unlock, err
}
