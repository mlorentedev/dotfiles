package doctor

import (
	"strings"
	"time"
)

// dockerEngineTimeout bounds `docker info`: a stale socket can leave it waiting
// on a daemon that never answers, and doctor is the last step of setup.
const dockerEngineTimeout = 5 * time.Second

// engineProbeAttempts × engineProbeInterval bound the wait after --fix starts
// Colima: its VM takes tens of seconds to boot. Tests zero the interval.
var (
	engineProbeAttempts = 30
	engineProbeInterval = 3 * time.Second
)

// checkDockerEngine reports whether the docker CLI has an engine behind it. On
// the Mac the engine is Colima's VM, which Homebrew runs as a launchd service
// once `brew services start colima` has run (#2013 P5b); until then the CLI is
// installed and every `docker compose` fails. With --fix doctor runs that
// command itself, so no step is left to the owner. A WARN, never a FAIL: an
// engine is stopped on purpose as often as by accident, and the Windows gate
// fails on any [FAIL] line its known-failures list does not name.
func checkDockerEngine(sys *System, rep *Report, fix bool) {
	rep.Section("Docker engine")

	if !sys.has("docker") {
		rep.Skip("docker not on PATH (see Core tools)")
		return
	}

	v, errOut := dockerInfo(sys)
	if v != "" {
		rep.Pass("engine reachable: server " + v)
		return
	}
	if sys.GOOS == "linux" && strings.Contains(strings.ToLower(errOut), "permission denied") {
		// The daemon runs and the socket refuses this user: starting the unit
		// again changes nothing. apt's docker.io creates the docker group but
		// adds nobody to it.
		rep.Warn("the docker engine is running but this user cannot open its socket " +
			"(run: sudo usermod -aG docker $USER, then log in again)")
		return
	}
	if fix && sys.GOOS == "darwin" && sys.has("brew") && sys.has("colima") {
		startColima(sys, rep)
		return
	}
	rep.Warn("docker has no engine to talk to (" + dockerEngineRemedy(sys) + ")")
}

func dockerServerVersion(sys *System) string {
	v, _ := dockerInfo(sys)
	return v
}

// dockerInfo returns the engine's server version, or "" and the CLI's stderr,
// which is what tells a stopped engine from a socket this user cannot open.
func dockerInfo(sys *System) (version, errOut string) {
	out, errOut, err := sys.CommandOutputBounded(dockerEngineTimeout, "docker", "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		return "", errOut
	}
	return strings.TrimSpace(out), ""
}

// startColima registers Colima as a launchd service, which also starts it now
// and at every login, then waits for the engine to answer.
func startColima(sys *System, rep *Report) {
	if _, errOut, err := sys.CommandOutputBounded(time.Minute, "brew", "services", "start", "colima"); err != nil {
		rep.Warn("brew services start colima failed: " + strings.TrimSpace(errOut+" "+err.Error()))
		return
	}
	rep.Fix("started Colima as a launchd service (brew services start colima)")
	waitForEngine(sys, rep)
}

// waitForEngine polls the engine Colima is booting until it answers.
func waitForEngine(sys *System, rep *Report) {
	for i := 0; i < engineProbeAttempts; i++ {
		if v := dockerServerVersion(sys); v != "" {
			rep.Pass("engine reachable: server " + v)
			return
		}
		time.Sleep(engineProbeInterval)
	}
	rep.Warn("Colima is starting but the engine has not answered yet; re-run dotf doctor in a minute")
}

func dockerEngineRemedy(sys *System) string {
	switch sys.GOOS {
	case "darwin":
		// --fix starts Colima through Homebrew, so naming --fix while either
		// is missing would send the owner back to the command that just
		// did nothing.
		if !sys.has("brew") {
			return "Homebrew is not installed; install it from https://brew.sh, then run: dotf tools install, then dotf doctor --fix"
		}
		if !sys.has("colima") {
			return "colima is not installed (run: dotf tools install, then dotf doctor --fix)"
		}
		return "run: dotf doctor --fix, which starts Colima as a launchd service"
	case "windows":
		return "start Docker Desktop"
	}
	return "start it: sudo systemctl start docker"
}

// checkDockerCompose covers the one name in setup-linux.sh's check_dependencies
// list that no doctor section, contract binary or package list mentioned:
// docker-compose. OPS-043 deletes that shell call, so without this the tool
// would lose its only mention.
//
// It probes the v2 CLI plugin first because that is the supported form —
// compose v2 is `docker compose`, a plugin with no entry on PATH, so
// `command -v docker-compose` (what the shell call did) reports "missing" on a
// current install where compose works fine. Measured on msi 2026-09-02: the
// standalone v1 binary and plugin v2.39.1 are both present, and either check
// alone would have described that box wrongly.
//
// Absence is a WARN on darwin and Linux, where docker is provisioned and every
// `docker compose` fails without the plugin, and never a FAIL: the Windows gate
// fails on any unlisted [FAIL] line, and an engine-less box is not broken. On
// Windows compose ships inside Docker Desktop, so its absence is a SKIP.
func checkDockerCompose(sys *System, rep *Report) {
	rep.Section("Docker Compose")

	if !sys.has("docker") {
		rep.Skip("docker not on PATH — nothing for the compose plugin to attach to (see Core tools)")
		return
	}

	if out, err := sys.CommandOutput("docker", "compose", "version"); err == nil {
		if v := strings.TrimSpace(out); v != "" {
			rep.Pass("compose v2 plugin: " + v)
			return
		}
	}

	if sys.has("docker-compose") {
		// Homebrew's docker-compose is the v2 binary, which `docker compose`
		// finds only once docker-config registers its directory as a CLI
		// plugin dir. Calling that "legacy v1" passed a Mac on which every
		// `docker compose` failed (#2013 P5b).
		out, _ := sys.CommandOutput("docker-compose", "version")
		if v := strings.TrimSpace(out); strings.Contains(v, "v2.") {
			rep.Warn("compose v2 is installed but `docker compose` cannot find it (" + v +
				"); run: dotf deploy docker-config")
			return
		}
		rep.Pass("docker-compose found (legacy standalone v1; `docker compose` is the supported form)")
		return
	}

	switch sys.GOOS {
	case "darwin":
		rep.Warn("docker is installed without compose (run: dotf tools install)")
	case "linux":
		// No compose row on Linux yet: Ubuntu's docker-compose-v2 depends on
		// docker.io and would displace Docker's docker-ce, so the plugin has
		// to come from whichever source the engine came from.
		rep.Warn("docker is installed without compose; install the plugin from docker's own source: " +
			"sudo apt-get install docker-compose-v2 with Ubuntu's docker.io, docker-compose-plugin with Docker's docker-ce")
	default:
		rep.Skip("compose not found; on Windows it ships with Docker Desktop")
	}
}
