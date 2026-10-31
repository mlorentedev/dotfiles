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
	// unitMainPID is the unit's MainPID in the user manager.
	unitMainPID = func(unit string) (int, error) {
		out, err := exec.Command("systemctl", "--user", "show", "-p", "MainPID", "--value", unit).Output()
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(strings.TrimSpace(string(out)))
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
	unit, unitErr := selfUnit()
	dir := skippedSecretsDir()
	if err == nil {
		// A unit that resolves its secrets is no longer waiting for them, however
		// the vault came to be unlocked.
		if unit != "" && unitErr == nil && dir != "" {
			_ = secrets.ClearDegraded(dir, unit)
		}
		return injected, nil
	}
	if !skipLocked || !errors.Is(err, secrets.ErrBWVaultLocked) {
		return nil, err
	}
	_, _ = fmt.Fprintf(errOut, "warning: Bitwarden is locked, so the command runs without %s "+
		"(--skip-locked). Run `dotf secrets unlock`", selectionNames(sel))
	// Every way the trace can be lost is said out loud: without it the unlock
	// restarts nothing and doctor passes a unit that has no secrets.
	switch {
	case unit == "":
		_, _ = fmt.Fprintln(errOut, ".")
	case unitErr != nil:
		_, _ = fmt.Fprintf(errOut, ". This may be %s, but its main process could not be confirmed (%v), "+
			"so the unlock will not restart it: run `systemctl --user restart %s` after it.\n", unit, unitErr, unit)
	default:
		mErr := errors.New("no XDG_RUNTIME_DIR")
		if dir != "" {
			mErr = secrets.MarkDegraded(dir, unit, time.Now())
		}
		if mErr != nil {
			_, _ = fmt.Fprintf(errOut, ". It could not record that %s runs without them (%v), so the unlock "+
				"will not restart it: run `systemctl --user restart %s` after it.\n", unit, mErr, unit)
			break
		}
		_, _ = fmt.Fprintf(errOut, ", which restarts %s with them.\n", unit)
	}
	return nil, nil
}

// selfUnit is the systemd unit this process is the main process of, or "".
// Being in a unit's cgroup is not enough: a shell inside a terminal that runs
// as a service is in that service's cgroup too, and `unlock` would restart the
// terminal. Only the unit's own MainPID is the run the unit started.
//
// An error names the unit it could not confirm, so the caller can say which
// unit the unlock will not restart.
func selfUnit() (string, error) {
	cg, err := selfCgroup()
	if err != nil {
		return "", nil
	}
	unit := secrets.UnitFromCgroup(cg)
	if unit == "" {
		return "", nil
	}
	pid, err := unitMainPID(unit)
	if err != nil {
		return unit, err
	}
	if pid != os.Getpid() {
		return "", nil
	}
	return unit, nil
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
