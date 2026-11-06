package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// HARNESS-142: a review worktree has no branch, so no pull request can ever
// merge it, and an interrupted review leaves its output untracked inside it.
// Under the ordinary gates it would never be reapable; its lease is the gate.
func TestClassifyAReviewWorktreeByItsLease(t *testing.T) {
	now := time.Now()
	review := func(lease time.Duration) Info {
		return Info{Path: "/wt/r", IsDetached: true, Dirty: true,
			Metadata: &Metadata{Kind: KindReview, ReapOK: true, CreatedAt: now.Add(-time.Minute), LeaseExpiresAt: now.Add(lease)}}
	}
	if state, reason := Classify(review(time.Hour), now); state != StateActive || !strings.Contains(reason, "review") {
		t.Errorf("live review worktree: %s (%s), want ACTIVE naming the review", state, reason)
	}
	if state, reason := Classify(review(-time.Minute), now); state != StateReapable || !strings.Contains(reason, "review") {
		t.Errorf("expired review worktree, dirty and unmerged: %s (%s), want REAPABLE naming the review", state, reason)
	}
}

type reviewSweep struct {
	repo, wt string
	runner   *MockSweepRunner
	forced   []string
	plain    []string
	branches []string
}

func newReviewSweep(t *testing.T, lease time.Duration, now time.Time) *reviewSweep {
	t.Helper()
	tmp := t.TempDir()
	s := &reviewSweep{repo: filepath.Join(tmp, "myrepo"), wt: filepath.Join(tmp, "worktrees", "myrepo", "h142-first-ab12")}
	for _, d := range []string{s.repo, s.wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := SaveMetadata(s.wt, Metadata{Kind: KindReview, ReapOK: true,
		CreatedAt: now.Add(-3 * time.Hour), LeaseExpiresAt: now.Add(lease)}); err != nil {
		t.Fatal(err)
	}
	s.runner = &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: "worktree " + s.repo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
				"worktree " + s.wt + "\nHEAD 2222\ndetached\n\n",
			DirtyPaths: map[string]bool{s.wt: true}, // the interrupted review's untracked output
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

func TestReviewWorktreeSwept(t *testing.T) {
	withHostProcessInside(t, false)
	now := time.Now()
	s := newReviewSweep(t, -time.Minute, now)

	report := s.sweep(t, now)
	if len(report.Reaped) != 1 || len(s.forced) != 1 || s.forced[0] != s.wt {
		t.Fatalf("reaped %d, forced removals %v; want the review worktree removed with --force", len(report.Reaped), s.forced)
	}
	if len(s.plain) != 0 || len(s.branches) != 0 {
		t.Errorf("plain removals %v, branch deletions %v: a review worktree has no branch and its output is untracked", s.plain, s.branches)
	}
}

func TestSweepKeepsAReviewWorktreeWhileItsLeaseLivesOrAReviewerRuns(t *testing.T) {
	now := time.Now()

	withHostProcessInside(t, false)
	live := newReviewSweep(t, time.Hour, now)
	if report := live.sweep(t, now); len(report.Reaped) != 0 || len(live.forced) != 0 {
		t.Errorf("live lease: reaped %d, forced %v", len(report.Reaped), live.forced)
	}

	// A reviewer still running past its lease is the case the lease alone
	// cannot see; Gate f still refuses it.
	withHostProcessInside(t, true)
	running := newReviewSweep(t, -time.Minute, now)
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
