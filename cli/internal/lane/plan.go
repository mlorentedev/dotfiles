package lane

import (
	"fmt"
	"sort"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// Kind is what an action does.
type Kind string

const (
	KindReap  Kind = "reap"  // remove a worktree whose PR merged
	KindStart Kind = "start" // start the next queued item
	KindAdopt Kind = "adopt" // give a worktree a lease
	KindFlag  Kind = "flag"  // report a lane that needs a human, with no verb to run
)

// Class is who may run an action (decision D1).
type Class string

const (
	// ClassAuto is an action an existing verb accepts without --force.
	ClassAuto Class = "auto"
	// ClassPending is one whose only missing gate is time. plan says when it
	// is due and asks nothing.
	ClassPending Class = "pending"
	// ClassAmbiguous is everything else: it carries the one command that
	// resolves it, so the human's part is a yes or a no.
	ClassAmbiguous Class = "ambiguous"
)

// Action is one step a reconcile would take.
type Action struct {
	Kind   Kind      `json:"kind"`
	Class  Class     `json:"class"`
	Branch string    `json:"branch"`
	Path   string    `json:"path,omitempty"`
	Ticket int       `json:"ticket,omitempty"`
	Reason string    `json:"reason"`
	Due    time.Time `json:"due,omitzero"`
	// Resolve is the exact command that settles an ambiguous action. It is
	// empty when the controller cannot derive an intent and proposes nothing.
	Resolve string `json:"resolve,omitempty"`
}

// minAge is how old a worktree must be before sweep may reap it, as
// worktree.Classify holds it.
const minAge = 15 * time.Minute

// Plan is what a reconcile would do over rows. It reads and mutates nothing:
// the verbs that act on it are the existing ones, and only auto actions are
// ever run (decision D1). Landing is never an action (D3, see LandingOrder),
// and adoption is never auto (D2): adopting writes ReapOK, which starts the
// clock towards a reap.
func Plan(rows []Row, queue []int, now time.Time) []Action {
	var acts []Action
	for _, r := range rows {
		switch r.Phase {
		case PhaseLanded:
			acts = append(acts, planLanded(r, now))
		case PhaseStalled, PhaseOrphan:
			acts = append(acts, abandon(r, string(r.Phase)+": "+phaseWhy(r.Phase)))
		case PhaseUnadopted:
			acts = append(acts, planUnadopted(r))
		}
	}
	if a, ok := planStart(rows, queue); ok {
		acts = append(acts, a)
	}
	return acts
}

func phaseWhy(p Phase) string {
	if p == PhaseOrphan {
		return "the upstream branch is gone and its PR is not merged"
	}
	return "no PR, an expired lease and no live session"
}

// abandon flags a lane only a human can judge. The command it carries removes
// the worktree, so the yes is an explicit one.
func abandon(r Row, why string) Action {
	return Action{Kind: KindFlag, Class: ClassAmbiguous, Branch: r.Branch, Path: r.Path, Ticket: r.Ticket, Reason: why,
		Resolve: "dotf worktree done --force " + r.Path}
}

// planLanded mirrors sweep's gates through the state Classify already
// computed: REAPABLE is what sweep reaps, and the time-only reasons for ACTIVE
// are the pending ones. Gate f is the rest: anyone who might be inside, or
// whose liveness could not be read, makes it ambiguous.
func planLanded(r Row, now time.Time) Action {
	a := Action{Kind: KindReap, Branch: r.Branch, Path: r.Path, Ticket: r.Ticket}
	switch {
	case r.Live != LiveIdle:
		// Before any time-only reason: a pending reap would read "wait, and
		// it goes", and a worktree someone may be inside must not read so.
		return abandon(r, "PR merged, but liveness is "+string(r.Live)+": someone may be inside")
	case r.State == worktree.StateReapable:
		a.Class, a.Reason = ClassAuto, "PR merged, clean tree, lease expired, nobody inside"
		return a
	case r.State == worktree.StateActive && r.Meta != nil && r.Meta.ReapOK:
		if due := reapDue(r.Meta); due.After(now) {
			a.Class, a.Due, a.Reason = ClassPending, due, "PR merged; waiting for the lease and the minimum age"
			return a
		}
	}
	why := string(r.State)
	if r.StateReason != "" {
		why += " (" + r.StateReason + ")"
	}
	return abandon(r, "PR merged, but sweep will not reap it: "+why)
}

// reapDue is when the time-only gates have both passed.
func reapDue(m *worktree.Metadata) time.Time {
	due := m.CreatedAt.Add(minAge)
	if m.LeaseExpiresAt.After(due) {
		due = m.LeaseExpiresAt
	}
	return due
}

// planUnadopted proposes adoption only when the branch or a PR makes the
// intent derivable; otherwise it reports the worktree and proposes nothing,
// because the controller records what git says and never invents intent.
func planUnadopted(r Row) Action {
	a := Action{Kind: KindFlag, Class: ClassAmbiguous, Branch: r.Branch, Path: r.Path, Ticket: r.Ticket, Reason: "no lease, and no intent to adopt it under"}
	if r.Ticket != 0 || r.PR != 0 {
		a.Kind, a.Reason = KindAdopt, "no lease; the branch or its PR names the intent"
		a.Resolve = "dotf worktree adopt " + r.Path
	}
	return a
}

// planStart starts the first pending item in queue order, and only while no
// lane is in flight: a second apply must find the lane the first one started
// and do nothing (changed=0), not start the next item.
func planStart(rows []Row, queue []int) (Action, bool) {
	for _, r := range rows {
		if r.Phase == PhaseRunning || r.Phase == PhaseReview || r.Phase == PhaseLanding {
			return Action{}, false
		}
	}
	for _, n := range queue {
		for _, r := range rows {
			if r.Phase == PhasePending && r.Ticket == n {
				return Action{Kind: KindStart, Class: ClassAuto, Branch: r.Branch, Ticket: n, Reason: fmt.Sprintf("first open item of the queue, #%d, and no lane is in flight", n)}, true
			}
		}
	}
	return Action{}, false
}

// Landing is one open PR in the landing order.
type Landing struct {
	PR     int    `json:"pr"`
	Branch string `json:"branch"`
	Head   bool   `json:"head"`
	// Command is what lands the head, run by the lane that opened the PR or by
	// the owner (decision D3). It is empty for the rest.
	Command string `json:"command,omitempty"`
}

// LandingOrder lists the open PRs of the lanes in Review or Landing, lowest
// number first. It never produces an action: landing under the standing grant
// covers a PR the agent opened, and `apply` run from main would widen it.
func LandingOrder(rows []Row) []Landing {
	var order []Landing
	for _, r := range rows {
		if r.Phase != PhaseLanding && r.Phase != PhaseReview {
			continue
		}
		l := Landing{PR: r.PR, Branch: r.Branch, Head: r.Phase == PhaseLanding}
		if l.Head {
			l.Command = fmt.Sprintf("dotf pr triage-queue --pr %d && dotf pr land %d", r.PR, r.PR)
		}
		order = append(order, l)
	}
	sort.Slice(order, func(a, b int) bool { return order[a].PR < order[b].PR })
	return order
}
