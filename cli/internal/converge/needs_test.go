package converge

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// needing is a fake that declares its needs.
type needing struct {
	*fake
	needs []string
}

func (n needing) Needs() []string { return n.needs }

func details(rep Report) map[string]string {
	out := map[string]string{}
	for _, e := range rep.Entries {
		out[e.Name] = e.Status.String() + ": " + e.Detail
	}
	return out
}

// BUG-173: a failed step keeps from running only the steps that need it,
// directly or through another. The rest run, and the run still fails.
func TestRun_AFailureHoldsBackOnlyItsDependents(t *testing.T) {
	root := needing{&fake{name: "root"}, nil}
	leaf := needing{&fake{name: "leaf", err: errors.New("boom")}, []string{"root"}}
	other := needing{&fake{name: "other", changes: 1}, []string{"root"}}
	child := needing{&fake{name: "child"}, []string{"leaf"}}
	grandchild := needing{&fake{name: "grandchild"}, []string{"other", "child"}}

	rep, err := Run([]Reconciler{root, leaf, other, child, grandchild}, Env{GOOS: "linux"}, false)
	if err == nil || !strings.Contains(err.Error(), "converge: leaf: boom") {
		t.Fatalf("err = %v, want the leaf's failure: a step that changed after it must not erase it", err)
	}
	if !other.applied || !other.probed {
		t.Error("other does not need the failed step: want it applied and probed")
	}
	if child.applied || grandchild.applied {
		t.Error("child and grandchild need the failed step: want neither applied")
	}
	got := details(rep)
	for name, want := range map[string]string{
		"leaf":       "failed: boom",
		"other":      "change: other detail",
		"child":      "skipped: not reached: leaf failed",
		"grandchild": "skipped: not reached: needs child, leaf failed",
	} {
		if got[name] != want {
			t.Errorf("%s = %q, want %q", name, got[name], want)
		}
	}
}

// Every failure is in the returned error, not only the first.
func TestRun_JoinsEveryFailure(t *testing.T) {
	a := needing{&fake{name: "a", err: errors.New("first")}, nil}
	b := needing{&fake{name: "b", probeErr: errors.New("second")}, nil}

	_, err := Run([]Reconciler{a, b}, Env{GOOS: "linux"}, false)
	if err == nil || !strings.Contains(err.Error(), "converge: a: first") || !strings.Contains(err.Error(), "converge: b: post-condition: second") {
		t.Fatalf("err = %v, want both failures named", err)
	}
}

// A reconciler that declares nothing needs every one before it, which is what
// the run did before steps could say otherwise.
func TestRun_AnUndeclaredStepNeedsEveryStepBeforeIt(t *testing.T) {
	broken := &fake{name: "broken", err: errors.New("boom")}
	later := &fake{name: "later"}
	last := &fake{name: "last"}

	rep, err := Run([]Reconciler{broken, later, last}, Env{GOOS: "linux"}, false)
	if err == nil {
		t.Fatal("want the failure")
	}
	if later.applied || last.applied {
		t.Error("undeclared steps after a failure must not run")
	}
	if got := details(rep)["last"]; got != "skipped: not reached: broken failed" {
		t.Errorf("last = %q, want it not reached for the step that failed", got)
	}
}

// A need that is not in the run (--only), that skipped, or that this OS does
// not support holds nothing back.
func TestRun_AnAbsentOrSkippedNeedDoesNotBlock(t *testing.T) {
	skipped := needing{&fake{name: "skipped", skip: "nothing to do here"}, nil}
	foreign := needing{&fake{name: "foreign", platforms: []string{"windows"}}, nil}
	step := needing{&fake{name: "step"}, []string{"not-selected", "skipped", "foreign"}}

	_, err := Run([]Reconciler{skipped, foreign, step}, Env{GOOS: "linux"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !step.applied {
		t.Error("want the step applied: none of its needs failed")
	}
}

// A step that failed is not applied again when a later step changes, and a
// failed revisit does not stop the steps that do not need it.
func TestRun_RevisitSkipsAFailedStepAndItsFailureIsNotTerminal(t *testing.T) {
	failed := revisitingNeeder{&fake{name: "failed", err: errors.New("boom")}, "tools"}
	tools := needing{&fake{name: "tools", changes: 1}, nil}
	after := needing{&fake{name: "after"}, nil}

	_, err := Run([]Reconciler{failed, tools, after}, Env{GOOS: "linux"}, false)
	if err == nil {
		t.Fatal("want the failure")
	}
	if failed.applies != 1 {
		t.Errorf("failed step applied %d times, want once: a step that failed is not revisited", failed.applies)
	}
	if !after.applied {
		t.Error("want the step after tools applied")
	}
}

func TestRun_ARevisitThatFailsHoldsBackItsDependents(t *testing.T) {
	harness := &fake{name: "harness"}
	tools := needing{&fake{name: "tools", changes: 1}, nil}
	needsHarness := needing{&fake{name: "needs-harness"}, []string{"harness"}}
	free := needing{&fake{name: "free"}, nil}
	reg := []Reconciler{revisitingNeeder{harness, "tools"}, tools, needsHarness, free}

	// The first apply passes; the revisit after tools fails.
	harness.failFrom = 2
	_, err := Run(reg, Env{GOOS: "linux"}, false)
	if err == nil || !strings.Contains(err.Error(), "converge: harness:") {
		t.Fatalf("err = %v, want the revisit's failure", err)
	}
	if needsHarness.applied {
		t.Error("a step that needs the failed revisit must not run")
	}
	if !free.applied {
		t.Error("a step that does not need it must run")
	}
}

type revisitingNeeder struct {
	*fake
	after string
}

func (r revisitingNeeder) RevisitAfter() string { return r.after }
func (r revisitingNeeder) Needs() []string      { return nil }

// Unreached reads the same graph the run reads, without running anything.
func TestUnreached(t *testing.T) {
	reg := []Reconciler{
		needing{&fake{name: "a"}, nil},
		needing{&fake{name: "b"}, []string{"a"}},
		needing{&fake{name: "c"}, []string{"b"}},
		needing{&fake{name: "d"}, nil},
	}
	if got := Unreached(reg, "a"); !slices.Equal(got, []string{"b", "c"}) {
		t.Errorf("Unreached(a) = %v, want [b c]", got)
	}
	if got := Unreached(reg, "d"); len(got) != 0 {
		t.Errorf("Unreached(d) = %v, want none", got)
	}
}
