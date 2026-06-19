package doctor

import (
	"bytes"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// patOneSecret is the minimal env-mapping.conf for the single-secret cases: one
// github.* PAT-backed mapping, read from GITHUB_PERSONAL_ACCESS_TOKEN.
const patOneSecret = "GITHUB_PERSONAL_ACCESS_TOKEN=github.token\n"

// patCfg builds a Config rooted at a temp dotfiles dir with the given
// env-mapping.conf, mirroring how checkPATExpiry resolves the mapping.
func patCfg(t *testing.T, mapping string) *Config {
	t.Helper()
	dotfiles := t.TempDir()
	writeFile(t, filepath.Join(dotfiles, "sensitive", "env-mapping.conf"), mapping)
	return &Config{DotfilesDir: dotfiles}
}

// expiryHeader renders an expiry `days` from the fixed test clock in the exact
// live layout GitHub returns ("2026-09-15 07:11:31 UTC", per the captured
// rotated token), so the parse path under test sees real-world input.
func expiryHeader(days int) string {
	return fixedTestNow.AddDate(0, 0, days).Format("2006-01-02 15:04:05 MST")
}

// TestCheckPATExpiry_Classification is the AC2/AC3 table: one row per
// classification branch, each driven by a canned HTTPGet + the fixed clock, so
// the SKIP/WARN/FAIL/PASS decision is fully deterministic and offline.
func TestCheckPATExpiry_Classification(t *testing.T) {
	days := func(n int) *int { return &n }

	cases := []struct {
		name string
		// HTTP probe result.
		status  int
		expiry  *int // header offset (days) from fixedTestNow; nil ⇒ header absent
		httpErr bool
		// environment.
		tokenSet bool   // GITHUB_PERSONAL_ACCESS_TOKEN present?
		warnDays string // DOTF_PAT_EXPIRY_WARN_DAYS override; "" ⇒ unset
		// expectations.
		wantFailures int
		wantSubstr   string
	}{
		// AC2 — the five branches the spec enumerates.
		{"http 401 → FAIL", 401, nil, false, true, "", 1, "token invalid or expired (HTTP 401)"},
		{"expiry within threshold → WARN", 200, days(5), false, true, "", 0, "expires in 5 day(s)"},
		{"valid with runway → PASS", 200, days(60), false, true, "", 0, "valid, expires in 60 day(s)"},
		{"token env-unset → SKIP", 200, days(60), false, false, "", 0, "not in environment — run secrets_refresh"},
		{"network error → WARN", 0, nil, true, true, "", 0, "could not reach api.github.com"},

		// Remaining branches in the function (each is a distinct decision).
		{"200 no expiry header → PASS", 200, nil, false, true, "", 0, "valid, no expiry set"},
		{"unexpected 500 → WARN", 500, nil, false, true, "", 0, "unexpected HTTP 500"},
		{"at/just past expiry but 200 → WARN", 200, days(-1), false, true, "", 0, "at/just past its stated expiry"},

		// AC3 — the same 20-day runway flips PASS→WARN as the threshold widens.
		{"AC3 default 14: 20d is PASS", 200, days(20), false, true, "", 0, "valid, expires in 20 day(s)"},
		{"AC3 override 30: 20d is WARN", 200, days(20), false, true, "30", 0, "expires in 20 day(s) (2026-07-07) — rotate soon"},
		{"AC3 bad override: default + WARN", 200, days(60), false, true, "abc", 0, `DOTF_PAT_EXPIRY_WARN_DAYS="abc" is not a non-negative integer`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{}
			if tc.tokenSet {
				env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "tok"
			}
			if tc.warnDays != "" {
				env["DOTF_PAT_EXPIRY_WARN_DAYS"] = tc.warnDays
			}
			sys := newSys(env, nil, nil)
			sys.Now = func() time.Time { return fixedTestNow }
			sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
				if tc.httpErr {
					return 0, nil, errors.New("dial tcp: offline")
				}
				h := http.Header{}
				if tc.expiry != nil {
					h.Set(patExpiryHeader, expiryHeader(*tc.expiry))
				}
				return tc.status, h, nil
			}

			var buf bytes.Buffer
			rep := capture(&buf)
			checkPATExpiry(sys, patCfg(t, patOneSecret), rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// TestCheckPATExpiry_ProbesEachFilenameOnce covers AC1: github.token is mapped
// by two env vars yet must be probed exactly once (dedupe by .age filename), and
// a non-github secret is never probed at all.
func TestCheckPATExpiry_ProbesEachFilenameOnce(t *testing.T) {
	mapping := "GITHUB_PERSONAL_ACCESS_TOKEN=github.token\n" +
		"RELEASE_TOKEN=github.token\n" + // same filename → must NOT add a second probe
		"DOCKERHUB_TOKEN=dockerhub.token\n" // not github.* → must be ignored

	env := map[string]string{
		"GITHUB_PERSONAL_ACCESS_TOKEN": "tok",
		"RELEASE_TOKEN":                "tok",
		"DOCKERHUB_TOKEN":              "tok",
	}
	sys := newSys(env, nil, nil)
	sys.Now = func() time.Time { return fixedTestNow }

	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, mapping), rep)

	if calls != 1 {
		t.Fatalf("want exactly 1 probe (github.token deduped, dockerhub ignored); got %d\n%s", calls, buf.String())
	}
}

// TestCheckPATExpiry_FallsBackToSecondAlias guards the alias-resolution fix:
// github.token is mapped by both GITHUB_PERSONAL_ACCESS_TOKEN and RELEASE_TOKEN.
// When only the SECOND alias is exported, the token must still be probed — a
// single unset alias must not yield a false SKIP.
func TestCheckPATExpiry_FallsBackToSecondAlias(t *testing.T) {
	mapping := "GITHUB_PERSONAL_ACCESS_TOKEN=github.token\n" +
		"RELEASE_TOKEN=github.token\n"

	sys := newSys(map[string]string{"RELEASE_TOKEN": "tok"}, nil, nil) // first alias unset
	sys.Now = func() time.Time { return fixedTestNow }
	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, mapping), rep)

	if calls != 1 {
		t.Fatalf("a set fallback alias must be probed, not SKIPped; got %d probe(s)\n%s", calls, buf.String())
	}
	if strings.Contains(buf.String(), "not in environment") {
		t.Fatalf("must not report SKIP when a fallback alias is set\n%s", buf.String())
	}
}

