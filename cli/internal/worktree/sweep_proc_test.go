package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The refusing branch of Gate f only executes on a platform this CI does not
// run, so without a seam it is unreachable by any test and its behaviour rests
// on reading the code. That is exactly how the defect these tests replace
// survived an adversarial review: isHostProcessInside read /proc with no
// build-tag split, returned false when os.ReadDir failed, and therefore
// reported "nobody is inside" on every Windows run.
//
// Driving the seam both ways is what makes the fix checkable. The direction
// that matters is the second one: a gate that cannot answer must not clear the
// worktree for deletion.
//
// Every test here drives gateF or SweepWithRunner, never a wrapper. Round 3
// found the previous version pinning isCandidateForReap, which production did
// not call — a refusal proven on a function that could not prevent a deletion.

func withHostProcessInside(t *testing.T, answer bool) {
	t.Helper()
	withGateFReading(t, GateFReading{Inside: answer})
}

func withGateFReading(t *testing.T, reading GateFReading) {
	t.Helper()
	original := hostProcessInside
	hostProcessInside = func(string) GateFReading { return reading }
	t.Cleanup(func() { hostProcessInside = original })
}

func reapableInfo(t *testing.T) Info {
	t.Helper()
	return Info{
		Path:     filepath.Join(t.TempDir(), "repo-wt-feat"),
		Branch:   "feat/x",
		State:    StateReapable,
		PRMerged: true,
		Dirty:    false,
		IsOrphan: false,
		IsMain:   false,
	}
}

func TestGateFRefusesWhenProcessDiscoveryCannotAnswer(t *testing.T) {
	withHostProcessInside(t, true)

	if _, ok := gateF(reapableInfo(t), "/somewhere/else"); ok {
		t.Error("a worktree was cleared for reaping while Gate f reported a live process inside; " +
			"an unanswerable gate must refuse, because the caller deletes on a false")
	}
}

func TestGateFAllowsAReapableWorktreeWhenNothingIsInside(t *testing.T) {
	withHostProcessInside(t, false)

	if _, ok := gateF(reapableInfo(t), "/somewhere/else"); !ok {
		t.Error("a reapable worktree with no process inside was refused; the gate has stopped " +
			"discriminating and now refuses everything, which passes the test above vacuously")
	}
}

// Uninspectable processes deliberately do NOT block the reap -- refusing on them
// would make sweep inert on Linux, since /proc/1/cwd is unreadable to every
// non-root caller. This pins that the count survives the trip to the report;
// that the producer actually increments it is a separate test, in
// sweep_proc_linux_test.go, because a seam cannot see its own producer.
func TestUninspectableProcessesDoNotBlockButAreCarried(t *testing.T) {
	withGateFReading(t, GateFReading{Inside: false, Uninspectable: 7})

	reading, ok := gateF(reapableInfo(t), "/somewhere/else")
	if !ok {
		t.Error("unreadable processes blocked the reap; refusing on them makes sweep " +
			"permanently inert on Linux, which is why they are reported instead")
	}
	if reading.Uninspectable != 7 {
		t.Errorf("gateF dropped the uninspectable count: got %d, want 7 — the report would "+
			"then describe a reach the scan did not have", reading.Uninspectable)
	}
}

