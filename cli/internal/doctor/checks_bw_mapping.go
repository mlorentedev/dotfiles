package doctor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// checkBWMapping asserts that every Bitwarden item the registry names actually
// exists in the vault — the drift between the mapping SSOT and the store it maps
// into.
//
// It exists because that drift is not a cosmetic mismatch: `dotf secrets run`
// resolves the WHOLE registry when the caller passes no --only and fails fast on
// the first unresolvable entry, by design (a child must never launch with a
// partially-populated secret set). So one stale item name takes down every
// unscoped run — which is the `pi` shell wrapper and, worse, `dotf spec review`,
// whose launcher builds an unscoped run. On 2026-08-15 a single entry naming
// `dockerhub` while the vault held `DockerHub` made the adversarial-review gate
// unrunnable for every spec in every repo, and the only symptom anyone saw was a
// review that produced an empty transcript.
//
// The check is name-only: it lists item names through the daemon and compares
// sets. It never reads a field, never resolves a secret, and never sees a value —
// so it stays cheap enough for the full sweep and safe to run anywhere.
//
// Severity mirrors checkBitwardenReach's rule: an unreachable or locked vault is
// not a finding here (that section owns it), but a reachable vault missing an
// item a live entry depends on is a FAIL, because something is already broken.
func checkBWMapping(sys *System, cfg *Config, rep *Report) {
	rep.Section("Bitwarden mapping (registry -> vault)")

	reg, err := loadRegistry(cfg)
	if err != nil {
		rep.Skip("secrets/registry.yaml not readable — checkSecrets owns that failure")
		return
	}

	declared := map[string][]string{} // item name -> secret ids naming it
	for i := range reg.Secrets {
		s := &reg.Secrets[i]
		if s.Backend != secrets.BackendBW || s.BW == nil || s.BW.Item == "" {
			continue
		}
		declared[s.BW.Item] = append(declared[s.BW.Item], s.ID)
	}
	if len(declared) == 0 {
		rep.Skip("no bw-backed secrets in the registry")
		return
	}

	present, err := sys.BWItemNames()
	if err != nil {
		// Locked, absent daemon, transport error: not this section's finding.
		reason := err.Error()
		if strings.Contains(reason, "connection refused") || strings.Contains(reason, "unreachable") {
			reason = "bw serve daemon not running"
		}
		rep.Skip(fmt.Sprintf("vault item list unavailable (%s) — mapping unverifiable", reason))
		return
	}
	have := make(map[string]bool, len(present))
	for _, n := range present {
		have[n] = true
	}

	missing := make([]string, 0, len(declared))
	for item := range declared {
		if !have[item] {
			missing = append(missing, item)
		}
	}
	sort.Strings(missing)

	for _, item := range missing {
		ids := declared[item]
		sort.Strings(ids)
		rep.Fail(fmt.Sprintf(
			"%s: no such item in the vault, named by %s — every `dotf secrets run` without --only fails on it, including `dotf spec review`",
			item, strings.Join(ids, ", ")))
	}
	if len(missing) == 0 {
		rep.Pass(fmt.Sprintf("all %d bw item(s) named by the registry exist in the vault", len(declared)))
	}
}
