package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// bwStaleSync is how long the Bitwarden vault may go unsynced before doctor
// warns.
//
// It is a proxy for token liveness, and the only one available WITHOUT an
// unlocked session. `bw status` is served from local state: it keeps reporting
// "locked" indefinitely after the server has expired the refresh token, which is
// exactly how BUG-074 stayed invisible for 45 days — the operator discovered it
// only by running `bw unlock` by hand and getting `invalid_grant` (HTTP 400).
// Elapsed time since the last SUCCESSFUL sync is the observable that does move.
//
// 30d is deliberately shorter than the observed 45d failure: the warning has to
// land while the token is still renewable, not once it is already dead.
const bwStaleSync = 30 * 24 * time.Hour

// Deadlines for the two bw subprocesses. `status` is served from local state and
// should be near-instant, so its budget only has to cover a cold Node start;
// `sync` is a real network round-trip against the Bitwarden API and legitimately
// takes seconds on a large vault. Both are bounded because doctor is the last
// step of setup-linux.sh — an unbounded hang here hangs a bootstrap.
const (
	bwStatusTimeout = 15 * time.Second
	bwSyncTimeout   = 45 * time.Second
)

// bwState is the subset of `bw status` this check consumes. Bitwarden emits more
// fields (serverUrl, userEmail, userId); parsing only what is used keeps the
// check indifferent to upstream additions.
type bwState struct {
	Status   string `json:"status"`   // unauthenticated | locked | unlocked
	LastSync string `json:"lastSync"` // RFC3339, or empty when never synced
}

// checkBitwardenReach proves the LIVE secrets SSOT is reachable, not merely
// installed (BUG-074).
//
// The defect this replaces: checkSecretsTooling proved the age floor — the
// BACKUP tier — by behaviour (a real encrypt/decrypt round-trip of a sentinel),
// while proving Bitwarden — the tier ADR-028 designates the live SSOT — by
// `sys.has("bw")`, i.e. a binary on PATH. So doctor printed a green
// "bw (Bitwarden CLI — live secrets SSOT) found" against a vault that could not
// be reached at all. That is the class already named by #898 (a check never
// observed failing is not evidence) and #852 (verify behaviour, not
// representation).
//
// Three tiers, because what can be proven depends on what the operator has:
//
//  1. `bw status` (offline, always): catches `unauthenticated` — a definite,
//     actionable break.
//  2. lastSync age (offline, always): catches the silent pre-expiry window that
//     status alone reports as healthy. This is the tier that would have caught
//     the incident.
//  3. an authenticated round-trip (only with an unlocked vault): the true
//     analogue of the age round-trip.
//
// SEVERITY is keyed to real exposure via BWBackedSecrets, not to a flat policy.
// With every registry entry still on `backend: age`, an unreachable Bitwarden
// breaks nothing, and a red doctor for a harmless condition is how operators are
// trained to ignore red doctors. From the first `backend: bw` entry onward the
// same state is a hard failure, so it FAILs.
func checkBitwardenReach(sys *System, rep *Report) {
	rep.Section("Bitwarden reach (live secrets SSOT)")

	if !sys.has("bw") {
		// checkSecretsTooling already FAILs on absent bw; repeating it here would
		// report one defect twice and inflate the failure count.
		rep.Skip("bw not in PATH — reach unverifiable (secrets tooling owns that FAIL)")
		return
	}

	live, regErr := sys.BWBackedSecrets()
	if regErr != nil {
		// An unreadable registry is its own defect, owned by the secrets checks.
		// Here it only costs the severity signal, so degrade to the safe side
		// (treat as unexposed) and say so rather than guessing.
		rep.Warn("registry unreadable (" + regErr.Error() + ") — reach severity degraded to advisory")
	}

	out, err := sys.CommandOutputBounded(bwStatusTimeout, "bw", "status")
	if err != nil {
		rep.Warn("`bw status` did not run (" + bwFailDetail(out, err) + ") — reach unverified")
		return
	}
	var st bwState
	if jErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &st); jErr != nil {
		rep.Warn("`bw status` returned no parseable JSON — reach unverified")
		return
	}

	switch st.Status {
	case "unauthenticated":
		// `bw unlock` CANNOT recover this state, and its master-password prompt
		// makes it look like a credential problem. Naming the right verb is the
		// whole value of the check at this point.
		msg := "Bitwarden session is gone (`bw status`: unauthenticated) — run `bw login`, not `bw unlock`: an expired refresh token cannot be unlocked"
		if live > 0 {
			rep.Fail(fmt.Sprintf("%s — %d registry secret(s) resolve through it and cannot be read", msg, live))
			rep.Fix("export BW_SESSION=$(bw login --raw) && bw sync")
			return
		}
		rep.Warn(msg + " — no backend:bw secret depends on it yet")
		return
	case "locked", "unlocked":
		// Both are authenticated states; keep going.
	default:
		rep.Warn("unrecognised `bw status` value " + strconv.Quote(st.Status) + " — reach unverified")
		return
	}

	checkBWSyncAge(sys, rep, st.LastSync)

	if st.Status == "locked" {
		// Not a defect: a locked vault is the normal resting state. But say
		// plainly that reach was NOT exercised, so a green section is never read
		// as proof the token still works.
		rep.Info("vault locked — token not exercised; `export BW_SESSION=$(bw unlock --raw)` before re-running to prove reach")
		return
	}

	// Tier 3. `bw sync` is the correct deep probe because it is the operation
	// that actually refreshes the access token against the server — the exact
	// path that failed in the incident. `bw list` would not do: it is served from
	// the local cache and passes against a dead token.
	//
	// It mutates local state (it advances lastSync and renews the token) inside
	// what is otherwise a read-only diagnostic. That is deliberate — but the
	// honest claim is narrower than "doctor is the keep-alive". This tier only
	// runs on an UNLOCKED vault, and a locked vault is the resting state; nothing
	// runs doctor on a timer either. So the renewal is OPPORTUNISTIC: it happens
	// when an operator already has a session open, and not at all otherwise.
	//
	// The tier that would actually have prevented the 45-day expiry is tier 2,
	// which needs no session: the staleness warning fires on a locked vault and
	// names `bw sync` while the token is still renewable. Tier 3 proves reach;
	// tier 2 is what catches the silent expiry.
	if syncOut, syncErr := sys.CommandOutputBounded(bwSyncTimeout, "bw", "sync"); syncErr != nil {
		rep.Fail("Bitwarden sync FAILED on an unlocked vault (" + bwFailDetail(syncOut, syncErr) + ") — the live SSOT is not reachable")
		return
	}
	rep.Pass("Bitwarden reach verified (authenticated sync round-trip)")
}

