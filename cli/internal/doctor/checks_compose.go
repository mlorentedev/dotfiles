package doctor

import "strings"

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
