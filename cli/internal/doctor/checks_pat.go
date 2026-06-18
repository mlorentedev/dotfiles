package doctor

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
// key — github.token is mapped by two env vars) and a representative env var to
// read the live token value from.
type patSecret struct {
	filename string // e.g. "github.token"
	envVar   string // e.g. "GITHUB_PERSONAL_ACCESS_TOKEN"
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
		rep.Skip("no github.* PAT-backed secrets in env-mapping.conf")
		return
	}

	warnDays := patWarnDays(sys, rep)

	for _, s := range secrets {
		token := sys.Getenv(s.envVar)
		if token == "" {
			rep.Skip(fmt.Sprintf("%s (%s) not in environment — run secrets_refresh", s.filename, s.envVar))
			continue
		}

		status, hdr, err := sys.HTTPGet(githubAPIUser, map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "application/vnd.github+json",
			"User-Agent":    "dotf-doctor",
		})
		switch {
		case err != nil:
			rep.Warn(fmt.Sprintf("%s: could not reach api.github.com (%v) — liveness check skipped", s.filename, err))
			continue
		case status == http.StatusUnauthorized:
			rep.Fail(fmt.Sprintf("%s: token invalid or expired (HTTP 401) — rotate it", s.filename))
			continue
		case status != http.StatusOK:
			rep.Warn(fmt.Sprintf("%s: unexpected HTTP %d from api.github.com — liveness inconclusive", s.filename, status))
			continue
		}

		// 200 OK: the token authenticates. Worst case from here is "rotate soon"
		// (WARN), never FAIL — a token that just succeeded is not dead.
		raw := strings.TrimSpace(hdr.Get(patExpiryHeader))
		if raw == "" {
			rep.Pass(fmt.Sprintf("%s: valid, no expiry set", s.filename))
			continue
		}
		exp, perr := parsePATExpiry(raw)
		if perr != nil {
			rep.Warn(fmt.Sprintf("%s: valid, but could not parse expiry %q — %v", s.filename, raw, perr))
			continue
		}

		day := exp.Format("2006-01-02")
		days := int(exp.Sub(sys.Now()).Hours() / 24)
		switch {
		case days <= 0:
			rep.Warn(fmt.Sprintf("%s: at/just past its stated expiry (%s) but still accepted — rotate now", s.filename, day))
		case days <= warnDays:
			rep.Warn(fmt.Sprintf("%s: expires in %d day(s) (%s) — rotate soon", s.filename, days, day))
		default:
			rep.Pass(fmt.Sprintf("%s: valid, expires in %d day(s) (%s)", s.filename, days, day))
		}
	}
}

// githubPATSecrets parses env-mapping.conf and returns the unique github.*
// PAT-backed secrets, one per .age filename (github.token is mapped by both
// GITHUB_PERSONAL_ACCESS_TOKEN and RELEASE_TOKEN — probed once). The first env
// var seen for a filename is the representative value source. A missing or
// unreadable mapping yields nil (checkSecrets owns the "mapping exists" assertion).
func githubPATSecrets(cfg *Config) []patSecret {
	mapping := filepath.Join(cfg.DotfilesDir, "sensitive", "env-mapping.conf")
	raw, err := os.ReadFile(mapping)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var out []patSecret
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		varName, val, _ := strings.Cut(line, "=")
		varName = strings.TrimSpace(varName)
		val = strings.TrimSpace(val)

		// File secrets (@VAR=file>dest): the filename is the part before '>'.
		fname := val
		if strings.HasPrefix(varName, "@") {
			fname, _, _ = strings.Cut(val, ">")
			fname = strings.TrimSpace(fname)
			varName = strings.TrimPrefix(varName, "@")
		}
		if !strings.HasPrefix(fname, "github.") || seen[fname] {
			continue
		}
		seen[fname] = true
		out = append(out, patSecret{filename: fname, envVar: varName})
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
