package lane

import "fmt"

// Verbs are the existing commands apply is allowed to run. Apply holds no
// other way to change anything, so what it can do is what these two can.
type Verbs struct {
	// Sweep reaps what worktree sweep accepts, under its own fail-closed
	// gates, and returns how many it removed.
	Sweep func() (reaped int, err error)
	// Next starts the next queued item and says whether an agent was started.
	Next func() (started bool, err error)
}

// Applied is what one apply changed.
type Applied struct {
	Changed int `json:"changed"`
}

// Apply runs the auto actions of a plan, and nothing else (decision D1).
// Adoption (D2) and landing (D3) are never actions it can run, and a pending
// or ambiguous action is left to the time or the human it names.
//
// Reaps go through one sweep, not one removal per action: sweep re-derives
// what is reapable under its own gates, so the plan only decides whether to
// call it. What changed is what sweep removed. A start counts only when an
// agent was started, because a start that printed commands changed nothing, and
// a second apply must report changed=0.
func Apply(actions []Action, v Verbs) (Applied, error) {
	var out Applied
	var reap, start bool
	for _, a := range actions {
		if a.Class != ClassAuto {
			continue
		}
		switch a.Kind {
		case KindReap:
			reap = true
		case KindStart:
			start = true
		}
	}
	if reap {
		n, err := v.Sweep()
		out.Changed += n
		if err != nil {
			return out, fmt.Errorf("apply: sweep: %w", err)
		}
	}
	if start {
		started, err := v.Next()
		if started {
			out.Changed++
		}
		if err != nil {
			return out, fmt.Errorf("apply: next: %w", err)
		}
	}
	return out, nil
}
