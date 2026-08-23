//go:build !windows

package agent

import (
	"errors"
	"os"
	"syscall"
)

// tryTakeSlot opens the slot file and takes an exclusive, non-blocking lock on
// it. It reports (nil, false, nil) when another process holds the slot — a busy
// slot is an expected answer, not an error — and a non-nil error only when the
// attempt itself could not be made.
//
// flock is used rather than the file's existence because the kernel releases it
// when the holder dies. A `dotf` killed with SIGKILL therefore frees its slot,
// where an existence-based marker would leak one until something reaped it.
func tryTakeSlot(path string) (*os.File, bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return f, true, nil
	}
	_ = f.Close()
	// EWOULDBLOCK (== EAGAIN on Linux) is "someone else holds it". Anything
	// else — EBADF, ENOLCK, a filesystem that cannot lock — is a real failure
	// and must not be read as a free slot.
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return nil, false, nil
	}
	return nil, false, err
}

// releaseSlot drops the lock. Closing the descriptor would also do it; the
// explicit unlock states the intent, and both are best-effort because a failure
// here cannot be actioned — the kernel releases the lock at process exit
// regardless.
func releaseSlot(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
