package doctor

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// patSecret is a GitHub PAT-backed mapping: the age-blob filename (the dedupe
// key) plus every env var that maps to it. github.token is mapped by BOTH
// GITHUB_PERSONAL_ACCESS_TOKEN and RELEASE_TOKEN, so all aliases are kept — any
// one being exported is enough to probe the token (the first non-empty wins).
type patSecret struct {
	filename string   // e.g. "github.token"
	envVars  []string // e.g. ["GITHUB_PERSONAL_ACCESS_TOKEN", "RELEASE_TOKEN"]
}

// checkPATExpiry probes each GitHub PAT-backed secret for liveness + days to
// expiry — the complement to checkSecrets, which only validates that the .age
// blobs exist. Classification: HTTP 401 → FAIL (the dead-token case that broke
// release-please CI, and the only branch that drives a non-zero exit); expiring
// within the threshold → WARN; healthy with runway → PASS; token not in the
// environment → SKIP (fresh shell, no alarm); network unreachable → WARN
// (offline is not a setup failure). It is network-bound, so doctor.go runs it in
// the full sweep only — never under --quick (the SessionStart hot path).
func checkPATExpiry(sys *System, cfg *Config, rep *Report) {
	rep.Section("PAT expiry")

	secrets := githubPATSecrets(cfg)
	if len(secrets) == 0 {
		rep.Skip("no github.* PAT-backed secrets in the registry")
		return
	}

	warnDays := patWarnDays(sys, rep)
	for _, s := range secrets {
		probePATSecret(sys, s, warnDays, rep)
	}
}

// resolvePATToken returns the first non-empty value among the secret's env
// aliases. github.token is mapped by both GITHUB_PERSONAL_ACCESS_TOKEN and
// RELEASE_TOKEN; either being exported is enough to probe it, so a single unset
// alias never produces a false SKIP.
func resolvePATToken(sys *System, s patSecret) string {
	for _, v := range s.envVars {
		if val := sys.Getenv(v); val != "" {
			return val
		}
	}
	return ""
}

// probePATSecret runs the liveness probe for one secret and reports its outcome.
// Split out of checkPATExpiry to keep both within the < 40-line / < 10-complexity
// budget (AGENTS.md). HTTP 401 → FAIL (the only non-zero-exit branch); transport
// error or non-200 → WARN; 200 hands off to reportPATExpiry for the expiry math.
func probePATSecret(sys *System, s patSecret, warnDays int, rep *Report) {
	token := resolvePATToken(sys, s)
	if token == "" {
		rep.Skip(fmt.Sprintf("%s (%s) not in environment — run secrets_refresh", s.filename, strings.Join(s.envVars, ", ")))
		return
	}

	status, hdr, err := sys.HTTPGet(githubAPIUser, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
		"User-Agent":    "dotf-doctor",
	})
	switch {
	case err != nil:
		rep.Warn(fmt.Sprintf("%s: could not reach api.github.com (%v) — liveness check skipped", s.filename, err))
	case status == http.StatusUnauthorized:
		rep.Fail(fmt.Sprintf("%s: token invalid or expired (HTTP 401) — rotate it", s.filename))
	case status != http.StatusOK:
		rep.Warn(fmt.Sprintf("%s: unexpected HTTP %d from api.github.com — liveness inconclusive", s.filename, status))
	default:
		// 200 OK: the token authenticates. Worst case from here is "rotate soon"
		// (WARN), never FAIL — a token that just succeeded is not dead.
		reportPATExpiry(s.filename, hdr, sys.Now(), warnDays, rep)
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

	idx := map[string]int{} // age source → index into out
	var out []patSecret
	for _, e := range reg.Entries("") { // home unused: github secrets are env, not file
		if e.IsFile || !strings.HasPrefix(e.File, "github.") {
			continue
		}
		if i, ok := idx[e.File]; ok {
			out[i].envVars = append(out[i].envVars, e.Var)
			continue
		}
		idx[e.File] = len(out)
		out = append(out, patSecret{filename: e.File, envVars: []string{e.Var}})
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
