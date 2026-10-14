package converge

import (
	"errors"
	"strings"
	"testing"
)

// recorder is an Observer that logs what it is told, in order.
type recorder struct{ log []string }

func (r *recorder) Start(name string) { r.log = append(r.log, "start "+name) }
func (r *recorder) Done(e Entry)      { r.log = append(r.log, "done "+e.Name+": "+e.Detail) }

// CLI-114 AC5: the observer hears each step that runs, start then done, in
// run order and with its final entry (a revisit's "after" prefix included).
// A step the OS does not support is done without a start: nothing ran. The
// entries the run never reached are never done; the renderer flushes them.
func TestRunObserved_ReportsEachStepItRuns(t *testing.T) {
	harness := revisiting{&fake{name: "harness"}, "tools"}
	win := &fake{name: "win", platforms: []string{"windows"}}
	tools := &fake{name: "tools", changes: 1}
	broken := &fake{name: "broken", err: errors.New("boom")}
	later := &fake{name: "later"}

	var obs recorder
	rep, err := RunObserved([]Reconciler{harness, win, tools, broken, later}, Env{GOOS: "linux"}, false, &obs)
	if err == nil {
		t.Fatal("want the broken step's error")
	}
	want := []string{
		"start harness", "done harness: harness detail",
		"done win: not supported on linux",
		"start tools", "done tools: tools detail",
		"start harness", "done harness: after tools: harness detail",
		"start broken", "done broken: boom",
	}
	if got := strings.Join(obs.log, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("observer heard:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	if n := len(rep.Entries); n != 6 || rep.Entries[5].Detail != "not reached" {
		t.Errorf("entries %d, last %+v: want the unreached step in the report", n, rep.Entries[n-1])
	}
}

// A plan's gate skips the steps after it without running them: they are never
// done either.
func TestRunObserved_AGateSkipsWithoutRunning(t *testing.T) {
	var obs recorder
	_, err := RunObserved([]Reconciler{&fake{name: "checkout", changes: 1, gate: "no checkout yet"}, &fake{name: "later"}}, Env{GOOS: "linux"}, true, &obs)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(obs.log, "|"); got != "start checkout|done checkout: checkout detail" {
		t.Errorf("observer heard %q, want only the step that ran", got)
	}
}
