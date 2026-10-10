package doctor

import (
	"strings"
	"time"
)

// dockerEngineTimeout bounds `docker info`: a stale socket can leave it waiting
// on a daemon that never answers, and doctor is the last step of setup.
const dockerEngineTimeout = 5 * time.Second

// checkDockerEngine reports whether the docker CLI has an engine behind it. On
// the Mac the engine is Colima's VM, which Homebrew runs as a launchd service
// once `brew services start colima` has been run (#2013 P5b); until then the
// CLI is installed and every `docker compose` fails. A WARN, never a FAIL: an
// engine is stopped on purpose as often as by accident, and the Windows gate
// fails on any [FAIL] line its known-failures list does not name.
func checkDockerEngine(sys *System, rep *Report) {
	rep.Section("Docker engine")

	if !sys.has("docker") {
		rep.Skip("docker not on PATH (see Core tools)")
		return
	}

	out, _, err := sys.CommandOutputBounded(dockerEngineTimeout, "docker", "info", "--format", "{{.ServerVersion}}")
	if v := strings.TrimSpace(out); err == nil && v != "" {
		rep.Pass("engine reachable: server " + v)
		return
	}
	rep.Warn("docker has no engine to talk to (" + dockerEngineRemedy(sys.GOOS) + ")")
}

func dockerEngineRemedy(goos string) string {
	switch goos {
	case "darwin":
		return "run once: brew services start colima; launchd restarts it at login"
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
// Absence is a SKIP, never a FAIL: the repo provisions compose in no installer
// block, no versions.conf pin and no contract binary, so failing on it would red
// a box that never asked for it — the same reasoning BUG-052 applied to
// terraform.
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
		rep.Pass("docker-compose found (legacy standalone v1; `docker compose` is the supported form)")
		return
	}

	rep.Skip("compose not installed (optional — the repo provisions neither the plugin nor the binary)")
}
