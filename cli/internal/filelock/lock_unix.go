//go:build !windows

package filelock

import (
	"errors"
	"os"
	"syscall"
)

// TryLock takes the lock on path without waiting, or returns ErrLocked. Locks
// belong to the open file description, so two opens in one process contend
// like two processes do.
func TryLock(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) // #nosec G304 -- caller-chosen lock path
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		// EWOULDBLOCK (== EAGAIN on Linux) is "someone else holds it". Anything
		// else, such as a filesystem that cannot lock, is a real failure.
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
