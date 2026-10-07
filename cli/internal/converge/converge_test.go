package converge

import (
	"errors"
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
	return Result{Changes: f.changes, Detail: f.name + " detail"}, f.err
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
