package lane

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// row builds a lane row the way Status does, from the worktree's own state.
func row(branch string, phase Phase, o func(*Row)) Row {
	r := Row{Branch: branch, Path: "/wt/" + branch, Phase: phase, Live: LiveIdle, Herdr: HerdrListed}
	if o != nil {
		o(&r)
	}
	return r
}

func lease(created, expires time.Time, reapOK bool) *worktree.Metadata {
	return &worktree.Metadata{Issue: 7, CreatedAt: created, LeaseExpiresAt: expires, ReapOK: reapOK}
}

func reapable(r *Row) {
	r.State = worktree.StateReapable
	r.Meta = lease(phaseNow.Add(-48*time.Hour), phaseNow.Add(-time.Hour), true)
	r.PR, r.PRState = 10, string(PRMerged)
}

// find is the one action planned for branch.
func find(t *testing.T, acts []Action, branch string) Action {
	t.Helper()
	var got []Action
	for _, a := range acts {
		if a.Branch == branch {
			got = append(got, a)
		}
	}
	if len(got) != 1 {
		t.Fatalf("%s: %d actions, want 1: %+v", branch, len(got), acts)
	}
	return got[0]
}

// Plan classes every action D1 describes. One row per case, and the table
// is the semantics: an action is auto only where an existing verb accepts it
// without --force, pending where only time is missing, ambiguous otherwise.
func TestPlan(t *testing.T) {
	soon := phaseNow.Add(3 * time.Hour)
	cases := []struct {
		name  string
		row   Row
		want  Action // Kind, Class, Due and Resolve are compared
		queue []int
	}{
		{"reapable and idle is auto", row("fix/a", PhaseLanded, reapable), Action{Kind: KindReap, Class: ClassAuto}, nil},
		{"reapable with a live session is ambiguous", row("fix/a", PhaseLanded, func(r *Row) { reapable(r); r.Live = LiveYes }),
			Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"reapable with unknown liveness is ambiguous", row("fix/a", PhaseLanded, func(r *Row) { reapable(r); r.Live = LiveUnknown }),
			Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"merged with the lease still running and a live session is ambiguous, not pending", row("fix/a", PhaseLanded, func(r *Row) {
			r.State, r.Live, r.Meta = worktree.StateActive, LiveYes, lease(phaseNow.Add(-24*time.Hour), soon, true)
		}), Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"merged and dirty is ambiguous", row("fix/a", PhaseLanded, func(r *Row) {
			r.State, r.Dirty, r.Meta = worktree.StateDirty, true, lease(phaseNow.Add(-48*time.Hour), phaseNow.Add(-time.Hour), true)
		}), Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"merged with the lease still running is pending until it expires", row("fix/a", PhaseLanded, func(r *Row) {
			r.State, r.Meta = worktree.StateActive, lease(phaseNow.Add(-24*time.Hour), soon, true)
		}), Action{Kind: KindReap, Class: ClassPending, Due: soon}, nil},
		{"merged and newly created is pending until it is 15 minutes old", row("fix/a", PhaseLanded, func(r *Row) {
			r.State, r.Meta = worktree.StateActive, lease(phaseNow.Add(-5*time.Minute), phaseNow.Add(-time.Minute), true)
		}), Action{Kind: KindReap, Class: ClassPending, Due: phaseNow.Add(10 * time.Minute)}, nil},
		{"merged under a reap hold is ambiguous", row("fix/a", PhaseLanded, func(r *Row) {
			r.State, r.Meta = worktree.StateActive, lease(phaseNow.Add(-48*time.Hour), phaseNow.Add(-time.Hour), false)
		}), Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"stalled is ambiguous", row("fix/a", PhaseStalled, nil),
			Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		{"orphan is ambiguous", row("fix/a", PhaseOrphan, nil),
			Action{Kind: KindFlag, Class: ClassAmbiguous, Resolve: "dotf worktree done --force /wt/fix/a"}, nil},
		// D2: adoption is never auto, and is proposed only when intent is derivable.
		{"unadopted with a ticket in the branch is an adoption proposal", row("fix/2501-a", PhaseUnadopted, func(r *Row) { r.Ticket = 2501 }),
			Action{Kind: KindAdopt, Class: ClassAmbiguous, Resolve: "dotf worktree adopt /wt/fix/2501-a"}, nil},
		{"unadopted with a PR is an adoption proposal", row("fix/a", PhaseUnadopted, func(r *Row) { r.PR, r.PRState = 5, string(PROpen) }),
			Action{Kind: KindAdopt, Class: ClassAmbiguous, Resolve: "dotf worktree adopt /wt/fix/a"}, nil},
		{"unadopted with no derivable intent is reported with no proposal", row("fix/a", PhaseUnadopted, nil),
			Action{Kind: KindFlag, Class: ClassAmbiguous}, nil},
		// The next queued item starts on its own only when nothing else is in flight.
		{"the first pending item starts when no lane is active", row("fix/p1", PhasePending, func(r *Row) { r.Ticket, r.Path = 1, "" }),
			Action{Kind: KindStart, Class: ClassAuto}, []int{1, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := find(t, Plan([]Row{c.row}, c.queue, phaseNow), c.row.Branch)
			if got.Kind != c.want.Kind || got.Class != c.want.Class || !got.Due.Equal(c.want.Due) || got.Resolve != c.want.Resolve {
				t.Errorf("action = %+v, want kind=%s class=%s due=%s resolve=%q", got, c.want.Kind, c.want.Class, c.want.Due, c.want.Resolve)
			}
		})
	}
	t.Run("a start waits while any lane is in flight, and only the first pending item is started", func(t *testing.T) {
		pend := func(b string, n int) Row { return row(b, PhasePending, func(r *Row) { r.Ticket, r.Path = n, "" }) }
		busy := Plan([]Row{row("fix/r", PhaseRunning, nil), pend("fix/p1", 1)}, []int{1}, phaseNow)
		if len(busy) != 0 {
			t.Errorf("a Running lane is in flight, but plan has %+v", busy)
		}
		acts := Plan([]Row{pend("fix/p2", 2), pend("fix/p1", 1)}, []int{1, 2}, phaseNow)
		if len(acts) != 1 || acts[0].Branch != "fix/p1" || acts[0].Kind != KindStart {
			t.Errorf("want one start for the first item in queue order (fix/p1), got %+v", acts)
		}
	})
	t.Run("Running, Review, Landing and Reaped plan nothing", func(t *testing.T) {
		rows := []Row{row("a", PhaseRunning, nil), row("b", PhaseReview, nil), row("c", PhaseLanding, nil), row("d", PhaseReaped, nil)}
		if acts := Plan(rows, nil, phaseNow); len(acts) != 0 {
			t.Errorf("got %+v, want none", acts)
		}
	})
	t.Run("landing is ordered and never an action", func(t *testing.T) {
		rows := []Row{
			row("fix/second", PhaseReview, func(r *Row) { r.PR, r.PRState = 30, string(PROpen) }),
			row("fix/first", PhaseLanding, func(r *Row) { r.PR, r.PRState = 20, string(PROpen) }),
		}
		for _, a := range Plan(rows, nil, phaseNow) {
			t.Errorf("landing planned an action: %+v", a)
		}
		order := LandingOrder(rows)
		if len(order) != 2 || order[0].PR != 20 || !order[0].Head || order[1].PR != 30 || order[1].Head {
			t.Fatalf("order = %+v, want PR 20 (head) then 30", order)
		}
		if want := "dotf pr triage-queue --pr 20 && dotf pr land 20"; order[0].Command != want {
			t.Errorf("head command = %q, want %q", order[0].Command, want)
		}
		if order[1].Command != "" {
			t.Errorf("only the head gets a command, got %q", order[1].Command)
		}
	})
}

// minAge restates a literal in worktree.Classify, which is not exported. Plan
// prints a due time from it, so a drift would print a time sweep still refuses.
// This pins the two to the same second.
func TestMinAgeMatchesClassify(t *testing.T) {
	at := func(age time.Duration) worktree.LifecycleState {
		info := worktree.Info{
			Metadata: &worktree.Metadata{CreatedAt: phaseNow.Add(-age), LeaseExpiresAt: phaseNow.Add(-time.Hour), ReapOK: true},
			PRMerged: true,
		}
		state, _ := worktree.Classify(info, phaseNow)
		return state
	}
	if got := at(minAge - time.Second); got != worktree.StateActive {
		t.Errorf("just under minAge classifies %s, want ACTIVE", got)
	}
	if got := at(minAge); got != worktree.StateReapable {
		t.Errorf("exactly minAge classifies %s, want REAPABLE", got)
	}
}

// The command an ambiguous action carries is pasted into a shell: a path with a
// space is quoted, and one no shell quotes alike is not given as a command at
// all, with the reason said instead (lesson 449).
func TestPlanQuotesTheResolvingCommand(t *testing.T) {
	spaced := row("fix/a", PhaseUnadopted, func(r *Row) { r.Path, r.Ticket = "/wt/My Worktrees/fix/a", 5 })
	if got := find(t, Plan([]Row{spaced}, nil, phaseNow), "fix/a").Resolve; got != `dotf worktree adopt "/wt/My Worktrees/fix/a"` {
		t.Errorf("adopt = %q, want the spaced path in double quotes", got)
	}
	stalled := row("fix/b", PhaseStalled, func(r *Row) { r.Path = "/wt/My Worktrees/fix/b" })
	if got := find(t, Plan([]Row{stalled}, nil, phaseNow), "fix/b").Resolve; got != `dotf worktree done --force "/wt/My Worktrees/fix/b"` {
		t.Errorf("done = %q, want the spaced path in double quotes", got)
	}
	live := row("fix/c", PhaseStalled, func(r *Row) { r.Path = "/wt/a$b/fix/c" })
	a := find(t, Plan([]Row{live}, nil, phaseNow), "fix/c")
	if a.Resolve != "" || !strings.Contains(a.Reason, "no shell quotes alike") {
		t.Errorf("unquotable path: resolve=%q reason=%q, want no command and the reason", a.Resolve, a.Reason)
	}
}

// The head's landing conditions come from the landing reader, for the head
// only, and a reader that cannot answer leaves the head unchecked: it never
// reads as clear.
func TestCheckHead(t *testing.T) {
	order := func() []Landing {
		return []Landing{{PR: 20, Branch: "fix/a", Head: true}, {PR: 30, Branch: "fix/b"}}
	}
	t.Run("reads the head and only the head", func(t *testing.T) {
		var asked []int
		o := order()
		CheckHead(o, func(pr int) ([]string, error) { asked = append(asked, pr); return []string{"test: pending"}, nil })
		if len(asked) != 1 || asked[0] != 20 {
			t.Errorf("asked %v, want only #20", asked)
		}
		if !o[0].Checked || len(o[0].Blockers) != 1 || o[1].Checked {
			t.Errorf("order = %+v", o)
		}
	})
	t.Run("a head with no blockers is checked and clear", func(t *testing.T) {
		o := order()
		CheckHead(o, func(int) ([]string, error) { return nil, nil })
		if !o[0].Checked || len(o[0].Blockers) != 0 {
			t.Errorf("head = %+v", o[0])
		}
	})
	t.Run("a reader that fails leaves the head unchecked, with the reason", func(t *testing.T) {
		o := order()
		CheckHead(o, func(int) ([]string, error) { return nil, errors.New("HTTP 502") })
		if o[0].Checked || len(o[0].Blockers) != 1 || !strings.Contains(o[0].Blockers[0], "HTTP 502") {
			t.Errorf("head = %+v, want unchecked and the error named", o[0])
		}
	})
	t.Run("no reader leaves the order as it was", func(t *testing.T) {
		o := order()
		CheckHead(o, nil)
		if o[0].Checked || o[0].Blockers != nil {
			t.Errorf("head = %+v", o[0])
		}
	})
}
