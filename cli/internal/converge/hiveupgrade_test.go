package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// hiveUpgradeEnv is a scratch home; on windows it holds the script copy the
// old setup deployed, when script is set.
func hiveUpgradeEnv(t *testing.T, goos string, script bool) Env {
	t.Helper()
	e := Env{Home: t.TempDir(), GOOS: goos}
	if script {
		p := hiveUpgradeScript(e.Home)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# hive-upgrade\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

func (f *fakeBackend) retire() hiveUpgradeRetire {
	return hiveUpgradeRetire{scheduler: func(Env) (schedule.Backend, error) { return f, nil }}
}

// A machine set up before #2857 has the timer, installed by a setup script and
// so unmanaged: the step removes it anyway, because nothing may upgrade hive
// unattended (hive ADR-020), and a second run changes nothing.
func TestHiveUpgradeRetire_RemovesTheTimerThenIsConverged(t *testing.T) {
	f := &fakeBackend{state: schedule.State{Present: true, Where: "hive-upgrade.timer"}}
	r, e := f.retire(), hiveUpgradeEnv(t, "linux", false)

	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes != 1 || f.removals != 0 || !strings.Contains(plan.Detail, "hive-upgrade.timer") {
		t.Fatalf("plan: %+v, %v, removals %d", plan, err, f.removals)
	}
	if err := r.Probe(e); err == nil {
		t.Error("the probe must fail while the timer is installed")
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 || f.removals != 1 {
		t.Fatalf("apply: %+v, %v, removals %d", res, err, f.removals)
	}
	if err := r.Probe(e); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
	again, err := r.Reconcile(e, false)
	if err != nil || again.Changes != 0 || f.removals != 1 || again.Skip != "" {
		t.Errorf("second apply must be changed=0: %+v, %v, removals %d", again, err, f.removals)
	}
}

// On windows the task ran a copy of windows/hive-upgrade.ps1 under ~/.claude:
// a retire that left it would leave half of the upgrade machinery behind.
func TestHiveUpgradeRetire_WindowsRemovesTheTaskAndTheScriptCopy(t *testing.T) {
	f := &fakeBackend{state: schedule.State{Present: true, Where: `task \DotfilesHiveUpgrade`}}
	r, e := f.retire(), hiveUpgradeEnv(t, "windows", true)

	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes != 2 {
		t.Fatalf("plan: %+v, %v", plan, err)
	}
	if _, err := os.Stat(hiveUpgradeScript(e.Home)); err != nil {
		t.Fatalf("a plan must not delete the script: %v", err)
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 2 || f.removals != 1 {
		t.Fatalf("apply: %+v, %v, removals %d", res, err, f.removals)
	}
	if _, err := os.Stat(hiveUpgradeScript(e.Home)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the script copy must be gone: %v", err)
	}
	if err := r.Probe(e); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
}

// The script alone, with no task, is still a leftover to remove.
func TestHiveUpgradeRetire_ScriptWithoutTaskIsRemoved(t *testing.T) {
	f := &fakeBackend{state: schedule.State{Where: `task \DotfilesHiveUpgrade`}}
	r, e := f.retire(), hiveUpgradeEnv(t, "windows", true)
	res, err := r.Reconcile(e, false)
	if err != nil || res.Changes != 1 || f.removals != 0 {
		t.Fatalf("apply: %+v, %v, removals %d", res, err, f.removals)
	}
	if _, err := os.Stat(hiveUpgradeScript(e.Home)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the script copy must be gone: %v", err)
	}
}

func TestHiveUpgradeRetire_NothingInstalledIsConverged(t *testing.T) {
	f := &fakeBackend{state: schedule.State{Where: "hive-upgrade.timer"}}
	r, e := f.retire(), hiveUpgradeEnv(t, "linux", false)
	res, err := r.Reconcile(e, false)
	if err != nil || res.Changes != 0 || res.Skip != "" || f.removals != 0 {
		t.Fatalf("got %+v, %v, removals %d", res, err, f.removals)
	}
}

// A scheduler dotf cannot reach is an error, never "nothing to remove": the
// timer could still be there (the read-error rule of schedule.Schtasks).
func TestHiveUpgradeRetire_UnreadableSchedulerFails(t *testing.T) {
	f := &fakeBackend{inspectErr: errors.New("no user bus")}
	r, e := f.retire(), hiveUpgradeEnv(t, "linux", false)
	if _, err := r.Reconcile(e, true); err == nil {
		t.Error("an unreadable scheduler must fail the step")
	}
	if err := r.Probe(e); err == nil {
		t.Error("an unreadable scheduler must fail the probe")
	}
}

// A machine with no scheduler backend has no timer to retire.
func TestHiveUpgradeRetire_UnsupportedSchedulerIsASkip(t *testing.T) {
	r := hiveUpgradeRetire{scheduler: func(Env) (schedule.Backend, error) { return nil, schedule.ErrUnsupported }}
	res, err := r.Reconcile(hiveUpgradeEnv(t, "linux", false), false)
	if err != nil || res.Skip == "" {
		t.Fatalf("got %+v, %v", res, err)
	}
}

// The job names are the ones the setup scripts used: a different name would
// find nothing and report converged on a machine that still has the timer.
func TestHiveUpgradeRetire_NamesMatchWhatSetupInstalled(t *testing.T) {
	if hiveUpgradeJob.Names.Systemd != "hive-upgrade" || hiveUpgradeJob.Names.Task != "DotfilesHiveUpgrade" {
		t.Errorf("names: %+v", hiveUpgradeJob.Names)
	}
	if got := hiveUpgradeScript("/h"); got != filepath.Join("/h", ".claude", "scripts", "hive-upgrade.ps1") {
		t.Errorf("script path: %s", got)
	}
}
