package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// heldIssueRunner lists one other worktree, on fix/held, whose lease for
// issue 2824 expires at expires.
func heldIssueRunner(t *testing.T, expires time.Time) (*MockAddRunner, string) {
	t.Helper()
	other := t.TempDir()
	if err := SaveMetadata(other, Metadata{Issue: 2824, LeaseExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	porcelain := "worktree " + other + "\nHEAD abc\nbranch refs/heads/fix/held\n\n"
	return &MockAddRunner{
		OnWorktreeList: func(string) (string, error) { return porcelain, nil },
		OnWorktreeAdd:  func(_, target, _, _ string) error { return os.MkdirAll(target, 0o755) },
	}, other
}

func heldIssueOpts(t *testing.T, issue int) AddOptions {
	return AddOptions{RepoRoot: filepath.Join(t.TempDir(), "repo"), WorktreeRoot: t.TempDir(), Slug: "second", Issue: issue,
		LockPath: filepath.Join(t.TempDir(), "wt.lock")}
}

func TestAddRefusesAnIssueALiveLeaseHolds(t *testing.T) {
	now := time.Now()
	runner, other := heldIssueRunner(t, now.Add(time.Hour))
	_, err := AddWithRunner(heldIssueOpts(t, 2824), runner, now)
	if err == nil {
		t.Fatal("add took an issue another live worktree holds")
	}
	for _, want := range []string{other, "fix/held", "--share-issue"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
	if runner.DefaultBaseRan {
		t.Error("add fetched before refusing")
	}
}

func TestAddTakesAnIssueWhoseLeaseExpired(t *testing.T) {
	now := time.Now()
	runner, _ := heldIssueRunner(t, now.Add(-time.Minute))
	if _, err := AddWithRunner(heldIssueOpts(t, 2824), runner, now); err != nil {
		t.Fatalf("an expired lease blocked add: %v", err)
	}
}

func TestAddSharesAHeldIssueWhenAsked(t *testing.T) {
	now := time.Now()
	runner, _ := heldIssueRunner(t, now.Add(time.Hour))
	runner.OnWorktreeList = func(string) (string, error) { return "", errors.New("git worktree list failed") }
	opts := heldIssueOpts(t, 2824)
	opts.ShareIssue = true
	if _, err := AddWithRunner(opts, runner, now); err != nil {
		t.Fatalf("--share-issue did not let add proceed: %v", err)
	}
	if runner.WorktreeListRan {
		t.Error("--share-issue still listed the worktrees")
	}
}

func TestAddRefusesWhenItCannotCheckTheIssue(t *testing.T) {
	runner := &MockAddRunner{OnWorktreeList: func(string) (string, error) { return "", errors.New("boom") }}
	_, err := AddWithRunner(heldIssueOpts(t, 2824), runner, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--share-issue") {
		t.Fatalf("a failed list did not refuse with the override named: %v", err)
	}
}

func TestAddWithoutAnIssueDoesNotScan(t *testing.T) {
	runner, _ := heldIssueRunner(t, time.Now().Add(time.Hour))
	if _, err := AddWithRunner(heldIssueOpts(t, 0), runner, time.Now()); err != nil {
		t.Fatal(err)
	}
	if runner.WorktreeListRan {
		t.Error("add without --issue listed the worktrees")
	}
}

// Two adds for one issue must not both pass the scan before either writes its
// lease: the scan through SaveMetadata runs under the worktree lock.
func TestAddForAnIssueWaitsOnTheWorktreeLock(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "wt.lock")
	unlock, err := TryLockFile(lock)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	runner, _ := heldIssueRunner(t, time.Now().Add(-time.Minute))
	opts := heldIssueOpts(t, 2824)
	opts.LockPath, opts.LockTimeout = lock, 50*time.Millisecond
	if _, err := AddWithRunner(opts, runner, time.Now()); err == nil || runner.WorktreeListRan {
		t.Fatalf("add scanned for #2824 while another add held the lock: %v", err)
	}
	opts.Issue = 0
	if _, err := AddWithRunner(opts, runner, time.Now()); err != nil {
		t.Fatalf("add without --issue waited on the lock: %v", err)
	}
}

// The lock must still be held while the scan runs and while git creates the
// worktree, which is after the scan and before the lease is written: an add
// that let go of it any earlier would let a second add pass the same scan.
func TestAddHoldsTheWorktreeLockFromScanToLease(t *testing.T) {
	runner, _ := heldIssueRunner(t, time.Now().Add(-time.Minute))
	opts := heldIssueOpts(t, 2824)
	held := func(stage string) {
		unlock, err := TryLockFile(opts.LockPath)
		if err == nil {
			unlock()
			t.Errorf("the worktree lock was free during %s", stage)
		} else if !errors.Is(err, ErrLocked) {
			t.Fatalf("probe the lock during %s: %v", stage, err)
		}
	}
	list, add := runner.OnWorktreeList, runner.OnWorktreeAdd
	runner.OnWorktreeList = func(repo string) (string, error) { held("the scan"); return list(repo) }
	runner.OnWorktreeAdd = func(repo, target, branch, base string) error {
		held("the worktree add")
		return add(repo, target, branch, base)
	}
	if _, err := AddWithRunner(opts, runner, time.Now()); err != nil {
		t.Fatal(err)
	}
}
