package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// HARNESS-142: a review worktree has no branch, so no pull request can ever
// merge it; its lease is the gate. Uncommitted content may be the only copy
// of a verdict, so it is reapable only once that output was copied back.
func TestClassifyAReviewWorktreeByItsLeaseAndSavedOutput(t *testing.T) {
	now := time.Now()
	review := func(lease time.Duration, dirty, saved bool) Info {
		return Info{Path: "/wt/r", IsDetached: true, Dirty: dirty,
			Metadata: &Metadata{Kind: KindReview, ReapOK: true, OutputSaved: saved, CreatedAt: now.Add(-time.Minute), LeaseExpiresAt: now.Add(lease)}}
	}
	held := review(-time.Minute, false, false)
	held.Metadata.ReapOK = false
	if state, reason := Classify(held, now); state != StateActive || !strings.Contains(reason, "reap hold") {
		t.Errorf("held: %s (%s), want ACTIVE naming the reap hold, as sweep refuses it", state, reason)
	}
	for _, c := range []struct {
		name         string
		lease        time.Duration
		dirty, saved bool
		want         LifecycleState
		reason       string
	}{
		{"live", time.Hour, true, false, StateActive, "review"},
		{"expired, clean", -time.Minute, false, false, StateReapable, "review"},
		{"expired, dirty, output saved", -time.Minute, true, true, StateReapable, "review"},
		{"expired, dirty, output never copied back", -time.Minute, true, false, StateDirty, "never copied back"},
	} {
		if state, reason := Classify(review(c.lease, c.dirty, c.saved), now); state != c.want || !strings.Contains(reason, c.reason) {
			t.Errorf("%s: %s (%s), want %s naming %q", c.name, state, reason, c.want, c.reason)
		}
	}
}

type reviewSweep struct {
	repo, wt string
	runner   *MockSweepRunner
	forced   []string
	plain    []string
	branches []string
}

// newReviewSweep lists one review worktree past or within its lease, holding
// untracked reviewer output; saved says whether spec deadline copied it back.
func newReviewSweep(t *testing.T, lease time.Duration, saved bool, now time.Time) *reviewSweep {
	t.Helper()
	tmp := t.TempDir()
	s := &reviewSweep{repo: filepath.Join(tmp, "myrepo"), wt: filepath.Join(tmp, "worktrees", "myrepo", "h142-first-ab12")}
	for _, d := range []string{s.repo, s.wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := SaveMetadata(s.wt, Metadata{Kind: KindReview, ReapOK: true, OutputSaved: saved,
		CreatedAt: now.Add(-3 * time.Hour), LeaseExpiresAt: now.Add(lease)}); err != nil {
		t.Fatal(err)
	}
	s.runner = &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: "worktree " + s.repo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
				"worktree " + s.wt + "\nHEAD 2222\ndetached\n\n",
			DirtyPaths: map[string]bool{s.wt: true},
		},
		OnWorktreeRemove:      func(_, p string) error { s.plain = append(s.plain, p); return nil },
		OnWorktreeRemoveForce: func(_, p string) error { s.forced = append(s.forced, p); return nil },
		OnBranchDelete:        func(_, b string) (string, error) { s.branches = append(s.branches, b); return "", nil },
	}
	return s
}

func (s *reviewSweep) sweep(t *testing.T, now time.Time) *SweepReport {
	t.Helper()
	report, err := SweepWithRunner(SweepOptions{RepoRoot: s.repo, LockPath: filepath.Join(t.TempDir(), "sweep.lock")}, s.runner, now)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// A run whose cleanup was interrupted after copy-back (AC3): sweep removes
// the worktree with --force, deleting no branch.
func TestReviewWorktreeSwept(t *testing.T) {
	withHostProcessInside(t, false)
	now := time.Now()
	s := newReviewSweep(t, -time.Minute, true, now)

	report := s.sweep(t, now)
	if len(report.Reaped) != 1 || len(s.forced) != 1 || s.forced[0] != s.wt {
		t.Fatalf("reaped %d, forced removals %v; want the review worktree removed with --force", len(report.Reaped), s.forced)
	}
	if len(s.plain) != 0 || len(s.branches) != 0 {
		t.Errorf("plain removals %v, branch deletions %v: a review worktree has no branch and its output is untracked", s.plain, s.branches)
	}
}

func TestSweepKeepsAReviewWorktreeThatMayHoldTheOnlyVerdict(t *testing.T) {
	now := time.Now()
	withHostProcessInside(t, false)

	unsaved := newReviewSweep(t, -time.Minute, false, now)
	if report := unsaved.sweep(t, now); len(report.Reaped) != 0 || len(unsaved.forced) != 0 {
		t.Errorf("output never copied back: reaped %d, forced %v", len(report.Reaped), unsaved.forced)
	}

	unreadable := newReviewSweep(t, -time.Minute, true, now)
	unreadable.runner.DirtyErrors = map[string]error{unreadable.wt: errors.New("index.lock exists")}
	if report := unreadable.sweep(t, now); len(report.Reaped) != 0 || len(unreadable.forced) != 0 {
		t.Errorf("unreadable state: reaped %d, forced %v", len(report.Reaped), unreadable.forced)
	}
}

func TestSweepKeepsAReviewWorktreeWhileItsLeaseLivesOrAReviewerRuns(t *testing.T) {
	now := time.Now()

	withHostProcessInside(t, false)
	live := newReviewSweep(t, time.Hour, true, now)
	if report := live.sweep(t, now); len(report.Reaped) != 0 || len(live.forced) != 0 {
		t.Errorf("live lease: reaped %d, forced %v", len(report.Reaped), live.forced)
	}

	// A reviewer still running past its lease is the case the lease alone
	// cannot see; Gate f still refuses it.
	withHostProcessInside(t, true)
	running := newReviewSweep(t, -time.Minute, true, now)
	if report := running.sweep(t, now); len(report.Reaped) != 0 || len(running.forced) != 0 {
		t.Errorf("process inside: reaped %d, forced %v", len(report.Reaped), running.forced)
	}
}

// The real runner: plain `git worktree remove` refuses a tree with untracked
// files, which is exactly what an interrupted review leaves; --force does not.
func TestRealRunnerForceRemovesAReviewWorktreeWithUntrackedOutput(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "myrepo")
	gitT(t, tmp, "init", "-q", "-b", "main", repo)
	gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	wt := filepath.Join(tmp, "review")
	if err := (&RealAddRunner{}).WorktreeAddDetached(repo, wt, gitT(t, repo, "rev-parse", "HEAD")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "review.md"), []byte("verdict\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &RealSweepRunner{}
	if err := r.WorktreeRemove(repo, wt); err == nil {
		t.Fatal("plain remove took a tree with untracked output: the forced path would be unneeded")
	}
	if err := r.WorktreeRemoveForce(repo, wt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree still on disk: %v", err)
	}
}
