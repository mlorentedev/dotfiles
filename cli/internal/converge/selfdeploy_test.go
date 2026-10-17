package converge

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// fakeBackend is a scheduler holding one trigger: installing it makes it
// current and managed, removing it leaves nothing.
type fakeBackend struct {
	state              schedule.State
	installs, removals int
}

func (f *fakeBackend) Inspect(schedule.Job) (schedule.State, error) { return f.state, nil }
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
