package doctor

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

const (
	// defaultPATWarnDays is the lead time before expiry at which the check WARNs.
	// Overridable via DOTF_PAT_EXPIRY_WARN_DAYS.
	defaultPATWarnDays = 14
	// githubAPIUser is the cheapest authenticated REST endpoint; its response
	// carries the expiry header for the presented token.
	githubAPIUser = "https://api.github.com/user"
	// patExpiryHeader is the response header GitHub sets on authenticated REST
	// calls for tokens that have an expiry (classic + fine-grained PATs).
	patExpiryHeader = "github-authentication-token-expiration"
)

// patValidateKey is the registry's declared marker for "this secret is a GitHub
// token whose liveness can be probed" (`validate: github-token`). Selecting on it
// rather than on an age-blob name prefix is what makes the check backend-neutral:
// a bw-backed PAT has no age filename to match, which is how BITACORA_PAT became
// invisible to this section the day it migrated (#961, REFACTOR-012). The same
// marker gates `dotf secrets sync ci`'s pre-upload probe.
const patValidateKey = "github-token"

// patSecret is one GitHub PAT-backed secret: the entry to resolve, a display name,
// and every env var that maps to the same underlying source. github.token is mapped
// by BOTH GITHUB_PERSONAL_ACCESS_TOKEN and RELEASE_TOKEN, so the aliases are kept
// together — the token is probed once and named by whichever var the reader knows.
type patSecret struct {
	name    string        // display: age base name, or bw item/field
	entry   secrets.Entry // the entry resolved for the probe
	envVars []string      // every var mapping to this source
}

// checkPATExpiry probes each GitHub PAT-backed secret for liveness + days to
// expiry — the complement to checkSecrets, which only validates that the .age
// blobs exist. Classification: HTTP 401 → FAIL (the dead-token case that broke
// release-please CI, and the only branch that drives a non-zero exit); expiring
// within the threshold → WARN; healthy with runway → PASS; secret not
// provisioned here, or a locked vault → SKIP (no alarm, but stated as
// unmonitored); unresolvable or network unreachable → WARN (neither is a setup
// failure). It is network-bound, so doctor.go runs it in the full sweep only —
// never under --quick (the SessionStart hot path).
//
// Which secrets it covers is the registry's call, not a naming convention's:
// everything marked `validate: github-token`, whatever backend it lives on.
func checkPATExpiry(sys *System, cfg *Config, rep *Report) {
	rep.Section("PAT expiry")

	pats := githubPATSecrets(cfg) // not `secrets`: that name is the package this file resolves through
	if len(pats) == 0 {
		rep.Skip("no secrets marked `validate: " + patValidateKey + "` in the registry")
		return
	}

	warnDays := patWarnDays(sys, rep)
	for _, s := range pats {
		probePATSecret(sys, s, warnDays, rep)
	}
}

// resolvePATToken resolves the secret's value through the Loader seam — the same
// read path `dotf secrets run` uses — and reports the outcome when it cannot.
// It returns ok=false when the caller must not probe.
//
// It does NOT read the environment. Under ADR-028 secrets are never exported into
// the ambient shell, so the previous env read resolved to "" on every correctly
// configured machine: the whole section SKIPped, and told the reader to run
// `secrets_refresh`, a command retired with the login-time loader. A check that
// cannot pass on the architecture it ships with is not a check (REFACTOR-012).
//
// A bw-backed secret is gated on the serve daemon's own state rather than
// attempted optimistically: resolution against a locked vault is the ~1.5s-per-
// secret shellout that made shell startup hang for 45 seconds (BUG-080), and
// "locked" is a fact the daemon reports directly, so there is nothing to learn
// from trying.
func resolvePATToken(sys *System, s patSecret, rep *Report) (string, bool) {
	if s.entry.Backend == secrets.BackendBW {
		state, err := sys.BWServeStatus()
		switch {
		case err != nil:
			rep.Warn(fmt.Sprintf("%s: could not determine Bitwarden state (%v) — liveness check skipped", s.name, err))
			return "", false
		case state != "unlocked":
			rep.Skip(fmt.Sprintf("%s (%s) is bw-backed and the vault is %s — run `dotf secrets unlock`",
				s.name, strings.Join(s.envVars, ", "), state))
			return "", false
		}
	}

	token, err := sys.ResolveSecret(s.entry)
	if err != nil {
		reportPATResolveFailure(s, err, rep)
		return "", false
	}
	return token, true
}

// reportPATResolveFailure classifies a resolution error into this section's
// severity vocabulary, deliberately WITHOUT adding a FAIL branch.
//
// HTTP 401 stays the only FAIL here, and therefore the only branch that drives a
// non-zero exit from `dotf doctor`. Resolution introduces failure modes that did
// not exist when that contract was written — an absent secret, a backend that
// cannot produce a declared one — and either could be argued into a FAIL. Both
// are refused: doctor is the last step of setup-linux.sh, so a new non-zero
// branch fails the bootstrap of every machine caught mid-migration, and changing
// a fleet-wide exit contract is a policy decision, not a side effect of repairing
// a tagged-union consumer (REFACTOR-012).
//
// ErrSecretAbsent inherits the role the env-unset SKIP used to play: this machine
// does not have this secret, which is not an alarm. But the SKIP must not read as
// "nothing to do" — an unmonitored token passing silently is the whole defect
// being fixed — so it says outright that the expiry is not being watched here.
// Every message names a command that exists; the one it replaces did not.
func reportPATResolveFailure(s patSecret, err error, rep *Report) {
	aliases := strings.Join(s.envVars, ", ")
	if errors.Is(err, secrets.ErrSecretAbsent) {
		rep.Skip(fmt.Sprintf("%s (%s) is not provisioned on this machine — its expiry is NOT being monitored here; `dotf secrets verify` lists what is missing",
			s.name, aliases))
		return
	}
	rep.Warn(fmt.Sprintf("%s (%s) is declared but did not resolve (%v) — its expiry is NOT being monitored; `dotf secrets verify` isolates the cause",
		s.name, aliases, err))
}

