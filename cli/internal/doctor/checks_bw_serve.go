package doctor

import "strconv"

// checkBWServeDaemon reports the dotf-managed bw serve daemon's lock state —
// absent / locked / unlocked — as its own section, distinct from
// checkBitwardenReach's CLI-shellout reach tiers (CLI-024-secrets-bw-serve,
// AC4). The daemon is optional: `dotf secrets` falls back to the CLI
// shellout when none is running, so no state this check observes is ever a
// FAIL — only checkBitwardenReach's tiers can fail doctor over Bitwarden.
func checkBWServeDaemon(sys *System, rep *Report) {
	rep.Section("bw serve daemon (optional local unlock cache)")
	st, err := sys.BWServeStatus()
	if err != nil {
		rep.Warn("bw serve daemon status unreadable (" + err.Error() + ")")
		return
	}
	switch st {
	case "absent":
		rep.Info("no daemon running — dotf secrets uses the CLI shellout (run `dotf secrets unlock` to start one)")
	case "locked":
		rep.Info("daemon running, locked — run `dotf secrets unlock` to use it")
	case "unlocked":
		rep.Pass("daemon running and unlocked")
	default:
		rep.Warn("unrecognised bw serve status " + strconv.Quote(st))
	}
}
