package lane

import (
	"errors"
	"testing"
	"time"
)

// recorder is a Verbs that counts what Apply asks of it. Sweep answers with
// the number of worktrees it "reaped".
type recorder struct {
	sweeps, nexts int
	reaped        int
	started       bool
	err           error
}

func (r *recorder) verbs() Verbs {
	return Verbs{
		Sweep: func() (int, error) { r.sweeps++; return r.reaped, r.err },
		Next:  func() (bool, error) { r.nexts++; return r.started, r.err },
	}
}

func TestApply(t *testing.T) {
	auto := []Action{
		{Kind: KindReap, Class: ClassAuto, Branch: "a"},
		{Kind: KindReap, Class: ClassAuto, Branch: "b"},
		{Kind: KindStart, Class: ClassAuto, Branch: "c", Ticket: 9},
	}
	notAuto := []Action{
		{Kind: KindReap, Class: ClassPending, Branch: "p", Due: phaseNow.Add(time.Hour)},
		{Kind: KindReap, Class: ClassAmbiguous, Branch: "x", Resolve: "dotf worktree done --force /wt/x"},
		{Kind: KindAdopt, Class: ClassAmbiguous, Branch: "y", Resolve: "dotf worktree adopt /wt/y"},
		{Kind: KindFlag, Class: ClassAmbiguous, Branch: "z"},
	}

	t.Run("runs sweep once and next once for the auto actions and counts what changed", func(t *testing.T) {
		rec := &recorder{reaped: 2, started: true}
		got, err := Apply(auto, rec.verbs())
		if err != nil {
			t.Fatal(err)
		}
		if rec.sweeps != 1 || rec.nexts != 1 {
			t.Errorf("sweeps=%d nexts=%d, want 1 and 1", rec.sweeps, rec.nexts)
		}
		if got.Changed != 3 {
			t.Errorf("changed=%d, want 3 (two reaped, one started)", got.Changed)
		}
	})

	t.Run("never runs a verb for pending, adopt, flag or any ambiguous action", func(t *testing.T) {
		rec := &recorder{reaped: 5, started: true}
		got, err := Apply(notAuto, rec.verbs())
		if err != nil {
			t.Fatal(err)
		}
		if rec.sweeps != 0 || rec.nexts != 0 || got.Changed != 0 {
			t.Errorf("sweeps=%d nexts=%d changed=%d, want all 0", rec.sweeps, rec.nexts, got.Changed)
		}
	})

	t.Run("counts what sweep reaped, not what the plan proposed", func(t *testing.T) {
		rec := &recorder{reaped: 1}
		got, err := Apply(auto[:2], rec.verbs())
		if err != nil {
			t.Fatal(err)
		}
		if got.Changed != 1 {
			t.Errorf("changed=%d, want 1: sweep's own gates decide", got.Changed)
		}
	})

	t.Run("a start that only printed commands is not a change", func(t *testing.T) {
		rec := &recorder{started: false}
		got, err := Apply(auto[2:], rec.verbs())
		if err != nil {
			t.Fatal(err)
		}
		if rec.nexts != 1 || got.Changed != 0 {
			t.Errorf("nexts=%d changed=%d, want 1 and 0", rec.nexts, got.Changed)
		}
	})

	t.Run("a second apply over the post-state changes nothing", func(t *testing.T) {
		rows := []Row{row("feat/q", PhasePending, func(r *Row) { r.Ticket = 4 })}
		first := Plan(rows, []int{4}, phaseNow)
		rec := &recorder{started: true}
		if got, _ := Apply(first, rec.verbs()); got.Changed != 1 {
			t.Fatalf("first apply changed=%d, want 1", got.Changed)
		}
		after := []Row{row("feat/q", PhaseRunning, func(r *Row) { r.Ticket = 4 })}
		second := Plan(after, []int{4}, phaseNow)
		rec = &recorder{started: true}
		got, err := Apply(second, rec.verbs())
		if err != nil || got.Changed != 0 || rec.sweeps+rec.nexts != 0 {
			t.Errorf("second apply changed=%d calls=%d err=%v, want 0", got.Changed, rec.sweeps+rec.nexts, err)
		}
	})

	t.Run("stops at the first verb error and reports what had changed", func(t *testing.T) {
		boom := errors.New("boom")
		rec := &recorder{reaped: 1, err: nil}
		v := rec.verbs()
		v.Next = func() (bool, error) { return false, boom }
		got, err := Apply(auto, v)
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v, want boom", err)
		}
		if got.Changed != 1 {
			t.Errorf("changed=%d, want the 1 reap that happened before the error", got.Changed)
		}
	})
}

// A worktree someone may be inside is never proposed as auto, so apply never
// hands it to sweep: the plan is the first gate, sweep's gate f the second.
func TestApplyNeverReapsALiveWorktree(t *testing.T) {
	for name, live := range map[string]Liveness{
		"a process inside":         LiveYes,
		"a recent session":         LiveYes,
		"liveness that is unknown": LiveUnknown,
	} {
		t.Run(name, func(t *testing.T) {
			rows := []Row{row("feat/busy", PhaseLanded, func(r *Row) { reapable(r); r.Live = live })}
			acts := Plan(rows, nil, phaseNow)
			a := find(t, acts, "feat/busy")
			if a.Class != ClassAmbiguous {
				t.Fatalf("class=%s, want ambiguous", a.Class)
			}
			rec := &recorder{reaped: 1}
			got, err := Apply(acts, rec.verbs())
			if err != nil {
				t.Fatal(err)
			}
			if rec.sweeps != 0 || got.Changed != 0 {
				t.Errorf("sweeps=%d changed=%d: a live worktree reached sweep", rec.sweeps, got.Changed)
			}
		})
	}
}
