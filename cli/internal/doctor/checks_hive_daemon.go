package doctor

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// hiveStatusTimeout bounds `hive service status`: a Python start plus hive's
// own 2s pinned /health probe, with room for a cold interpreter cache.
const hiveStatusTimeout = 20 * time.Second

// hiveDaemonPrefix starts the verdict line `hive service status` prints last.
// On Linux the systemctl passthrough comes first, so the verdict is found by
// prefix on its own line, never by matching the whole stdout.
const hiveDaemonPrefix = "hive daemon: "

// checkHiveDaemonAnswers is BUG-115 (#2231): every agent registers hive as
// `hive client`, a stdio shim that fails outright when no `hive serve` daemon
// answers. Doctor used to inspect the daemon's supervisor, so on a host with
// no systemd (darwin) a dead daemon read as an INFO line while every session's
// hive MCP was CONNECTION_CLOSED.
//
// It probes the daemon, not the supervisor, through hive's own
// `hive service status`: that command derives the per-user port and proves the
// listener against the daemon's pinned certificate on every OS. Doing either
// here would duplicate logic hive owns and let the two drift.
func checkHiveDaemonAnswers(sys *System, cfg *Config, rep *Report) {
	if !hiveClientRegistered(sys, cfg, rep) {
		return
	}
	rep.Section("hive daemon (the backend of every agent's `hive client`)")

	if _, err := sys.LookPath("hive"); err != nil {
		rep.Fail("mcp-servers.json registers hive as `hive client`, but `hive` is not on PATH, " +
			"so the hive MCP server fails to start in every agent. Fix: `uv tool install --upgrade hive-vault`.")
		return
	}

	stdout, stderr, err := sys.CommandOutputBounded(hiveStatusTimeout, "hive", "service", "status")
	state, answered := hiveDaemonState(stdout)
	switch {
	case !answered:
		detail := lastLine(stderr)
		if err != nil && detail == "" {
			detail = err.Error()
		}
		rep.Fail("`hive service status` gave no verdict, so whether the daemon answers is unknown (" +
			detail + "). Every agent's `hive client` depends on it.")
	case state == "healthy" && err != nil:
		// The command exits 0 only when healthy, so a healthy line with an
		// error is a probe that died after printing: no verdict either.
		rep.Fail("`hive service status` printed healthy but failed (" + err.Error() +
			"), so whether the daemon answers is unknown. Every agent's `hive client` depends on it.")
	case state == "healthy":
		rep.Pass("the hive daemon answers on its pinned port, so `hive client` can connect")
	default:
		rep.Fail("the hive daemon is " + state + ", so the hive MCP server fails in every agent " +
			"(`hive client` never starts a server of its own). " + hiveDaemonRemedy(sys.GOOS))
	}
}

// hiveClientRegistered reports whether agents on this host launch `hive client`.
// The claim needs the declaration AND either its prerequisite or hive itself:
// Register skips a server whose prerequisite binary is absent, but a host that
// still has `hive` may hold a registration made before the prerequisite went
// away, and that one still launches the shim.
func hiveClientRegistered(sys *System, cfg *Config, rep *Report) bool {
	servers, err := claude.LoadServers(filepath.Join(cfg.DotfilesDir, claude.ServersRel))
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		rep.Section("hive daemon (the backend of every agent's `hive client`)")
		rep.Warn("could not read the MCP server list, so the hive daemon was not checked (" + err.Error() + ")")
		return false
	}
	for _, s := range servers {
		// Tokens, not a prefix: registration splits Args with strings.Fields.
		args := strings.Fields(s.Args)
		if s.Name != "hive" || len(args) < 2 || args[0] != "hive" || args[1] != "client" {
			continue
		}
		if s.PrerequisiteBinary == "" {
			return true
		}
		if _, err := sys.LookPath(s.PrerequisiteBinary); err == nil {
			return true
		}
		_, err := sys.LookPath("hive")
		return err == nil
	}
	return false
}

// hiveDaemonState returns the state on the `hive daemon: ` line, and false when
// stdout has no such line: an older hive, a crash, or a timeout. That case is
// reported as "no verdict", never as "down".
func hiveDaemonState(stdout string) (string, bool) {
	for _, line := range strings.Split(stdout, "\n") {
		if state, ok := strings.CutPrefix(strings.TrimSpace(line), hiveDaemonPrefix); ok {
			return state, true
		}
	}
	return "", false
}

// hiveDaemonRemedy names the fix for the supervisor this OS has. darwin has
// none yet (#2013 row S2), so it says so instead of naming a setup script the
// Mac must not run (ADR-045).
func hiveDaemonRemedy(goos string) string {
	switch goos {
	case "darwin":
		return "macOS has no hive supervisor yet: start `hive serve` by hand, " +
			"then confirm with `hive service status`."
	case "windows":
		return "Fix: `hive service install` (Scheduled Task), then confirm with `hive service status`."
	default:
		return "Fix: `hive service install`, or `systemctl --user restart hive.service`, " +
			"then confirm with `hive service status`."
	}
}
