package lane

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// holdingSweep is a sweep runner that parks the first run inside the sweep, so
// that run holds the sweep lock for as long as the test wants. IsPRMerged is
// asked once per worktree, under the lock, by the real sweep.
type holdingSweep struct {
	ForgeSweep
	once    sync.Once
	inside  chan struct{}
	release chan struct{}
}

func (h *holdingSweep) IsPRMerged(repo, branch string) (bool, error) {
	h.once.Do(func() {
		close(h.inside)
		<-h.release
	})
	return h.ForgeSweep.IsPRMerged(repo, branch)
}

// Two `lane apply` runs started at once (#2617, a question the CLI-127 round-3
// review left open). The sweep half is serialised by the worktree lock: the
// second run is refused with ErrLocked, not queued behind the first. Apply stops
// at a failed sweep, so the refused run never reaches the start half and cannot
// start a second lane; the first run starts exactly one. How a start is claimed
// between sessions that do reach it is lane_claim_test.go's question, not this
// one's. This runs the real sweep over a real repository.
func TestApplyTwoConcurrentRunsGiveOneChangeSetAndOneLoudRefusal(t *testing.T) {
	now := time.Now()
	repo, wts := fixtureRepo(t, now)
	forge := CachedForge(&fakeForge{prs: map[string]PRInfo{"fix/merged": {Number: 10, State: PRMerged}}})
	lockPath := filepath.Join(t.TempDir(), "sweep.lock")

	acts := []Action{
		{Kind: KindReap, Class: ClassAuto, Branch: "fix/merged"},
		{Kind: KindStart, Class: ClassAuto, Branch: "fix/next", Ticket: 9},
	}
	var starts atomic.Int32
	verbsFor := func(runner worktree.SweepRunner) Verbs {
		return Verbs{
			Sweep: func() (int, error) {
				report, err := worktree.SweepWithRunner(worktree.SweepOptions{RepoRoot: repo, LockPath: lockPath}, runner, now)
				if err != nil {
					return 0, err
				}
				return len(report.Reaped), nil
			},
			Next: func() (bool, error) { starts.Add(1); return true, nil },
		}
	}

	first := &holdingSweep{
		ForgeSweep: ForgeSweep{RealSweepRunner: &worktree.RealSweepRunner{}, Forge: forge},
		inside:     make(chan struct{}),
		release:    make(chan struct{}),
	}
	type result struct {
		got Applied
		err error
	}
	done := make(chan result, 1)
	go func() {
		got, err := Apply(acts, verbsFor(first))
		done <- result{got, err}
	}()

	select {
	case <-first.inside:
	case <-time.After(30 * time.Second):
		t.Fatal("the first run never reached the sweep")
	}
	// The first run holds the sweep lock now. The second starts over the same
	// fixture and must be refused, loudly.
	second := ForgeSweep{RealSweepRunner: &worktree.RealSweepRunner{}, Forge: forge}
	got2, err2 := Apply(acts, verbsFor(second))
	if !errors.Is(err2, worktree.ErrLocked) {
		t.Errorf("the second run was not refused with the sweep lock: %v", err2)
	}
	if got2.Changed != 0 || starts.Load() != 0 {
		t.Errorf("the refused run changed %d and the starts so far are %d, want 0 and 0", got2.Changed, starts.Load())
	}

	close(first.release)
	var r1 result
	select {
	case r1 = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the first run never finished")
	}
	if r1.err != nil {
		t.Fatalf("the run that held the lock failed: %v", r1.err)
	}
	// One start, and the reap too wherever sweep can look at processes; where it
	// cannot (Windows) it refuses to guess and reaps nothing.
	want := 1
	if _, scans := worktree.ProcessInside(wts[0]); scans {
		want = 2
	}
	if r1.got.Changed != want {
		t.Errorf("the first run changed %d, want %d", r1.got.Changed, want)
	}
	if n := starts.Load(); n != 1 {
		t.Errorf("%d lanes started across the two runs, want exactly 1", n)
	}
}
