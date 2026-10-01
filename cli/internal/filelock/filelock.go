// Package filelock is an exclusive, cross-process lock on a file that the
// kernel releases when its holder exits: flock on Unix, an unshared CreateFile
// handle on Windows.
//
// Kernel release is the point. A lock file created with O_EXCL and deleted on
// unlock survives a writer killed in between, and every later writer then waits
// on a lock nobody holds until a human deletes it (#1884).
package filelock

import (
	"errors"
	"fmt"
	"time"
)

// ErrLocked reports that another open of the path holds the lock.
var ErrLocked = errors.New("held by another process")

// pollInterval is how often Lock retries. Writers this guards hold the lock for
// milliseconds, so a short poll costs little and keeps the wait short.
const pollInterval = 20 * time.Millisecond

// Lock takes the lock on path, waiting up to timeout for its holder to finish.
// The returned func releases it. A timeout is an error naming the path, so a
// lock that never frees is reported rather than waited on forever.
func Lock(path string, timeout time.Duration) (func(), error) {
	deadline := time.Now().Add(timeout)
	for {
		unlock, err := TryLock(path)
		if err == nil {
			return unlock, nil
		}
		if !errors.Is(err, ErrLocked) {
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("lock %s: %w after waiting %s", path, ErrLocked, timeout)
		}
		time.Sleep(pollInterval)
	}
}