// checkBWSyncAge reports how long ago the vault last synced. Split out so the
// staleness rule — the tier that would have caught BUG-074 — is testable on its
// own, independent of which status the vault happens to be in.
func checkBWSyncAge(sys *System, rep *Report, lastSync string) {
	if lastSync == "" {
		rep.Warn("Bitwarden has never synced — reach unproven (`bw sync`)")
		return
	}
	ts, err := time.Parse(time.RFC3339, lastSync)
	if err != nil {
		rep.Warn("Bitwarden lastSync is unparseable (" + strconv.Quote(lastSync) + ")")
		return
	}
	age := sys.Now().Sub(ts)
	if age < 0 {
		// A sync stamped in the future means the clock moved, not that the vault
		// is fresh. Reporting "synced -3d ago" would be nonsense, and silently
		// treating it as fresh would hide a genuinely stale token behind a skew.
		rep.Warn("Bitwarden lastSync is in the future — check the system clock; staleness cannot be judged")
		return
	}
	days := int(age.Hours() / 24)
	if age > bwStaleSync {
		rep.Warn(fmt.Sprintf("Bitwarden last synced %dd ago (>%dd) — the refresh token expires silently on an idle vault; run `bw sync` (BUG-074)",
			days, int(bwStaleSync.Hours()/24)))
		return
	}
	rep.Pass(fmt.Sprintf("Bitwarden synced %dd ago", days))
}

// bwBackedSecrets is the production BWBackedSecrets seam: it counts registry
// entries whose backend is Bitwarden.
//
// It reads env.RepoRegistryPath — the CHECKOUT-only path — and deliberately not
// cfg.DotfilesDir nor env.ResolveRegistryPath. The deployed copy under
// ~/.dotfiles lags the checkout until a setup run (doctor has a whole section
// for that drift), so counting there reports zero bw-backed secrets while the
// checkout is actively flipping entries to bw — pinning severity at advisory
// during exactly the migration window this check guards (ADR-030, #635).
//
// ResolveRegistryPath is the wrong seam for the same reason even though it
// "prefers" the checkout: when a checkout exists but its registry is absent, it
// silently falls back to the deployed copy — reintroducing the stale count
// through the back door. RepoRegistryPath fails loud instead, and the caller
// degrades severity with a stated reason rather than trusting a number whose
// source it cannot name. A machine with no checkout at all therefore reports
// advisory: there, the check genuinely cannot tell a migrated registry from a
// stale mirror.
func bwBackedSecrets() (int, error) {
	path, err := env.RepoRegistryPath()
	if err != nil {
		return 0, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	reg, err := secrets.ParseRegistry(raw)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range reg.Secrets {
		if s.Backend == "bw" {
			n++
		}
	}
	return n, nil
}

// bwFailDetail renders a failed bw invocation as one report line: the CLI's own
// first output line when it produced one, else the exec error.
//
// Both shapes occur. For expected states bw exits non-zero with a clean single
// line ("You are not logged in."), but on an expired refresh token it dies with a
// Node unhandled-promise-rejection stacktrace — reusing firstLine (the existing
// helper in checks_vault_hooks.go) collapses either into something readable
// instead of flooding the section with a stacktrace.
func bwFailDetail(out string, err error) string {
	if d := firstLine(out); d != "" {
		return d
	}
	return err.Error()
}
