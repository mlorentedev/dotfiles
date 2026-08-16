package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Deployed agent-config hygiene.
//
// Two defects, both of which shipped silently for months because every layer
// involved reported a health it had never established (BUG-081b, #987):
//
//   - setup logged "deployed with literal {env:VAR} placeholders (resolved at
//     runtime)". Nothing resolves that syntax. pi sends the placeholder itself
//     as the bearer token and the server answers 401 — indistinguishable from a
//     bad credential, which is why a session spent most of its length inspecting
//     Bitwarden, the registry and the injection path, all of them healthy.
//   - pi's own preflight agreed: isConfigValueConfigured("{env:NAN_API_KEY}")
//     returns true, because a string with no `$` has no env vars that could be
//     missing.
//
// The second defect is the successful path, and it is the worse one: when
// `dotf secrets render` DOES resolve the placeholder, it writes the credential
// into the deployed file, where it sits in plaintext. That is how a live API key
// reached a session transcript on 2026-08-15 — someone opened the config to
// debug the first defect. It also contradicts ADR-028, whose whole posture is
// that secrets are injected into one child process and never persisted.
//
// So this check asks two questions of a deployed agent config: can the tool
// that reads it actually resolve what it says, and is there a credential in it.

// piConfigRel is the deployed pi model registry, relative to $HOME. pi reads
// this path itself; it is not configurable from here.
const piConfigRel = ".pi/agent/models.json"

// piResolvable reports whether pi's own resolver can interpolate this value.
//
// pi (dist/core/resolve-config-value.js) interpolates `$VAR` and `${VAR}`, and
// executes a leading `!` as a shell command. Anything else is a literal — which
// is the whole bug: `{env:VAR}` contains no `$`, so it is sent verbatim.
func piResolvable(v string) bool {
	return strings.HasPrefix(v, "$") || strings.HasPrefix(v, "!")
}

func checkAgentConfigSecrets(sys *System, rep *Report) {
	rep.Section("Agent config secrets")

	path := filepath.Join(sys.home(), piConfigRel)
	raw, err := os.ReadFile(path) //nolint:gosec // a fixed path under the user's own home
	if err != nil {
		// No deployed pi config is not a defect: pi may simply not be installed
		// on this machine. Nothing to check beats a red section nobody can act on.
		rep.Skip("no deployed pi config at " + path + " — nothing to check")
		return
	}

	// The unresolvable-placeholder half. Checked on the raw bytes rather than
	// per-field: `{env:` anywhere in a pi config is broken by the same argument,
	// whether it sits in apiKey, a header, or a field added later.
	if strings.Contains(string(raw), "{env:") {
		rep.Fail(fmt.Sprintf(
			"%s still contains a {env:...} placeholder — that is dotf's syntax, not pi's, "+
				"so pi sends it verbatim as the credential and every call 401s; "+
				"re-run setup to redeploy the config", path))
	}

	var cfg struct {
		Providers map[string]struct {
			APIKey string `json:"apiKey"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		rep.Warn("deployed pi config is not parseable JSON (" + err.Error() + ") — its secrets could not be checked")
		return
	}

	// The materialised-secret half. NOTE what is reported: the provider name and
	// the fact that its key is a literal — never the value, not even truncated.
	// A check that prints a credential to prove a credential is on disk has
	// reproduced the defect it is reporting (CLI-038, #1012).
	var materialised []string
	for name, p := range cfg.Providers {
		if p.APIKey == "" || piResolvable(p.APIKey) {
			continue
		}
		materialised = append(materialised, name)
	}
	if len(materialised) > 0 {
		rep.Fail(fmt.Sprintf(
			"%s holds a literal credential for provider(s) %s — a deployed config must "+
				"reference the environment (\"${VAR}\"), never carry the secret; "+
				"re-run setup to redeploy from source (BUG-081b)",
			path, strings.Join(materialised, ", ")))
	}

	if !strings.Contains(string(raw), "{env:") && len(materialised) == 0 {
		rep.Pass("deployed pi config resolves its secrets from the environment")
	}
}
