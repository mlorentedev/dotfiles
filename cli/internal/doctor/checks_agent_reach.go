package doctor

import "strings"

// checkHiveBackendCanServe is CLI-042 AC9: "zero reachable models" must be
// caught at diagnostic time, not under dispatch load.
//
// This is the incident-to-guard emission for the defect that shaped the whole
// spec. hive was a declared backend of `dotf agent run` with no reachable
// provider for an unknown length of time and NOTHING NOTICED — the daemon was
// `active (running)`, its binary was on PATH, every probe said present, and it
// could not have answered a single request. Measured 2026-08-24: hive.service
// had been up for 2h40m with no NAN_API_KEY anywhere in its environment.
//
// The check is deliberately a PROXY rather than a live reachability probe.
// hive DOES expose the stronger surface — its `worker_status` MCP tool probes
// the provider rather than inferring it, built by mlorentedev/hive#384 for
// exactly this failure — but reaching it from here would mean an MCP client
// inside `dotf doctor`: a dependency, a handshake that can drift, and a surface
// bats cannot smoke. PR D rejected that same trade for the hive BACKEND, and it
// does not become a better trade for a diagnostic. So this asks the question
// that is free, and sound in the one direction that matters:
//
//	hive's worker serves exactly one pool (nan, since hive#384), and reaching it
//	needs BOTH halves of its contract — HIVE_WORKER_BASE_URL and
//	HIVE_WORKER_API_KEY. A supervised daemon missing either can serve NOTHING,
//	with certainty rather than by inference.
//
// It therefore cannot report every unreachable configuration and never claims
// to: a present-but-wrong key still reads as configured here, and only
// `worker_status` or a real dispatch would catch that. What it does guarantee is
// that the exact state nobody noticed for an unknown length of time goes red the
// moment it recurs, which is the bar the criterion sets.
func checkHiveBackendCanServe(sys *System, rep *Report) {
	rep.Section("hive backend reachability (dotf agent run)")

	// Not a Linux systemd host. Windows supervises the daemon through a
	// Scheduled Task (windows/hive-serve-supervisor.ps1), whose equivalent
	// assertion is that script's own concern — claiming a verdict here from a
	// host that cannot see the unit would be a guess wearing a check's clothes.
	if sys.GOOS == "windows" {
		rep.Info("skipped: the daemon is a Scheduled Task on Windows, not a systemd unit")
		return
	}
	if _, err := sys.LookPath("systemctl"); err != nil {
		rep.Info("skipped: no systemctl on this host, so no supervised hive daemon to inspect")
		return
	}

	// hive is not installed here, so nothing declares it as a backend and there
	// is no claim to falsify. Silent rather than Info: a machine that never had
	// hive should not carry a line about it in every doctor run.
	if _, err := sys.LookPath("hive"); err != nil {
		return
	}

	// Both directives in one call, WITHOUT --value, so each comes back on its own
	// `Key=value` line and the two cannot be confused for one another.
	out, err := sys.CommandOutput("systemctl", "--user", "show", "hive.service",
		"-p", "ExecStart", "-p", "Environment")
	if err != nil {
		rep.Warn("could not read hive.service, so backend reachability was not checked (" + err.Error() + ")")
		return
	}

	var execStart, environment string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "ExecStart="):
			execStart = strings.TrimPrefix(line, "ExecStart=")
		case strings.HasPrefix(line, "Environment="):
			environment = strings.TrimPrefix(line, "Environment=")
		}
	}

	// `systemctl show` on an unknown unit SUCCEEDS with an empty value — it does
	// not error. Telling that apart from a real reading matters: an empty string
	// must not be read as "no credential injection" and reported as the failure.
	// hive works without the daemon (the client falls back), so this is a state
	// to name, not to fail.
	if strings.TrimSpace(execStart) == "" {
		rep.Info("hive is installed but hive.service is not, so no supervised daemon is declared")
		return
	}

	// Both halves, because a daemon holding one serves exactly as little as one
	// holding neither — which is what `worker_status` reported on this machine
	// while systemd called the unit active.
	//
	// The credential arrives through `dotf secrets run` (AC7), and the token it
	// is scoped to is the registry ID: that selects every var the secret
	// exposes, which is how hive gets HIVE_WORKER_API_KEY from an item whose
	// primary name is NAN_API_KEY. Matching the mechanism alone would pass a
	// drop-in that injected some unrelated secret.
	missing := []string{}
	if !strings.Contains(execStart, "secrets run") || !strings.Contains(execStart, "NAN_API_KEY") {
		missing = append(missing, "the API key (no `dotf secrets run --only NAN_API_KEY` in ExecStart)")
	}
	// Never echo the value: an Environment= line is operator-controlled and this
	// message is written to a transcript.
	if !strings.Contains(environment, "HIVE_WORKER_BASE_URL=") {
		missing = append(missing, "the worker base URL (no HIVE_WORKER_BASE_URL in the unit)")
	}
	if len(missing) > 0 {
		rep.Fail("hive.service is running without " + strings.Join(missing, " and ") +
			", so the hive backend probes PRESENT and can serve nothing. " +
			"`dotf agent run` will route to it and every dispatch will fail. " +
			"Fix: re-run ./setup-linux.sh to deploy " +
			"systemd/hive.service.d/10-dotf-secrets.conf, then `systemctl --user " +
			"daemon-reload && systemctl --user restart hive.service`. Confirm with " +
			"hive's own worker_status, which must report Configured: yes.")
		return
	}

	rep.Pass("hive.service carries both halves of the worker contract — the backend can reach its pool")
}
