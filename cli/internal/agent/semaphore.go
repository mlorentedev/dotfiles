package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Semaphore bounds how many dispatches `dotf` runs against one pool at once.
//
// Read the guarantee narrowly, because the honest one is narrow: a hand-run
// `qq`, a pi TUI turn or a hive embedding call consumes a slot this never sees
// (ADR-032 §3). What this can promise is that **`dotf` alone will never be the
// cause of exhaustion** — not that exhaustion cannot happen.
//
// State is one lock file per slot under a per-pool directory. The lock is held
// by an OS file lock rather than by the file's existence, because the two
// differ in exactly the case that matters: the kernel drops a file lock when
// the holding process dies, so a `dotf` killed with SIGKILL frees its slot,
// while an existence-based marker would leak it until something reaped it. That
// same property is what makes release-before-reap trivial — closing our own
// descriptor frees the slot without touching the worker.
type Semaphore struct {
	dir string
}

// NewSemaphore returns a semaphore whose state lives under dir.
func NewSemaphore(dir string) *Semaphore { return &Semaphore{dir: dir} }

// Slot is one held unit of a pool's capacity. Release is safe to call more than
// once: the timeout path releases early and the normal path releases again on
// the way out.
type Slot struct {
	f      *os.File
	closed bool
}

// Release frees the slot. It never waits for anything: the whole point of AC3
// is that an abandoned worker does not keep the reserve under-counting what is
// free.
func (s *Slot) Release() {
	if s == nil || s.closed {
		return
	}
	s.closed = true
	releaseSlot(s.f)
}

// errPoolBusy marks the one refusal that is an availability fact rather than a
// fault: every slot is taken. It advances the chain; nothing else here does.
var errPoolBusy = errors.New("every declared slot for this pool is in use")

// IsPoolBusy reports whether err is the pool-is-full refusal.
//
// The distinction is the same one the dispatcher draws between *pool
// unavailable* and *task failed*, applied one layer down: a full pool means try
// the next entry, an unreadable counter means stop. Collapsing them would send
// the next pool a dispatch against the same broken state.
func IsPoolBusy(err error) bool { return errors.Is(err, errPoolBusy) }

// Acquire takes one of capacity slots for pool, or reports why it could not.
//
// It never treats an unusable state directory as an empty one. Reading a
// counter that cannot be read as "nothing in use" is the fail-OPEN direction:
// it would grant every slot, every time, exactly when the accounting is broken.
func (s *Semaphore) Acquire(pool string, capacity int) (*Slot, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("pool %q declares a capacity of %d, so no dispatch can be admitted", pool, capacity)
	}
	dir := filepath.Join(s.dir, pool)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("semaphore state for pool %q is unusable: %w\n"+
			"this is a refusal, not an empty counter: a counter that cannot be read would otherwise "+
			"read as zero in use and grant every slot", pool, err)
	}

	for i := 0; i < capacity; i++ {
		path := filepath.Join(dir, fmt.Sprintf("slot-%d.lock", i))
		f, taken, err := tryTakeSlot(path)
		if err != nil {
			return nil, fmt.Errorf("semaphore slot %d for pool %q is unusable: %w\n"+
				"this is a refusal, not a free slot: treating a lock that could not be attempted as "+
				"available would grant the slot precisely when the accounting is broken", i, pool, err)
		}
		if taken {
			return &Slot{f: f}, nil
		}
	}
	return nil, fmt.Errorf("pool %q: %w (capacity %d, all held by other dotf dispatches)",
		pool, errPoolBusy, capacity)
}

// DefaultSemaphoreDir is where slot state lives when the caller names no other
// place. It is deliberately NOT under the repo: the budget is a property of the
// machine, and two checkouts of this repo share one NaN subscription.
func DefaultSemaphoreDir() (string, error) {
	// A runtime dir is the right home on Linux — tmpfs, cleared on reboot, so a
	// stale lock cannot outlive the boot that orphaned it. Elsewhere the cache
	// dir is the portable equivalent; correctness does not depend on which,
	// because the lock is released by the kernel either way.
	if rt := os.Getenv("XDG_RUNTIME_DIR"); rt != "" {
		return filepath.Join(rt, "dotf", "slots"), nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no runtime or cache directory for semaphore state: %w", err)
	}
	return filepath.Join(cache, "dotf", "slots"), nil
}
