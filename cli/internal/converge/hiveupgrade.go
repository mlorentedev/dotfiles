package converge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// hiveUpgradeRetire removes the 15-minute hive upgrade timer the setup scripts
// installed until #2857: hive-upgrade.{service,timer} on linux, the
// DotfilesHiveUpgrade task and its script copy on windows. hive ADR-020 says
// hive notifies and the user upgrades, so nothing may upgrade it unattended.
//
// Unlike self-deploy it removes a trigger it did not write: that trigger has
// no owner left to adopt it, and leaving it in place is the defect.
type hiveUpgradeRetire struct {
	scheduler func(Env) (schedule.Backend, error) // NativeScheduler in production
}

// hiveUpgradeJob carries only the names the setup scripts used; Remove and
// Inspect address a trigger by name.
var hiveUpgradeJob = schedule.Job{Names: schedule.Names{Systemd: "hive-upgrade", Task: "DotfilesHiveUpgrade"}}

func (hiveUpgradeRetire) Name() string        { return "hive-upgrade-retire" }
func (hiveUpgradeRetire) Platforms() []string { return []string{"linux", "windows"} }

// hiveUpgradeScript is where setup-windows.ps1 copied windows/hive-upgrade.ps1.
func hiveUpgradeScript(home string) string {
	return filepath.Join(home, ".claude", "scripts", "hive-upgrade.ps1")
}

// hiveUpgradeLeft is what is still installed, or why the step does not apply.
type hiveUpgradeLeft struct {
	backend schedule.Backend
	trigger string // the trigger's Where, or empty when absent
	script  string // the script copy's path, or empty when absent
	skip    string
}

func (r hiveUpgradeRetire) inspect(e Env) (hiveUpgradeLeft, error) {
	if r.scheduler == nil {
		return hiveUpgradeLeft{skip: "no scheduler runner is wired into this registry"}, nil
	}
	b, err := r.scheduler(e)
	if errors.Is(err, schedule.ErrUnsupported) {
		return hiveUpgradeLeft{skip: err.Error()}, nil
	}
	if err != nil {
		return hiveUpgradeLeft{}, err
	}
	st, err := b.Inspect(hiveUpgradeJob)
	if err != nil {
		return hiveUpgradeLeft{}, err
	}
	left := hiveUpgradeLeft{backend: b}
	if st.Present {
		left.trigger = st.Where
	}
	if e.GOOS == "windows" {
		p := hiveUpgradeScript(e.Home)
		switch _, err := os.Stat(p); {
		case err == nil:
			left.script = p
		case !errors.Is(err, fs.ErrNotExist):
			return hiveUpgradeLeft{}, err
		}
	}
	return left, nil
}

func (l hiveUpgradeLeft) items() []string {
	var out []string
	for _, s := range []string{l.trigger, l.script} {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (r hiveUpgradeRetire) Reconcile(e Env, dryRun bool) (Result, error) {
	left, err := r.inspect(e)
	if left.skip != "" || err != nil {
		return Result{Skip: left.skip}, err
	}
	items := left.items()
	if len(items) == 0 {
		return Result{Detail: "no unattended hive upgrade is installed"}, nil
	}
	verb := "removed: "
	if dryRun {
		verb = "to remove: "
	}
	res := Result{Changes: len(items), Detail: verb + strings.Join(items, "; ") + " (hive ADR-020: nothing upgrades hive unattended)"}
	if dryRun {
		return res, nil
	}
	if left.trigger != "" {
		if err := left.backend.Remove(hiveUpgradeJob); err != nil {
			if e.GOOS == "windows" {
				// An elevated setup registered the task as S4U, which a
				// standard user cannot delete.
				err = fmt.Errorf("%w (a task an elevated setup registered needs an elevated converge)", err)
			}
			return res, err
		}
	}
	if left.script != "" {
		if err := os.Remove(left.script); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return res, err
		}
	}
	return res, nil
}

// Probe re-inspects: after an apply neither the trigger nor the script is left.
func (r hiveUpgradeRetire) Probe(e Env) error {
	left, err := r.inspect(e)
	if left.skip != "" || err != nil {
		return err
	}
	if items := left.items(); len(items) > 0 {
		return fmt.Errorf("the unattended hive upgrade is still installed: %s", strings.Join(items, "; "))
	}
	return nil
}
