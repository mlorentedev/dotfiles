package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// fakeBackend is a scheduler holding one trigger: installing it makes it
// current and managed, removing it leaves nothing.
type fakeBackend struct {
	state              schedule.State
	inspectErr         error
	installs, removals int
}

func (f *fakeBackend) Inspect(schedule.Job) (schedule.State, error) { return f.state, f.inspectErr }
func (f *fakeBackend) Install(schedule.Job) error {
	f.installs++
	f.state = schedule.State{Present: true, Managed: true, Where: f.state.Where}
	return nil
}
func (f *fakeBackend) Remove(schedule.Job) error {
	f.removals++
	f.state = schedule.State{Where: f.state.Where}
	return nil
}

// selfDeployEnv is a scratch machine: XDG_CONFIG_HOME wins over the home in
// MachinePath, so without the override every test shares the runner's
// machine.json, and the opted-in tests declare the role for the others.
func selfDeployEnv(t *testing.T, optIn bool) Env {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if optIn {
		if _, err := env.SetMachineRole(env.MachinePath(home), env.RoleSelfDeploy, true); err != nil {
			t.Fatal(err)
		}
	}
	return Env{Home: home, GOOS: "linux"}
}

func (f *fakeBackend) step() selfDeploy {
	return selfDeploy{scheduler: func(Env) (schedule.Backend, error) { return f, nil }}
}

var (
	absent    = schedule.State{Where: "dotfiles-selfupdate.timer"}
	unmanaged = schedule.State{Present: true, Pending: []string{"write units"}, Where: "dotfiles-selfupdate.timer"}
	managed   = schedule.State{Present: true, Managed: true, Where: "dotfiles-selfupdate.timer"}
	stale     = schedule.State{Present: true, Managed: true, Pending: []string{"write units", "enable --now"}, Where: "dotfiles-selfupdate.timer"}
)

func TestSelfDeploy_OptedInInstallsThenIsConverged(t *testing.T) {
	f := &fakeBackend{state: absent}
	f.state.Pending = []string{"write units", "enable --now"}
	r, e := f.step(), selfDeployEnv(t, true)

	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes != 2 || f.installs != 0 || !strings.HasPrefix(plan.Detail, "to apply: ") {
		t.Fatalf("plan: %+v, %v, installs %d", plan, err, f.installs)
	}
	if err := r.Probe(e); err == nil {
		t.Error("the probe must fail while the trigger is pending")
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 2 || f.installs != 1 {
		t.Fatalf("apply: %+v, %v, installs %d", res, err, f.installs)
	}
	again, err := r.Reconcile(e, false)
	if err != nil || again.Changes != 0 || f.installs != 1 {
		t.Errorf("second apply must be changed=0: %+v, %v, installs %d", again, err, f.installs)
	}
	if err := r.Probe(e); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
}

// A setup script's trigger is adopted once the machine declares the role.
func TestSelfDeploy_OptedInAdoptsAnUnmanagedTrigger(t *testing.T) {
	f := &fakeBackend{state: unmanaged}
	if res, err := f.step().Reconcile(selfDeployEnv(t, true), false); err != nil || res.Changes != 1 || f.installs != 1 {
		t.Errorf("adopt: %+v, %v, installs %d", res, err, f.installs)
	}
}

func TestSelfDeploy_WithoutTheRole(t *testing.T) {
	tests := []struct {
		name        string
		state       schedule.State
		wantSkip    string
		wantChanges int
		wantRemoved int
	}{
		{"nothing installed is a skip naming the opt-in", absent, "dotf env role add self-deploy", 0, 0},
		{"a setup script's trigger is left in place", unmanaged, "installed by a setup script", 0, 0},
		{"dotf's trigger is removed", managed, "", 1, 1},
		{"dotf's stale trigger is removed too", stale, "", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeBackend{state: tt.state}
			r, e := f.step(), selfDeployEnv(t, false)
			plan, err := r.Reconcile(e, true)
			if err != nil || f.removals != 0 {
				t.Fatalf("plan: %v, removals %d", err, f.removals)
			}
			res, err := r.Reconcile(e, false)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(res.Skip, tt.wantSkip) || (tt.wantSkip == "") != (res.Skip == "") {
				t.Errorf("skip = %q, want it to contain %q", res.Skip, tt.wantSkip)
			}
			if plan.Changes != tt.wantChanges || res.Changes != tt.wantChanges || f.removals != tt.wantRemoved {
				t.Errorf("changes plan %d apply %d removals %d, want %d/%d", plan.Changes, res.Changes, f.removals, tt.wantChanges, tt.wantRemoved)
			}
			if tt.wantSkip == "" {
				if err := r.Probe(e); err != nil {
					t.Errorf("probe after removal: %v", err)
				}
			}
		})
	}
}

