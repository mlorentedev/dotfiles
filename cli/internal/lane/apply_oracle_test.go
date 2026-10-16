package lane

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// plan and apply must answer "is it merged?" from the same place. Plan reads
// the forge; a sweep left on its own asks ancestry, which calls a worktree
// whose branch has no commit of its own merged, since its tip is already in the
// base. So a worktree plan handed to a human as ambiguous would be reaped by
// apply (CLI-127 review, F1). This runs the real sweep over a real repository.
func TestApplyNeverReapsWhatPlanCalledAmbiguous(t *testing.T) {
	now := time.Now()
	repo, wts := fixtureRepo(t, now)
	merged := wts[0]
	// Leased, expired, clean, at the base commit, and no pull request: ancestry
	// says merged, the forge says there is nothing to merge.
	stalled := filepath.Join(filepath.Dir(repo), "stalled")
	git(t, repo, "worktree", "add", "-q", "-b", "fix/stalled", stalled)
	if err := worktree.SaveMetadata(stalled, worktree.Metadata{
		Issue: 8, CreatedAt: now.Add(-72 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour), ReapOK: true,
	}); err != nil {
		t.Fatal(err)
	}

	forge := CachedForge(&fakeForge{prs: map[string]PRInfo{"fix/merged": {Number: 10, State: PRMerged}}})
	src := Sources{
		Now: now, Repo: repo,
		List: func() ([]worktree.Info, error) {
			return worktree.ListWithRunner(repo, ForgeGit{RealGitRunner: &worktree.RealGitRunner{}, Forge: forge}, now)
		},
		Forge: forge,
		Live:  func(string) Liveness { return LiveIdle },
		Herdr: fakeFinder{},
	}
	rows, err := Status(src)
	if err != nil {
		t.Fatal(err)
	}
	acts := Plan(rows, nil, now)
	if a := find(t, acts, "fix/stalled"); a.Class != ClassAmbiguous {
		t.Fatalf("plan classed the stalled worktree %s, want ambiguous: %+v", a.Class, a)
	}
	if a := find(t, acts, "fix/merged"); a.Class != ClassAuto {
		t.Fatalf("plan classed the merged worktree %s, want auto: %+v", a.Class, a)
	}

	verbs := Verbs{
		Sweep: func() (int, error) {
			report, err := worktree.SweepWithRunner(worktree.SweepOptions{RepoRoot: repo, LockPath: filepath.Join(t.TempDir(), "sweep.lock")},
				ForgeSweep{RealSweepRunner: &worktree.RealSweepRunner{}, Forge: forge}, now)
			if err != nil {
				return 0, err
			}
			return len(report.Reaped), nil
		},
		Next: func() (bool, error) { t.Fatal("apply started a lane"); return false, nil },
	}
	got, err := Apply(acts, verbs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stalled); err != nil {
		t.Errorf("apply removed a worktree plan classed ambiguous: %v", err)
	}
	// Where no process scan exists (Windows) sweep reaps nothing at all: gate f
	// refuses rather than guesses. The half above holds there too; this one
	// needs a platform that can look.
	if _, scans := worktree.ProcessInside(merged); !scans {
		t.Logf("no process scan on this platform: sweep reaps nothing, changed=%d", got.Changed)
		return
	}
	if got.Changed != 1 {
		t.Errorf("changed=%d, want 1: only the merged worktree", got.Changed)
	}
	if _, err := os.Stat(merged); !os.IsNotExist(err) {
		t.Errorf("apply left the merged worktree plan classed auto (stat err: %v)", err)
	}
}
