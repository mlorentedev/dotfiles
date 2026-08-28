package doctor

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
		reportBWServeCacheAge(sys, rep)
	case "unlocked":
		rep.Pass("daemon running and unlocked")
		reportBWServeCacheAge(sys, rep)
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

// bwServeStaleCache is the age past which the daemon's own vault cache is
// reported. Tighter than bwStaleSync (the CLI cache's token-expiry horizon)
// because this cache decides what every daemon-served read returns, and since
// CLI-056 `dotf secrets unlock` refreshes it for free.
const bwServeStaleCache = 7 * 24 * time.Hour

// reportBWServeCacheAge says how old the cache the daemon answers from is. It
// is a different number from `bw status`'s lastSync (checkBWSyncAge): the CLI
// and the daemon cache independently. On the Windows work box a daemon cache
// twelve days behind resolved a rotated token to its old value and a newer
// item to "not found", and doctor's PAT tier called the token dead (CLI-056,
// #1316) — this row is the one-line diagnosis that was missing.
func reportBWServeCacheAge(sys *System, rep *Report) {
	ts, err := sys.BWServeLastSync()
	if err != nil {
		rep.Warn("daemon vault cache age unreadable (" + err.Error() + ")")
		return
	}
	if ts.IsZero() {
		rep.Warn("daemon has never synced its vault cache — every daemon-served read resolves against nothing; run `dotf secrets unlock` (it syncs)")
		return
	}
	age := sys.Now().Sub(ts)
	days := int(age.Hours() / 24)
	switch {
	case age < 0:
		rep.Warn("daemon lastSync is in the future — check the system clock; cache age cannot be judged")
	case age > bwServeStaleCache:
		rep.Warn(fmt.Sprintf("daemon vault cache is %dd old (>%dd) — items created or rotated since resolve as missing or stale; run `dotf secrets unlock` (it syncs) (CLI-056)",
			days, int(bwServeStaleCache.Hours()/24)))
	default:
		rep.Pass(fmt.Sprintf("daemon vault cache synced %dd ago", days))
	}
}