func TestSelfDeploy_SkipsWithoutAScheduler(t *testing.T) {
	e := selfDeployEnv(t, true)
	if res, err := (selfDeploy{}).Reconcile(e, false); err != nil || res.Skip == "" {
		t.Errorf("no scheduler wired: %+v, %v", res, err)
	}
	none := selfDeploy{scheduler: func(Env) (schedule.Backend, error) { return nil, schedule.ErrUnsupported }}
	if res, err := none.Reconcile(e, false); err != nil || res.Skip == "" {
		t.Errorf("unsupported OS: %+v, %v", res, err)
	}
	broken := selfDeploy{scheduler: func(Env) (schedule.Backend, error) { return nil, errors.New("boom") }}
	if _, err := broken.Reconcile(e, false); err == nil {
		t.Error("any other scheduler error fails the step")
	}
}

// The production scheduler addresses the run's home on every OS.
func TestNativeScheduler_PicksTheBackendOfTheRunsOS(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	s := NativeScheduler(501, func() bool { return false })
	for goos, want := range map[string]any{"darwin": schedule.Launchd{}, "linux": schedule.Systemd{}, "windows": schedule.Schtasks{}} {
		b, err := s(Env{Home: "/home/me", GOOS: goos})
		if err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		switch b := b.(type) {
		case schedule.Launchd:
			if _, ok := want.(schedule.Launchd); !ok || b.UID != 501 || !strings.HasSuffix(b.Dir, "LaunchAgents") {
				t.Errorf("%s: %+v", goos, b)
			}
		case schedule.Systemd:
			if _, ok := want.(schedule.Systemd); !ok || b.Dir != filepath.Join("/home/me", ".config", "systemd", "user") {
				t.Errorf("%s: %+v", goos, b)
			}
		case schedule.Schtasks:
			if _, ok := want.(schedule.Schtasks); !ok {
				t.Errorf("%s: %+v", goos, b)
			}
		}
	}
}

// A machine that never opted in is not failed by a scheduler dotf cannot
// reach (OPS-053 review round 1, F1): a container or a server with no user
// bus. The role is read first, and without it an unreachable scheduler is a
// skip that names the error; nothing is installed or removed. With the role
// the same error stays a failure, since the machine asked for the trigger.
func TestSelfDeploy_WithoutTheRoleNeverFailsOnAnUnreachableScheduler(t *testing.T) {
	bus := errors.New("systemctl --user show: Failed to connect to bus")
	for _, tc := range []struct {
		name    string
		step    func(*fakeBackend) selfDeploy
		backend bool // a backend exists that could be written to
	}{
		{"inspect fails", func(f *fakeBackend) selfDeploy { f.inspectErr = bus; return f.step() }, true},
		{"no backend", func(*fakeBackend) selfDeploy {
			return selfDeploy{scheduler: func(Env) (schedule.Backend, error) { return nil, bus }}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeBackend{}
			r := tc.step(f)
			e := selfDeployEnv(t, false)
			for _, dry := range []bool{true, false} {
				res, err := r.Reconcile(e, dry)
				if err != nil || !strings.Contains(res.Skip, "Failed to connect to bus") || res.Changes != 0 {
					t.Errorf("dryRun=%v: res = %+v, err = %v; want a skip naming the error", dry, res, err)
				}
			}
			if err := r.Probe(e); err != nil {
				t.Errorf("probe: %v", err)
			}
			if tc.backend && f.installs+f.removals != 0 {
				t.Errorf("installs %d, removals %d: the backend was written", f.installs, f.removals)
			}
		})
	}
	f := &fakeBackend{inspectErr: bus}
	if _, err := f.step().Reconcile(selfDeployEnv(t, true), true); err == nil {
		t.Error("with the role, an unreachable scheduler must stay a failure")
	}
}

// The OK detail names the slot the job carries rather than a literal 03:00, so
// moving the trigger moves the report (OPS-053 review round 1, F7a; #2727).
func TestSelfDeploy_TheDetailNamesTheJobsSlot(t *testing.T) {
	for _, tc := range []struct {
		daily *schedule.Clock
		want  string
	}{
		{&schedule.Clock{Hour: 4, Minute: 15}, "daily at 04:15"},
		{&schedule.Clock{Hour: 23, Minute: 5}, "daily at 23:05"},
		{nil, "daily"},
	} {
		if got := selfDeployWhen(schedule.Job{Daily: tc.daily}); got != tc.want {
			t.Errorf("selfDeployWhen(%+v) = %q, want %q", tc.daily, got, tc.want)
		}
	}

	f := &fakeBackend{state: managed}
	e := selfDeployEnv(t, true)
	res, err := f.step().Reconcile(e, false)
	want := "dotf update " + selfDeployWhen(update.Trigger(e.GOOS, e.Home, os.Getenv)) + " from " + managed.Where
	if err != nil || res.Changes != 0 || res.Detail != want {
		t.Errorf("converged: %+v, %v; want detail %q", res, err, want)
	}
}
