// Package converge brings a machine to the state the checkout declares, through
// an ordered registry of reconcilers (ADR-045, #1843 B6/B7). The registry is
// code; what each reconciler converges is data the checkout carries.
package converge

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/platform"
)

// Env is what a reconciler needs to know about the machine it runs on.
type Env struct {
	RepoRoot  string // the dotfiles checkout
	Home      string
	DeployDir string // where the checkout's records are mirrored ($DOTFILES_DIR)
	GOOS      string
}

// Result is what one Reconcile call did, or under a plan would do.
type Result struct {
	Changes int    // 0 means converged
	Detail  string // one line for the report
	// Skip, when set, says why this reconciler does not apply on this machine
	// right now (a prerequisite it cannot provide itself). The run reports it
	// as skipped with this reason, never as passed, and does not probe it.
	Skip string
	// Gate, under a plan, says the reconcilers after this one read what this
	// change has not made yet (a checkout still to clone). They are reported
	// as skipped with this reason instead of failing on files that do not
	// exist. An apply ignores it: the change is made before they run.
	Gate string
}

// Reconciler converges one part of the machine.
//
// Reconcile must take the same path under dryRun as without it and stop only
// before writing, so a plan and an apply cannot disagree. Probe is the
// post-condition an apply must pass before the run reports success; a step
// whose failure could otherwise read as success is the defect this package
// exists to remove (lesson 337).
type Reconciler interface {
	Name() string
	Platforms() []string // absent means every OS (package platform)
	Reconcile(env Env, dryRun bool) (Result, error)
	Probe(env Env) error
}

// Status is the outcome of one reconciler in a run.
type Status int

const (
	StatusOK      Status = iota // already converged
	StatusChange                // changed, or would change under a plan
	StatusSkipped               // not applicable here, or not reached
	StatusFailed
)

func (s Status) String() string {
	return [...]string{"ok", "change", "skipped", "failed"}[s]
}

// Entry is one reconciler's line in a report.
type Entry struct {
	Name    string
	Status  Status
	Changes int
	Detail  string
}

// Report is the outcome of a run, one entry per registered reconciler.
type Report struct {
	DryRun  bool
	GOOS    string
	Entries []Entry
}

// Select keeps the named reconcilers, in registry order. An unknown name is an
// error that lists the known ones, so a typo cannot select nothing and pass.
func Select(reconcilers []Reconciler, names []string) ([]Reconciler, error) {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var out []Reconciler
	known := make([]string, 0, len(reconcilers))
	for _, r := range reconcilers {
		known = append(known, r.Name())
		if want[r.Name()] {
			out = append(out, r)
			delete(want, r.Name())
		}
	}
	if len(want) > 0 {
		unknown := make([]string, 0, len(want))
		for n := range want {
			unknown = append(unknown, n)
		}
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown reconciler(s) %s (known: %s)", strings.Join(unknown, ", "), strings.Join(known, ", "))
	}
	return out, nil
}

// Run drives the reconcilers in order. A reconciler that does not list
// env.GOOS is skipped, never passed. The first failure stops the run: the
// reconcilers after it are reported as not reached, because a later step may
// depend on the one that failed (ADR-041 decision 4). The returned error names
// the reconciler that failed.
func Run(reconcilers []Reconciler, env Env, dryRun bool) (Report, error) {
	rep := Report{DryRun: dryRun, GOOS: env.GOOS}
	var failed error
	gate := ""
	for _, r := range reconcilers {
		switch {
		case failed != nil:
			rep.Entries = append(rep.Entries, Entry{Name: r.Name(), Status: StatusSkipped, Detail: "not reached"})
			continue
		case gate != "":
			rep.Entries = append(rep.Entries, Entry{Name: r.Name(), Status: StatusSkipped, Detail: gate})
			continue
		}
		entry, g, err := runOne(r, env, dryRun)
		rep.Entries = append(rep.Entries, entry)
		if err != nil {
			failed = fmt.Errorf("converge: %s: %w", r.Name(), err)
		}
		if dryRun && g != "" {
			gate = g
		}
	}
	return rep, failed
}

func runOne(r Reconciler, env Env, dryRun bool) (Entry, string, error) {
	e := Entry{Name: r.Name()}
	if !platform.Supports(r.Platforms(), env.GOOS) {
		e.Status, e.Detail = StatusSkipped, "not supported on "+env.GOOS
		return e, "", nil
	}
	res, err := r.Reconcile(env, dryRun)
	if err == nil && res.Skip != "" {
		e.Status, e.Detail = StatusSkipped, res.Skip
		return e, "", nil
	}
	e.Changes, e.Detail = res.Changes, res.Detail
	if err == nil && !dryRun {
		if perr := r.Probe(env); perr != nil {
			err = fmt.Errorf("post-condition: %w", perr)
		}
	}
	switch {
	case err != nil:
		e.Status, e.Detail = StatusFailed, err.Error()
	case res.Changes > 0:
		e.Status = StatusChange
	default:
		e.Status = StatusOK
	}
	return e, res.Gate, err
}

// Summary counts the entries by status, for the report's last line.
func (r Report) Summary() map[Status]int {
	n := map[Status]int{}
	for _, e := range r.Entries {
		n[e.Status]++
	}
	return n
}
