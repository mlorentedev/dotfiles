package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// --skip-locked exists for a daemon whose secrets are optional to most of what
// it serves (dotfiles#3005). hive.service started as `dotf secrets run --only
// NAN_API_KEY -- hive serve`, so after every reboot the vault tools, which need
// no secret, stayed down until the operator unlocked Bitwarden for a key only
// the worker reads: measured 188 failed starts in 96 minutes on msi.
//
// With the flag, a LOCKED vault, and only a locked vault, runs the child with
// none of the selected secrets. Every other failure (wrong age key, typo, empty
// value) still refuses, and so does a partial set: the child gets all of the
// selection or none of it. When the run is a systemd unit, it leaves a trace,
// and `dotf secrets unlock` restarts that unit so it starts again with them.

// Seams, so tests read no real cgroup, write no real runtime dir and restart no
// real unit.
var (
	selfCgroup = func() (string, error) {
		b, err := os.ReadFile("/proc/self/cgroup")
		return string(b), err
	}
	// unitMainPID is the unit's MainPID in the user manager, or 0 when unreadable.
	unitMainPID = func(unit string) int {
		out, err := exec.Command("systemctl", "--user", "show", "-p", "MainPID", "--value", unit).Output()
		if err != nil {
			return 0
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return pid
	}
	skippedSecretsDir = func() string { return secrets.DegradedDir(os.Getenv) }
	systemctlUser     = func(args ...string) error {
		return exec.Command("systemctl", append([]string{"--user"}, args...)...).Run()
	}
	hasSystemctl = func() bool {
		_, err := exec.LookPath("systemctl")
		return runtime.GOOS == "linux" && err == nil
	}
)

// resolveOrSkipLocked resolves the selection, or, with skipLocked and a locked
// vault only, returns none of it and records the unit that ran without it.
func resolveOrSkipLocked(reg *secrets.Registry, sel map[string]bool, skipLocked bool, errOut io.Writer) ([]string, error) {
	injected, err := resolveInjectedSecrets(reg, sel)
	unit, dir := selfUnit(), skippedSecretsDir()
	if err == nil {
		// A unit that resolves its secrets is no longer waiting for them, however
		// the vault came to be unlocked.
		if unit != "" && dir != "" {
			_ = secrets.ClearDegraded(dir, unit)
		}
		return injected, nil
	}
	if !skipLocked || !errors.Is(err, secrets.ErrBWVaultLocked) {
		return nil, err
	}
	_, _ = fmt.Fprintf(errOut, "warning: Bitwarden is locked, so the command runs without %s "+
		"(--skip-locked). Run `dotf secrets unlock`", selectionNames(sel))
	if unit != "" && dir != "" {
		if mErr := secrets.MarkDegraded(dir, unit, time.Now()); mErr == nil {
			_, _ = fmt.Fprintf(errOut, ", which restarts %s with them", unit)
		}
	}
	_, _ = fmt.Fprintln(errOut, ".")
	return nil, nil
}

// selfUnit is the systemd unit this process is the main process of, or "".
// Being in a unit's cgroup is not enough: a shell inside a terminal that runs
// as a service is in that service's cgroup too, and `unlock` would restart the
// terminal. Only the unit's own MainPID is the run the unit started.
func selfUnit() string {
	cg, err := selfCgroup()
	if err != nil {
		return ""
	}
	unit := secrets.UnitFromCgroup(cg)
	if unit == "" || unitMainPID(unit) != os.Getpid() {
		return ""
	}
	return unit
}

func selectionNames(sel map[string]bool) string {
	if len(sel) == 0 {
		return "any mapped secret"
	}
	names := make([]string, 0, len(sel))
	for n := range sel {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// restartSkippedUnits restarts every unit that ran without its secrets, now
// that the vault is unlocked. try-restart leaves a unit that has since stopped
// stopped. A failure is reported, never returned: the unlock itself succeeded.
func restartSkippedUnits(out io.Writer) {
	dir := skippedSecretsDir()
	if dir == "" || !hasSystemctl() {
		return
	}
	units, err := secrets.DegradedUnits(dir)
	if err != nil {
		_, _ = fmt.Fprintf(out, "warning: could not read the units waiting for secrets: %v\n", err)
		return
	}
	names := make([]string, 0, len(units))
	for u := range units {
		names = append(names, u)
	}
	sort.Strings(names)
	for _, u := range names {
		if err := systemctlUser("try-restart", u); err != nil {
			_, _ = fmt.Fprintf(out, "warning: %s ran without its secrets since %s and did not restart: %v. "+
				"Run `systemctl --user restart %s`.\n", u, units[u], err, u)
			continue
		}
		_ = secrets.ClearDegraded(dir, u)
		_, _ = fmt.Fprintf(out, "restarted %s, which ran without its secrets since %s\n", u, units[u])
	}
}
