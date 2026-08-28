package doctor

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// bwServeTailLines is how many trailing log lines the "daemon exited" line
// carries: enough for the last Node error and its cause, not a stack dump.
const bwServeTailLines = 3

// checkBWServeDaemon reports the dotf-managed bw serve daemon's lock state —
// absent / locked / unlocked — as its own section, distinct from
// checkBitwardenReach's CLI-shellout reach tiers (CLI-024-secrets-bw-serve,
// AC4). The daemon is optional: `dotf secrets` falls back to the CLI
// shellout when none is running, so no state this check observes is ever a
// FAIL — only checkBitwardenReach's tiers can fail doctor over Bitwarden.
//
// "Absent" is two different facts, and since #1315 the pid file under the
// deploy dir's state area tells them apart: never started here, or started
// and gone. The second is the one that used to leave no trace.
func checkBWServeDaemon(sys *System, cfg *Config, rep *Report) {
	rep.Section("bw serve daemon (optional local unlock cache)")
	st, err := sys.BWServeStatus()
	if err != nil {
		rep.Warn("bw serve daemon status unreadable (" + err.Error() + ")")
		return
	}
	switch st {
	case "absent":
		reportAbsentBWServe(sys, secrets.NewBWServeState(cfg.DotfilesDir), rep)
	case "locked":
		rep.Info("daemon running, locked — run `dotf secrets unlock` to use it")
	case "unlocked":
		rep.Pass("daemon running and unlocked")
	default:
		rep.Warn("unrecognised bw serve status " + strconv.Quote(st))
	}
}

// reportAbsentBWServe renders "nothing answers" at the precision the trace
// allows. No pid file: nothing was started from this deploy dir, the resting
// state of a fresh box. A pid file whose pid is gone: the daemon exited, and
// its last log lines are the evidence nobody was there to read. A pid file
// whose pid is alive: it is still starting, or the pid now belongs to another
// process — said as the ambiguity it is, never as "the daemon is up".
//
// WARN, not FAIL, for the reason the section's doc gives: the shellout
// fallback still works, and reportAgentLaunchability already says what an
// absent daemon costs the wrappers.
func reportAbsentBWServe(sys *System, state secrets.BWServeState, rep *Report) {
	pid, err := state.ReadPID()
	switch {
	case errors.Is(err, os.ErrNotExist):
		rep.Info("no daemon running — dotf secrets uses the CLI shellout (run `dotf secrets unlock` to start one)")
		return
	case err != nil:
		rep.Warn("no daemon running, and its pid file is unreadable (" + err.Error() + ") — run `dotf secrets unlock` to start one")
		return
	}
	if sys.ProcessAlive != nil && sys.ProcessAlive(pid) {
		rep.Warn(fmt.Sprintf("daemon pid %d (%s) is alive but nothing answers on its port — still starting, or another process reusing the pid; log: %s",
			pid, state.PIDPath(), state.LogPath()))
		return
	}
	lines, readErr := state.LastLogLines(bwServeTailLines)
	tail := "log is empty"
	switch {
	case readErr != nil:
		tail = "log unreadable (" + readErr.Error() + ")"
	case len(lines) > 0:
		tail = strings.Join(lines, " | ")
	}
	rep.Warn(fmt.Sprintf("daemon exited — pid %d recorded in %s is gone; last lines: %s — run `dotf secrets unlock` to restart it (full log: %s)",
		pid, state.PIDPath(), tail, state.LogPath()))
}