// The gate that actually guards the deletion is the SECOND one.
//
// SweepWithRunner consults Gate f twice: once to select candidates, and again
// inside reapSingleWorktree under the lock, immediately before removing the
// worktree. The re-check exists for the window between them -- a shell that
// cds in while the sweep is deciding -- and a seam with one constant answer can
// never reach it, because a constant `true` is refused by the first call and a
// constant `false` never exercises the second. Round 3 measured the
// consequence: deleting the re-check left the whole suite green.
//
// So this seam answers false first and true afterwards, which is precisely the
// TOCTOU the re-check is for.
func TestSweepRefusesWhenAProcessArrivesAfterTheGate(t *testing.T) {
	original := hostProcessInside
	calls := 0
	hostProcessInside = func(string) GateFReading {
		calls++
		return GateFReading{Inside: calls > 1}
	}
	t.Cleanup(func() { hostProcessInside = original })

	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	reapableWT := filepath.Join(tmpDir, "myrepo-wt-reapable")
	if err := os.MkdirAll(mainRepo, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(reapableWT, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	now := time.Now()
	if err := SaveMetadata(reapableWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}

	removed := []string{}
	runner := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
				"worktree " + reapableWT + "\nHEAD 2222\nbranch refs/heads/feat/reapable\n\n",
			MergedBranches: map[string]bool{"feat/reapable": true},
		},
		OnWorktreeRemove: func(_, path string) error {
			removed = append(removed, path)
			return nil
		},
	}

	report, err := SweepWithRunner(SweepOptions{
		RepoRoot: mainRepo,
		LockPath: filepath.Join(tmpDir, "sweep.lock"),
	}, runner, now)
	if err != nil {
		t.Fatalf("unexpected error during sweep: %v", err)
	}

	// The anchor: without it, a fixture that never became reapable would pass
	// this test by reaping nothing for the wrong reason.
	if calls < 2 {
		t.Fatalf("Gate f was consulted %d time(s); the fixture never reached the re-check, "+
			"so this test proves nothing about it", calls)
	}
	if len(removed) != 0 {
		t.Errorf("a worktree was removed after Gate f reported a process had arrived inside it: %v", removed)
	}
	if len(report.Reaped) != 0 {
		t.Errorf("sweep reported reaping %v despite the second gate refusing", report.Reaped)
	}
	if report.SkippedCount != 2 { // mainRepo + the worktree the re-check saved
		t.Errorf("expected 2 skipped (main and the refused worktree), got %d", report.SkippedCount)
	}
}

// orderRecordingRunner notes the sequence of the checks that run under the lock,
// so the ORDER can be asserted rather than read.
type orderRecordingRunner struct {
	*MockSweepRunner
	order *[]string
}

func (r *orderRecordingRunner) IsDirty(path string) (bool, error) {
	*r.order = append(*r.order, "dirty")
	return r.MockSweepRunner.IsDirty(path)
}

func (r *orderRecordingRunner) IsPRMerged(repoRoot, branch string) (bool, error) {
	*r.order = append(*r.order, "merged")
	return r.MockSweepRunner.IsPRMerged(repoRoot, branch)
}

// Gate f must be the LAST check before the removal, and the ordering is a
// correctness property rather than a preference.
//
// isDirty and isMerged shell out to git. A Gate f check placed before them
// leaves that latency inside the window between "nobody is in there" and the
// directory being deleted, so a shell that cds in while git runs is missed.
// Round 4 raised it as a widened race window; the fix is the order, and this is
// what stops it silently reverting.
func TestGateFIsTheLastCheckBeforeRemoval(t *testing.T) {
	var order []string

	original := hostProcessInside
	hostProcessInside = func(string) GateFReading {
		order = append(order, "gatef")
		return GateFReading{Inside: false}
	}
	t.Cleanup(func() { hostProcessInside = original })

	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	reapableWT := filepath.Join(tmpDir, "myrepo-wt-reapable")
	for _, d := range []string{mainRepo, reapableWT} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	now := time.Now()
	if err := SaveMetadata(reapableWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}

	base := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
				"worktree " + reapableWT + "\nHEAD 2222\nbranch refs/heads/feat/reapable\n\n",
			MergedBranches: map[string]bool{"feat/reapable": true},
		},
		OnWorktreeRemove: func(_, _ string) error {
			order = append(order, "remove")
			return nil
		},
	}

	report, err := SweepWithRunner(SweepOptions{
		RepoRoot: mainRepo,
		LockPath: filepath.Join(tmpDir, "sweep.lock"),
	}, &orderRecordingRunner{MockSweepRunner: base, order: &order}, now)
	if err != nil {
		t.Fatalf("unexpected error during sweep: %v", err)
	}

	// Anchor: if nothing was reaped, the sequence below never reached the part
	// this test is about and every assertion on it would pass vacuously.
	if len(report.Reaped) != 1 {
		t.Fatalf("fixture did not reap: %v (order was %v) — the ordering assertions "+
			"would be meaningless", report.Reaped, order)
	}

	removeAt := indexOf(order, "remove")
	if removeAt < 1 {
		t.Fatalf("no removal recorded in %v", order)
	}
	if order[removeAt-1] != "gatef" {
		t.Errorf("the check immediately before the removal was %q, not Gate f.\n"+
			"full order: %v\n"+
			"Anything between Gate f and the removal is time in which a process can "+
			"enter the worktree unseen, and isDirty/isMerged are git shell-outs.",
			order[removeAt-1], order)
	}
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}
