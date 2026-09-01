package doctor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// bwMappingStaleSync is the threshold beyond which the local vault cache is
// considered potentially stale when reporting missing items (BUG-087).
//
// If the vault was synced within this window, a missing item is considered a
// genuine absence (FAIL). If the vault has not been synced or is older than
// this window, doctor emits a WARN indicating the item was not found in the
// local cache and advising a `dotf secrets unlock`, which syncs the daemon's
// cache (CLI-056). The remedy used to name `dotf secrets sync`, which
// materializes CI secrets and refreshes no cache at all.
const bwMappingStaleSync = 24 * time.Hour

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
// not a finding here (that section owns it). When an item appears missing:
//   - If the daemon's last sync is fresh (within bwMappingStaleSync), it is a FAIL
//     because the item is genuinely absent from the vault.
//   - If the daemon's last sync is stale or unknown (never synced), it is a WARN
//     explaining that the item was not found in the local cache and advising a sync (BUG-087).
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

	var lastSync time.Time
	if sys.BWLastSync != nil {
		if t, err := sys.BWLastSync(); err == nil {
			lastSync = t
		}
	}

	for _, item := range missing {
		ids := declared[item]
		sort.Strings(ids)

		if !lastSync.IsZero() && sys.Now().Sub(lastSync) >= 0 && sys.Now().Sub(lastSync) <= bwMappingStaleSync {
			rep.Fail(fmt.Sprintf(
				"%s: no such item in the vault, named by %s — every `dotf secrets run` without --only fails on it, including `dotf spec review`",
				item, strings.Join(ids, ", ")))
		} else if lastSync.IsZero() {
			rep.Warn(fmt.Sprintf(
				"%s: not found in local vault cache (never synced), named by %s — run `dotf secrets unlock` (syncs the daemon's cache) to refresh; if missing from vault, every unscoped `dotf secrets run` fails on it",
				item, strings.Join(ids, ", ")))
		} else {
			age := sys.Now().Sub(lastSync).Round(time.Minute)
			rep.Warn(fmt.Sprintf(
				"%s: not found in local vault cache (last synced %s ago), named by %s — run `dotf secrets unlock` (syncs the daemon's cache) to refresh; if missing from vault, every unscoped `dotf secrets run` fails on it",
				item, age, strings.Join(ids, ", ")))
		}
	}
	if len(missing) == 0 {
		rep.Pass(fmt.Sprintf("all %d bw item(s) named by the registry exist in the vault", len(declared)))
	}
}
