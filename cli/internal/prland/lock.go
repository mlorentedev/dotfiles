package prland

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

// ErrLandRunning reports that another `pr land` holds the lock for the
// repository.
var ErrLandRunning = errors.New("another pr land is running")

// unsafeName is every character a lock file name should not carry.
var unsafeName = regexp.MustCompile(`[^a-z0-9._+-]`)

// lockName turns owner/name into one file name. The two halves are joined by
// "+", which GitHub forbids in both, so a-b/c and a/b-c do not share a lock.
func lockName(repo string) string {
	return unsafeName.ReplaceAllString(strings.ReplaceAll(strings.ToLower(repo), "/", "+"), "_")
}

// AcquireLock takes the per-repository lock under dir, so two landers cannot
// race each other into updating every PR after every merge. The lock is a
// kernel lock (filelock): it is gone the moment its holder exits, however it
// exits, so a killed lander cannot leave the repository locked. A sibling PID
// file names the holder for the refusal.
//
// A PID file whose process is gone, with the lock free, is the trace of a
// holder that died: it is taken over, with a note on note. alive reports
// whether a PID names a running process.
//
// The returned release frees the lock and removes the PID file; it is safe to
// call more than once.
func AcquireLock(dir, repo string, alive func(pid int) bool, note io.Writer) (func(), error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create the pr land lock directory: %w", err)
	}
	name := lockName(repo)
	if name == "." || name == ".." {
		// filepath.Join would fold these into dir or its parent, and the lock
		// would land outside the directory it is meant to live in.
		return nil, fmt.Errorf("cannot lock %q: not a repository name (want owner/name)", repo)
	}
	base := filepath.Join(dir, name)
	pidPath := base + ".pid"
	unlock, err := filelock.TryLock(base + ".lock")
	if errors.Is(err, filelock.ErrLocked) {
		// A recorded PID that is gone is the previous holder's: the new holder
		// has not written its own yet.
		if pid := readPID(pidPath); pid > 0 && alive(pid) {
			return nil, fmt.Errorf("%w for %s: PID %d holds the lock; wait for it to finish", ErrLandRunning, repo, pid)
		}
		return nil, fmt.Errorf("%w for %s: its PID is not recorded yet; wait for it to finish", ErrLandRunning, repo)
	}
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", base+".lock", err)
	}
	if old := readPID(pidPath); old > 0 && old != os.Getpid() && !alive(old) {
		_, _ = fmt.Fprintf(note, "pr land: taking over the lock of PID %d, which is no longer running\n", old)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		unlock()
		return nil, fmt.Errorf("record the holder of the pr land lock: %w", err)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			// Remove the PID file while the lock is still ours: after the
			// unlock it could be the next holder's.
			_ = os.Remove(pidPath)
			unlock()
		})
	}, nil
}

// readPID reads the holder's PID, or 0 when the file is missing or unreadable.
func readPID(path string) int {
	b, err := os.ReadFile(path) // #nosec G304 -- a path this package built under the state dir
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// RepoName is the owner/name the lock is keyed on: the one the options name,
// else the repository gh resolves from the working directory.
func RepoName(ctx context.Context, o Options) (string, error) {
	if o.Repo != "" {
		return o.Repo, nil
	}
	out, err := o.Run(ctx, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	name := strings.TrimSpace(string(out))
	if err != nil || name == "" {
		return "", fmt.Errorf("cannot tell which repository to lock: pass --repo owner/name (gh repo view: %v)", err)
	}
	return name, nil
}
