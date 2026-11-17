// Package converge brings a machine to the state the checkout declares, through
// an ordered registry of reconcilers (ADR-045, #1843 B6/B7). The registry is
// code; what each reconciler converges is data the checkout carries.
package converge

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/platform"
)

// Env is what a reconciler needs to know about the machine it runs on.
type Env struct {
	RepoRoot  string // the dotfiles checkout
	Home      string
	DeployDir string // where the checkout's records are mirrored ($DOTFILES_DIR)
	GOOS      string
	Dotf      string // the running build's version
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
	// Opaque says the step cannot tell what it changed, or would change: a
	// script that only reports its exit status. The run reports it as opaque,
	// never as converged or changed, and Changes is not counted.
	Opaque bool
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

// Revisiter is a reconciler whose result depends on what a later one changes:
// the run applies it again right after that reconciler's apply changed the
// machine. records-harness revisits after tools, because it deploys the
// instruction files of the agents on PATH and tools installs agents (#2013
// D11). A plan never revisits: nothing it reports has been made yet.
type Revisiter interface {
	RevisitAfter() string // the reconciler whose change re-runs this one
}

// Status is the outcome of one reconciler in a run.
type Status int

const (
	StatusOK      Status = iota // already converged
	StatusChange                // changed, or would change under a plan
	StatusSkipped               // not applicable here, or not reached
	StatusFailed
	StatusOpaque // ran, or would run, but cannot say what it changed
)

func (s Status) String() string {
	return [...]string{"ok", "change", "skipped", "failed", "opaque"}[s]
}

// Entry is one reconciler's line in a report.
type Entry struct {
	Name    string
	Status  Status
	Changes int
	Detail  string
	// Duration is how long the step took; zero for one the run never
	// reached or a plan gate skipped.
	Duration time.Duration
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

// Observer hears a run as it happens (CLI-114): Start when a step begins to
// run, Done with its entry once that entry is final. A step the OS does not
// support, and one a failed step kept from running ("not reached"), are done
// without a start, so Done is heard in the report's order. The entries a
// plan's gate appends are never done: they are the tail of the returned
// report, for the caller to print.
type Observer interface {
	Start(name string)
	Done(e Entry)
}

type nopObserver struct{}

func (nopObserver) Start(string) {}
func (nopObserver) Done(Entry)   {}

// Run drives the reconcilers in order. A reconciler that does not list
// env.GOOS is skipped, never passed. A failure does not stop the run: it keeps
// from running the reconcilers that need the one that failed, directly or
// through another, and those are reported as not reached with the cause. The
// rest still run, in order (ADR-041 decision 4 as amended by ADR-064). The
// returned error joins every failure, each naming its reconciler.
func Run(reconcilers []Reconciler, env Env, dryRun bool) (Report, error) {
	return RunObserved(reconcilers, env, dryRun, nopObserver{})
}

// RunObserved is Run, telling obs about each step as it runs.
func RunObserved(reconcilers []Reconciler, env Env, dryRun bool, obs Observer) (Report, error) {
	rep := Report{DryRun: dryRun, GOOS: env.GOOS}
	var failures []error
	var ran []Reconciler
	down := downSet{}
	gate := ""
	for i, r := range reconcilers {
		if gate != "" {
			rep.Entries = append(rep.Entries, Entry{Name: r.Name(), Status: StatusSkipped, Detail: gate})
			continue
		}
		if why := down.blocks(needsOf(reconcilers, i)); why != "" {
			down[r.Name()] = why
			entry := Entry{Name: r.Name(), Status: StatusSkipped, Detail: NotReached + ": " + why}
			obs.Done(entry)
			rep.Entries = append(rep.Entries, entry)
			continue
		}
		entry, g, err := runOne(r, env, dryRun, obs)
		obs.Done(entry)
		rep.Entries = append(rep.Entries, entry)
		if err != nil {
			down[r.Name()] = r.Name() + " failed"
			failures = append(failures, fmt.Errorf("converge: %s: %w", r.Name(), err))
			continue
		}
		if dryRun && g != "" {
			gate = g
		}
		ran = append(ran, r)
		if !dryRun && entry.Status == StatusChange {
			failures = append(failures, revisit(&rep, ran, down, r.Name(), env, obs)...)
		}
	}
	return rep, errors.Join(failures...)
}

// revisit applies again each reconciler that ran without failing and revisits
// after the one named, in registry order, and appends their entries. One that
// fails is down from then on, like a step of the run's own loop, and the
// others are still revisited.
func revisit(rep *Report, ran []Reconciler, down downSet, after string, env Env, obs Observer) []error {
	var failures []error
	for _, q := range ran {
		if rv, ok := q.(Revisiter); !ok || rv.RevisitAfter() != after || !platform.Supports(q.Platforms(), env.GOOS) {
			continue
		}
		if _, isDown := down[q.Name()]; isDown {
			continue
		}
		entry, _, err := runOne(q, env, false, obs)
		entry.Detail = "after " + after + ": " + entry.Detail
		obs.Done(entry)
		rep.Entries = append(rep.Entries, entry)
		if err != nil {
			down[q.Name()] = q.Name() + " failed"
			failures = append(failures, fmt.Errorf("converge: %s: %w", q.Name(), err))
		}
	}
	return failures
}

// now is the clock runOne times a step by; tests replace it.
var now = time.Now

// runOne runs one reconciler and records how long it took (CLI-114 AC4).
// A step this OS does not support never starts, so obs hears no Start for it.
func runOne(r Reconciler, env Env, dryRun bool, obs Observer) (Entry, string, error) {
	if !platform.Supports(r.Platforms(), env.GOOS) {
		return Entry{Name: r.Name(), Status: StatusSkipped, Detail: "not supported on " + env.GOOS}, "", nil
	}
	obs.Start(r.Name())
	start := now()
	e, gate, err := runStep(r, env, dryRun)
	// Millisecond resolution, rounded once here so the console and the JSON
	// agree, and never under 1ms: duration 0 means the step never ran.
	e.Duration = max(now().Sub(start).Round(time.Millisecond), time.Millisecond)
	return e, gate, err
}

func runStep(r Reconciler, env Env, dryRun bool) (Entry, string, error) {
	e := Entry{Name: r.Name()}
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
	case err != nil && res.Changes > 0 && res.Detail != "":
		// What the step changed before failing stays on its line (BUG-136).
		e.Status, e.Detail = StatusFailed, res.Detail+"; failed: "+err.Error()
	case err != nil:
		e.Status, e.Detail = StatusFailed, err.Error()
	case res.Opaque:
		e.Status, e.Changes = StatusOpaque, 0
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