// TestCheckPATExpiry_QuickSkipsProbe covers AC4: the SessionStart hot path
// (--quick) must stay fork-free and offline, so a full Run with Quick:true makes
// zero HTTP probes even with a token in the environment.
func TestCheckPATExpiry_QuickSkipsProbe(t *testing.T) {
	dotfiles := t.TempDir()
	writeFile(t, filepath.Join(dotfiles, "sensitive", "env-mapping.conf"), patOneSecret)

	sys := newSys(map[string]string{
		"DOTFILES_DIR":                 dotfiles,
		"HOME":                         dotfiles,
		"GITHUB_PERSONAL_ACCESS_TOKEN": "tok",
	}, nil, nil)

	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	if _, err := Run(Options{Quick: true, System: sys, StartDir: dotfiles, Out: &buf}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("--quick must not probe PATs; got %d HTTP call(s)\n%s", calls, buf.String())
	}
}

// TestCheckPATExpiry_NoSecrets covers the empty-mapping branch: no github.*
// secrets ⇒ a single SKIP and no probe (e.g. a checkout without a github PAT).
func TestCheckPATExpiry_NoSecrets(t *testing.T) {
	sys := newSys(nil, nil, nil)
	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, "DOCKERHUB_TOKEN=dockerhub.token\n"), rep)

	if calls != 0 {
		t.Fatalf("no github.* secrets must mean no probe; got %d\n%s", calls, buf.String())
	}
	if !strings.Contains(buf.String(), "no github.* PAT-backed secrets") {
		t.Fatalf("expected the no-secrets SKIP message\n%s", buf.String())
	}
}
