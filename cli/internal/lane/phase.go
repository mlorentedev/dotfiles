package lane

import (
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// Phase is where one lane stands. It is computed from git, the forge and the
// process table on every read and never stored: a wrong phase is fixed here
// once, not by rewriting state on every worktree (ADR-050 decision 1).
type Phase string

const (
	// PhaseNone is a row that is not a lane: the main checkout, or nothing
	// queued, on disk or merged.
	PhaseNone Phase = ""
	// PhasePending is a queued issue that has no worktree yet.
	PhasePending Phase = "Pending"
	// PhaseRunning is a worktree with a lease and work in progress: a session
	// is live, or the lease has not run out.
	PhaseRunning Phase = "Running"
	// PhaseReview is an open PR that is not first in the landing order.
	PhaseReview Phase = "Review"
	// PhaseLanding is the open PR first in the landing order.
	PhaseLanding Phase = "Landing"
	// PhaseLanded is a merged PR whose worktree is still on disk.
	PhaseLanded Phase = "Landed"
	// PhaseReaped is a merged PR whose worktree is gone.
	PhaseReaped Phase = "Reaped"
	// PhaseStalled is a lane nobody is working: no PR (or a closed one), an
	// expired lease and no live session.
	PhaseStalled Phase = "Stalled"
	// PhaseOrphan is a branch whose upstream is gone with its PR unmerged.
	PhaseOrphan Phase = "Orphan"
	// PhaseUnadopted is a linked worktree with no dotf lease.
	PhaseUnadopted Phase = "Unadopted"
)

// PRState is what the forge says about a branch's newest pull request.
type PRState string

const (
	PRNone   PRState = ""
	PROpen   PRState = "open"
	PRMerged PRState = "merged"
	PRClosed PRState = "closed" // closed without merging
)

// Facts is everything Derive reads. HerdrPresent is carried for display only:
// herdr is the cockpit and decides nothing (ADR-032, ADR-050 decision 3).
type Facts struct {
	Info   worktree.Info
	Exists bool // the worktree directory is on disk
	Queued bool // an open, unassigned issue of .dotf/lane.json
	PR     PRState
	// Head is set on the open PR first in the landing order. The order is the
	// PRs' own, lowest number first: it is FIFO by opening time, which is what
	// the landing queue is meant to be until ADR-049's queue replaces it.
	Head         bool
	Live         bool // a process or a recent session says someone is working
	HerdrPresent bool
}

// Derive is the phase of one lane. The precedence is the order of the cases:
// a lane with no lease is Unadopted whatever its PR says, because dotf has no
// intent recorded for it, and a merged PR outranks everything else because
// done is judged from git and the forge (ADR-050 decision 3).
func Derive(f Facts, now time.Time) Phase {
	if f.Exists && f.Info.IsMain {
		return PhaseNone
	}
	if !f.Exists {
		return deriveGone(f)
	}
	if f.Info.Metadata == nil {
		return PhaseUnadopted
	}
	switch {
	case f.PR == PRMerged:
		return PhaseLanded
	case f.Info.IsOrphan:
		return PhaseOrphan
	case f.PR == PROpen && f.Head:
		return PhaseLanding
	case f.PR == PROpen:
		return PhaseReview
	case !f.Live && !f.Info.Metadata.LeaseExpiresAt.After(now):
		return PhaseStalled
	}
	return PhaseRunning
}

// deriveGone covers a lane with no worktree on disk: finished, or not started.
func deriveGone(f Facts) Phase {
	switch {
	case f.PR == PRMerged:
		return PhaseReaped
	case f.Queued:
		return PhasePending
	}
	return PhaseNone
}
