package lane

import (
	"errors"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

type fakeForge struct {
	prs   map[string]PRInfo
	calls map[string]int
}

func (f *fakeForge) PRFor(branch string) (PRInfo, error) {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[branch]++
	return f.prs[branch], nil
}

type fakeFinder struct {
	found map[string]Worktree
	err   error
}

func (f fakeFinder) Find(_, branch string) (Worktree, bool, error) {
	w, ok := f.found[branch]
	return w, ok, f.err
}

func wt(branch string, meta *worktree.Metadata) worktree.Info {
	return worktree.Info{Path: "/wt/" + branch, Branch: branch, Metadata: meta}
}

func rowFor(t *testing.T, rows []Row, branch string) Row {
	t.Helper()
	for _, r := range rows {
		if r.Branch == branch {
			return r
		}
	}
	t.Fatalf("no row for %s in %+v", branch, rows)
	return Row{}
}

// Status joins the worktree list, the forge, liveness and herdr into one row
// per lane, and reaches the forge once per branch however many rows ask.
func TestStatus_OneRowPerLaneWithItsDerivedPhase(t *testing.T) {
	main := worktree.Info{Path: "/repo", Branch: "main", IsMain: true}
	forge := &fakeForge{prs: map[string]PRInfo{
		"fix/review":  {Number: 30, State: PROpen},
		"fix/landing": {Number: 20, State: PROpen},
		"fix/landed":  {Number: 10, State: PRMerged},
		"fix/reaped":  {Number: 5, State: PRMerged},
	}}
	src := Sources{
		Now:   phaseNow,
		Queue: []int{1, 2, 3},
		Issues: fakeIssues{
			1: {Number: 1, Title: "Pending thing", State: "OPEN"},
			2: {Number: 2, Title: "Reaped", State: "CLOSED"},
			3: {Number: 3, Title: "Held", State: "OPEN", Assignees: []string{"someone"}},
		},
		BranchPrefix: "fix",
		List: func() ([]worktree.Info, error) {
			return []worktree.Info{
				main,
				wt("fix/running", leased(time.Hour)),
				wt("fix/landing", leased(time.Hour)),
				wt("fix/review", leased(time.Hour)),
				wt("fix/landed", leased(-time.Hour)),
				wt("fix/stalled", leased(-time.Hour)),
				{Path: "/wt/fix/orphan", Branch: "fix/orphan", IsOrphan: true, Metadata: leased(-time.Hour)},
				wt("fix/adopt-me", nil),
			}, nil
		},
		Forge: forge,
		Live:  func(p string) (bool, error) { return p == "/wt/fix/running", nil },
		Herdr: fakeFinder{found: map[string]Worktree{"fix/running": {Open: true}}},
	}
	rows, err := Status(src)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Phase{
		"fix/running":       PhaseRunning,
		"fix/landing":       PhaseLanding, // PR 20 is the lowest open number
		"fix/review":        PhaseReview,
		"fix/landed":        PhaseLanded,
		"fix/stalled":       PhaseStalled,
		"fix/orphan":        PhaseOrphan,
		"fix/adopt-me":      PhaseUnadopted,
		"fix/pending-thing": PhasePending,
		"fix/reaped":        PhaseReaped,
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for branch, phase := range want {
		if got := rowFor(t, rows, branch).Phase; got != phase {
			t.Errorf("%s: phase %q, want %q", branch, got, phase)
		}
	}
	for branch, n := range forge.calls {
		if n != 1 {
			t.Errorf("forge asked %d times for %s, want 1", n, branch)
		}
	}
}

// herdr being unreachable is a supported state: the rows are still there,
// marked absent, and the phases are the ones git and the forge give.
func TestStatus_HerdrAbsentStillAnswers(t *testing.T) {
	src := Sources{
		Now:   phaseNow,
		List:  func() ([]worktree.Info, error) { return []worktree.Info{wt("fix/a", leased(time.Hour))}, nil },
		Forge: &fakeForge{},
		Live:  func(string) (bool, error) { return false, nil },
		Herdr: fakeFinder{err: errors.New("no herdr server")},
	}
	rows, err := Status(src)
	if err != nil {
		t.Fatal(err)
	}
	if r := rows[0]; r.Herdr != HerdrAbsent || r.Phase != PhaseRunning {
		t.Errorf("row = %+v, want herdr absent and phase Running", r)
	}
}

// A liveness probe that fails reads as alive: reaping a live session is the
// expensive mistake, a stale "Running" row is not.
func TestStatus_UnreadableLivenessReadsAsLive(t *testing.T) {
	src := Sources{
		Now:   phaseNow,
		List:  func() ([]worktree.Info, error) { return []worktree.Info{wt("fix/a", leased(-time.Hour))}, nil },
		Forge: &fakeForge{},
		Live:  func(string) (bool, error) { return false, errors.New("cannot read the session store") },
		Herdr: fakeFinder{},
	}
	rows, err := Status(src)
	if err != nil {
		t.Fatal(err)
	}
	if r := rows[0]; !r.Live || r.Phase != PhaseRunning {
		t.Errorf("row = %+v, want live and Running", r)
	}
}