// probePATSecret runs the liveness probe for one secret and reports its outcome.
// Split out of checkPATExpiry to keep both within the < 40-line / < 10-complexity
// budget (AGENTS.md). HTTP 401 → FAIL (the only non-zero-exit branch); transport
// error or non-200 → WARN; 200 hands off to reportPATExpiry for the expiry math.
func probePATSecret(sys *System, s patSecret, warnDays int, rep *Report) {
	token, ok := resolvePATToken(sys, s, rep)
	if !ok {
		return
	}

	status, hdr, err := sys.HTTPGet(githubAPIUser, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
		"User-Agent":    "dotf-doctor",
	})
	switch {
	case err != nil:
		rep.Warn(fmt.Sprintf("%s: could not reach api.github.com (%v) — liveness check skipped", s.name, err))
	case status == http.StatusUnauthorized:
		// A 401 is also what a STALE VAULT CACHE looks like: the daemon served
		// the token's previous value (CLI-056, #1316). Name that first — it is
		// the cheaper and, measured once, the likelier cause.
		rep.Fail(fmt.Sprintf("%s: token rejected (HTTP 401) — if the vault cache is stale run `dotf secrets unlock` (it syncs); otherwise rotate it", s.name))
	case status != http.StatusOK:
		rep.Warn(fmt.Sprintf("%s: unexpected HTTP %d from api.github.com — liveness inconclusive", s.name, status))
	default:
		// 200 OK: the token authenticates. Worst case from here is "rotate soon"
		// (WARN), never FAIL — a token that just succeeded is not dead.
		reportPATExpiry(s.name, hdr, sys.Now(), warnDays, rep)
	}
}

// reportPATExpiry classifies the expiry header on a 200 response: absent header
// (non-expiring token) → PASS; at/past expiry → WARN; within warnDays → WARN;
// otherwise PASS with the runway.
func reportPATExpiry(filename string, hdr http.Header, now time.Time, warnDays int, rep *Report) {
	raw := strings.TrimSpace(hdr.Get(patExpiryHeader))
	if raw == "" {
		rep.Pass(fmt.Sprintf("%s: valid, no expiry set", filename))
		return
	}
	exp, err := parsePATExpiry(raw)
	if err != nil {
		rep.Warn(fmt.Sprintf("%s: valid, but could not parse expiry %q — %v", filename, raw, err))
		return
	}

	day := exp.Format("2006-01-02")
	days := int(exp.Sub(now).Hours() / 24)
	switch {
	case days <= 0:
		rep.Warn(fmt.Sprintf("%s: at/just past its stated expiry (%s) but still accepted — rotate now", filename, day))
	case days <= warnDays:
		rep.Warn(fmt.Sprintf("%s: expires in %d day(s) (%s) — rotate soon", filename, days, day))
	default:
		rep.Pass(fmt.Sprintf("%s: valid, expires in %d day(s) (%s)", filename, days, day))
	}
}

// githubPATSecrets reads secrets/registry.yaml and returns the unique github.*
// PAT-backed secrets, one per age source (the dedupe key), each carrying ALL of
// its env aliases (github.token is exposed as both GITHUB_PERSONAL_ACCESS_TOKEN
// and RELEASE_TOKEN — kept together so it is probed once but resolvable from
// either). A missing or invalid registry yields nil (checkSecrets owns the
// "registry exists" assertion). The age source is the .age basename, so it
// doubles as the probe's display name (e.g. "github.token").
func githubPATSecrets(cfg *Config) []patSecret {
	reg, err := loadRegistry(cfg)
	if err != nil {
		return nil
	}

	idx := map[string]int{} // Entry.SourceID() → index into out
	var out []patSecret
	for _, e := range reg.Entries("") { // home unused: github secrets are env, not file
		if e.IsFile || e.Validate != patValidateKey {
			continue
		}
		// Dedupe on the union's own identity, never on File alone: File is ""
		// for every bw entry, so a File key collapsed all of them into one
		// secret — the same defect as the filter above, one line down.
		id := e.SourceID()
		if i, ok := idx[id]; ok {
			out[i].envVars = append(out[i].envVars, e.Var)
			continue
		}
		idx[id] = len(out)
		out = append(out, patSecret{name: e.SourceDisplay(), entry: e, envVars: []string{e.Var}})
	}
	return out
}

// parsePATExpiry parses the github-authentication-token-expiration header. The
// observed live format is "2026-09-15 07:11:31 UTC"; numeric-offset and RFC3339
// forms are accepted as fallbacks.
func parsePATExpiry(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		time.RFC3339,
	}
	var firstErr error
	for _, l := range layouts {
		t, err := time.Parse(l, s)
		if err == nil {
			return t, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return time.Time{}, firstErr
}

// patWarnDays resolves the WARN lead-time (days), defaulting to defaultPATWarnDays
// and overridable via DOTF_PAT_EXPIRY_WARN_DAYS. A non-numeric / negative override
// is ignored with a WARN, so a typo never silently disables the threshold.
func patWarnDays(sys *System, rep *Report) int {
	raw := strings.TrimSpace(sys.Getenv("DOTF_PAT_EXPIRY_WARN_DAYS"))
	if raw == "" {
		return defaultPATWarnDays
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		rep.Warn(fmt.Sprintf("DOTF_PAT_EXPIRY_WARN_DAYS=%q is not a non-negative integer — using default %d", raw, defaultPATWarnDays))
		return defaultPATWarnDays
	}
	return n
}
