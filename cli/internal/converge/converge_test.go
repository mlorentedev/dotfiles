package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fake records how the runner drove it, so a test can assert that a plan never
// applied and never probed.
type fake struct {
	name      string
	platforms []string
	changes   int
	err       error
	probeErr  error
	gate      string
	skip      string

	applied, planned, probed bool
}

func (f *fake) Name() string        { return f.name }
func (f *fake) Platforms() []string { return f.platforms }

func (f *fake) Reconcile(_ Env, dryRun bool) (Result, error) {
	if dryRun {
		f.planned = true
	} else {
		f.applied = true
	}
	if f.skip != "" {
		return Result{Skip: f.skip}, f.err
	}
	return Result{Changes: f.changes, Detail: f.name + " detail", Gate: f.gate}, f.err
}

func (f *fake) Probe(Env) error {
	f.probed = true
	return f.probeErr
}

func TestRun_PlanReportsEveryReconcilerAndAppliesNothing(t *testing.T) {
	a, b := &fake{name: "a", changes: 3}, &fake{name: "b"}

	rep, err := Run([]Reconciler{a, b}, Env{GOOS: "linux"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if a.applied || b.applied || a.probed || b.probed {
		t.Fatal("a plan applied or probed a reconciler")
	}
	if got := statuses(rep); got != "a=change b=ok" {
		t.Errorf("statuses: %s", got)
	}
	if rep.Entries[0].Changes != 3 {
		t.Errorf("changes: want 3, got %d", rep.Entries[0].Changes)
	}
}

// A checkout still to clone holds back every later plan: they would read files
// that do not exist yet. An apply makes the change first, so it runs them all.
func TestRun_GateHoldsBackLaterPlansOnly(t *testing.T) {
	gate := func() (*fake, *fake) {
		return &fake{name: "checkout", changes: 1, gate: "waits for checkout"}, &fake{name: "records"}
	}

	co, records := gate()
	rep, err := Run([]Reconciler{co, records}, Env{GOOS: "linux"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if records.planned {
		t.Error("a plan behind a gate must not run")
	}
	if got := statuses(rep); got != "checkout=change records=skipped" || rep.Entries[1].Detail != "waits for checkout" {
		t.Errorf("plan: %s, %q", got, rep.Entries[1].Detail)
	}

	co, records = gate()
	if _, err := Run([]Reconciler{co, records}, Env{GOOS: "linux"}, false); err != nil {
		t.Fatal(err)
	}
	if !records.applied {
		t.Error("an apply must run past a gate: the change it waits for is made")
	}
}

func TestRun_UnlistedPlatformIsSkippedNotPassed(t *testing.T) {
	linuxOnly := &fake{name: "legacy", platforms: []string{"linux", "windows"}}

	rep, err := Run([]Reconciler{linuxOnly}, Env{GOOS: "darwin"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if linuxOnly.applied || linuxOnly.planned {
		t.Fatal("a reconciler ran on a platform it does not list")
	}
	e := rep.Entries[0]
	if e.Status != StatusSkipped || !strings.Contains(e.Detail, "darwin") {
		t.Errorf("want skipped naming darwin, got %s %q", e.Status, e.Detail)
	}
}

// A step that skips itself has nothing to verify: probing it would fail the
// run on a precondition the skip already reported.
func TestRun_ASkippedStepIsNeverProbed(t *testing.T) {
	unwired := &fake{name: "records-bind", skip: "no resolver", probeErr: errors.New("no resolver")}

	rep, err := Run([]Reconciler{unwired}, Env{GOOS: "linux"}, false)
	if err != nil {
		t.Fatalf("a skipped step failed the run: %v", err)
	}
	if unwired.probed {
		t.Error("the runner probed a step that skipped itself")
	}
	if e := rep.Entries[0]; e.Status != StatusSkipped || e.Detail != "no resolver" {
		t.Errorf("want skipped with the reason, got %s %q", e.Status, e.Detail)
	}
}

func TestRun_FailedProbeFailsTheRunNamingTheReconciler(t *testing.T) {
	bad := &fake{name: "records", probeErr: errors.New("CLAUDE.md absent")}
	after := &fake{name: "tools"}

	rep, err := Run([]Reconciler{bad, after}, Env{GOOS: "linux"}, false)
	if err == nil || !strings.Contains(err.Error(), "records") {
		t.Fatalf("want an error naming records, got %v", err)
	}
	if after.applied {
		t.Error("a reconciler after a failure still ran")
	}
	if got := statuses(rep); got != "records=failed tools=skipped" {
		t.Errorf("statuses: %s", got)
	}
}

func TestRun_ReconcileErrorFailsTheRun(t *testing.T) {
	bad := &fake{name: "records", err: errors.New("disk full")}

	_, err := Run([]Reconciler{bad}, Env{GOOS: "linux"}, true)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("want the reconcile error, got %v", err)
	}
	if bad.probed {
		t.Error("a failed reconcile was probed")
	}
}

func statuses(rep Report) string {
	parts := make([]string, 0, len(rep.Entries))
	for _, e := range rep.Entries {
		parts = append(parts, e.Name+"="+e.Status.String())
	}
	return strings.Join(parts, " ")
}

func TestWriteReport_RecordsTheRunAndItsOutcome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "converge", "last.json")
	rep := Report{GOOS: "darwin", Entries: []Entry{
		{Name: "records-mirror", Status: StatusChange, Changes: 3, Detail: "d"},
		{Name: "records-harness", Status: StatusFailed, Detail: "boom"},
	}}

	if err := WriteReport(path, rep, errors.New("converge: records-harness: boom")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"result": "failed"`, `"error": "converge: records-harness: boom"`, `"goos": "darwin"`, `"changed": 3`, `"status": "change"`, `"status": "failed"`, `"finished_at"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("report lacks %s:\n%s", want, raw)
		}
	}
}
