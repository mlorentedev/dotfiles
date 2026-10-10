package converge

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeSetup records the calls a legacy step makes, without running a script.
type fakeSetup struct {
	calls int
	env   Env
	err   error
}

func (f *fakeSetup) run(e Env) error {
	f.calls++
	f.env = e
	return f.err
}

// The setup scripts cannot plan, so a plan reports them as opaque, never as
// "no change", and never runs them.
func TestLegacySetup_PlanIsOpaqueAndRunsNothing(t *testing.T) {
	f := &fakeSetup{}
	rep, err := Run([]Reconciler{legacySetup{run: f.run}}, Env{RepoRoot: "/repo", GOOS: "linux"}, true)
	if err != nil {
		t.Fatal(err)
	}
	e := rep.Entries[0]
	if e.Status != StatusOpaque || !strings.Contains(e.Detail, "setup-linux.sh") {
		t.Errorf("plan entry = %+v, want opaque naming setup-linux.sh", e)
	}
	if f.calls != 0 {
		t.Error("a plan must not run the setup script")
	}
}

// An apply runs the OS's setup script once, after every native step, and
// reports it opaque: its exit status is all it says about what it changed.
func TestLegacySetup_ApplyRunsTheScriptAndStaysOpaque(t *testing.T) {
	f := &fakeSetup{}
	env := Env{RepoRoot: "/repo", GOOS: "windows"}
	rep, err := Run([]Reconciler{legacySetup{run: f.run}}, env, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.calls != 1 || f.env != env {
		t.Fatalf("setup ran %d time(s) with %+v, want once with %+v", f.calls, f.env, env)
	}
	if e := rep.Entries[0]; e.Status != StatusOpaque || e.Changes != 0 || !strings.Contains(e.Detail, "setup-windows.ps1") {
		t.Errorf("apply entry = %+v, want opaque, no counted changes, naming setup-windows.ps1", e)
	}
}

func TestLegacySetup_AFailingScriptFailsTheRun(t *testing.T) {
	f := &fakeSetup{err: errors.New("exit status 1")}
	_, err := Run([]Reconciler{legacySetup{run: f.run}}, Env{GOOS: "linux"}, false)
	if err == nil || !strings.Contains(err.Error(), "legacy-setup") {
		t.Fatalf("want a failure naming legacy-setup, got %v", err)
	}
}

// macOS has no setup script to run (ADR-045): the step is reported skipped
// with the OS named, never passed.
func TestLegacySetup_SkippedOnDarwin(t *testing.T) {
	f := &fakeSetup{}
	rep, _ := Run([]Reconciler{legacySetup{run: f.run}}, Env{GOOS: "darwin"}, false)
	if e := rep.Entries[0]; e.Status != StatusSkipped || !strings.Contains(e.Detail, "darwin") || f.calls != 0 {
		t.Errorf("darwin entry = %+v (ran %d), want skipped naming darwin", e, f.calls)
	}
}

// A setup script that calls `dotf converge` must not start another setup run:
// the step skips itself inside a run it started.
func TestLegacySetup_SkipsInsideASetupRunItStarted(t *testing.T) {
	t.Setenv(LegacySetupGuardEnv, "1")
	f := &fakeSetup{}
	rep, err := Run([]Reconciler{legacySetup{run: f.run}}, Env{GOOS: "linux"}, false)
	if err != nil || rep.Entries[0].Status != StatusSkipped || f.calls != 0 {
		t.Fatalf("got %+v, %v (ran %d); want skipped, not run", rep.Entries[0], err, f.calls)
	}
}

func TestLegacySetup_SkippedWithoutARunner(t *testing.T) {
	rep, err := Run([]Reconciler{legacySetup{}}, Env{GOOS: "linux"}, false)
	if err != nil || rep.Entries[0].Status != StatusSkipped {
		t.Fatalf("got %+v, %v; want skipped", rep.Entries[0], err)
	}
}

// ExecSetup is the production runner: it runs the checkout's script (or the
// DOTFILES_SELFUPDATE_SETUP_CMD override) from the checkout, and marks the
// child so a nested converge skips its own legacy step.
func TestExecSetup_RunsTheOverrideMarkedAndFromTheCheckout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("drives a POSIX script")
	}
	repo, out := t.TempDir(), filepath.Join(t.TempDir(), "out")
	script := filepath.Join(t.TempDir(), "setup.sh")
	body := "#!/bin/sh\nprintf '%s %s' \"$" + LegacySetupGuardEnv + "\" \"$PWD\" > " + out + "\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_SELFUPDATE_SETUP_CMD", script)

	if err := ExecSetup(Env{RepoRoot: repo, GOOS: runtime.GOOS}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	wantDir, _ := filepath.EvalSymlinks(repo)
	fields := strings.Fields(string(got))
	if len(fields) != 2 || fields[0] != "1" {
		t.Fatalf("child saw %q, want the guard set", got)
	}
	if gotDir, _ := filepath.EvalSymlinks(fields[1]); gotDir != wantDir {
		t.Errorf("child ran in %s, want the checkout %s", gotDir, wantDir)
	}
}

func TestExecSetup_AFailingScriptIsAnError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("drives a POSIX script")
	}
	script := filepath.Join(t.TempDir(), "setup.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_SELFUPDATE_SETUP_CMD", script)
	if err := ExecSetup(Env{RepoRoot: t.TempDir(), GOOS: runtime.GOOS}); err == nil {
		t.Fatal("a non-zero setup exit must be an error")
	}
}
