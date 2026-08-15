package doctor

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// patOneSecret is the minimal registry for the single-secret cases: one
// age-backed GitHub PAT, marked with the registry's own liveness marker.
// Selection keys on `validate:`, never on the age filename (REFACTOR-012).
const patOneSecret = "version: 1\nsecrets:\n" +
	"  - {id: github-token, plane: app, backend: age, age: github.token, validate: github-token, expose: {env: GITHUB_PERSONAL_ACCESS_TOKEN}}\n"

// patCfg builds a Config rooted at a temp dotfiles dir with the given
// secrets/registry.yaml, mirroring how checkPATExpiry resolves PAT secrets.
func patCfg(t *testing.T, registry string) *Config {
	t.Helper()
	dotfiles := t.TempDir()
	writeFile(t, filepath.Join(dotfiles, "secrets", "registry.yaml"), registry)
	return &Config{DotfilesDir: dotfiles}
}

// resolvesTo makes every entry resolve to tok — the "secrets are provisioned and
// the vault is reachable" box.
func resolvesTo(tok string) func(secrets.Entry) (string, error) {
	return func(secrets.Entry) (string, error) { return tok, nil }
}

// expiryHeader renders an expiry `days` from the fixed test clock in the exact
// live layout GitHub returns ("2026-09-15 07:11:31 UTC", per the captured
// rotated token), so the parse path under test sees real-world input.
func expiryHeader(days int) string {
	return fixedTestNow.AddDate(0, 0, days).Format("2006-01-02 15:04:05 MST")
}

// TestCheckPATExpiry_Classification is the severity table: one row per branch,
// each driven by a canned resolver + HTTPGet + the fixed clock, so the
// SKIP/WARN/FAIL/PASS decision is fully deterministic and offline.
//
// The contract it pins is that HTTP 401 is the ONLY row producing a failure.
// Resolution failures are reported, never escalated: doctor is the last step of
// setup-linux.sh and a new non-zero branch would fail the bootstrap of every
// machine caught mid-migration (REFACTOR-012 AC4).
func TestCheckPATExpiry_Classification(t *testing.T) {
	days := func(n int) *int { return &n }

	cases := []struct {
		name string
		// HTTP probe result.
		status  int
		expiry  *int // header offset (days) from fixedTestNow; nil ⇒ header absent
		httpErr bool
		// resolution result.
		resolveErr error // nil ⇒ resolves to a token
		// environment.
		warnDays string // DOTF_PAT_EXPIRY_WARN_DAYS override; "" ⇒ unset
		// expectations.
		wantFailures int
		wantSubstr   string
	}{
		// The probe branches.
		{"http 401 → FAIL", 401, nil, false, nil, "", 1, "token invalid or expired (HTTP 401)"},
		{"expiry within threshold → WARN", 200, days(5), false, nil, "", 0, "expires in 5 day(s)"},
		{"valid with runway → PASS", 200, days(60), false, nil, "", 0, "valid, expires in 60 day(s)"},
		{"network error → WARN", 0, nil, true, nil, "", 0, "could not reach api.github.com"},
		{"200 no expiry header → PASS", 200, nil, false, nil, "", 0, "valid, no expiry set"},
		{"unexpected 500 → WARN", 500, nil, false, nil, "", 0, "unexpected HTTP 500"},
		{"at/just past expiry but 200 → WARN", 200, days(-1), false, nil, "", 0, "at/just past its stated expiry"},

		// The resolution branches (REFACTOR-012). Neither raises a failure, and
		// both say outright that the expiry is unmonitored — a SKIP that reads
		// as "nothing to do" is the silence this ticket exists to end.
		{
			"secret not provisioned → SKIP, stated as unmonitored", 200, days(60), false,
			fmt.Errorf("%w: github.token", secrets.ErrSecretAbsent), "", 0,
			"is not provisioned on this machine — its expiry is NOT being monitored here",
		},
		{
			"backend error → WARN, not FAIL", 200, days(60), false,
			errors.New("age decrypt github.token: no identity matched any of the recipients"), "", 0,
			"declared but did not resolve",
		},

		// The warn-threshold override: the same 20-day runway flips PASS→WARN.
		{"default 14: 20d is PASS", 200, days(20), false, nil, "", 0, "valid, expires in 20 day(s)"},
		{"override 30: 20d is WARN", 200, days(20), false, nil, "30", 0, "expires in 20 day(s) (2026-07-07) — rotate soon"},
		{"bad override: default + WARN", 200, days(60), false, nil, "abc", 0, `DOTF_PAT_EXPIRY_WARN_DAYS="abc" is not a non-negative integer`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{}
			if tc.warnDays != "" {
				env["DOTF_PAT_EXPIRY_WARN_DAYS"] = tc.warnDays
			}
			sys := newSys(env, nil, nil)
			sys.Now = func() time.Time { return fixedTestNow }
			sys.ResolveSecret = resolvesTo("tok")
			if tc.resolveErr != nil {
				sys.ResolveSecret = func(secrets.Entry) (string, error) { return "", tc.resolveErr }
			}
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

// A bw-backed GitHub PAT must be selected. It was not: selection matched the age
// blob's "github." filename prefix, and a bw entry has no filename at all, so
// BITACORA_PAT silently stopped being expiry-monitored the day it migrated
// (#961). This is the regression test for the active half of REFACTOR-012.
func TestCheckPATExpiry_SelectsBwBackedPAT(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: BITACORA_PAT, plane: app, backend: bw, bw: {item: github-bitacora-pat, field: api-token}, validate: github-token, expose: {env: BITACORA_PAT}}\n"

	sys := newSys(nil, nil, nil)
	sys.Now = func() time.Time { return fixedTestNow }
	sys.BWServeStatus = func() (string, error) { return "unlocked", nil }
	sys.ResolveSecret = resolvesTo("tok")

	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, registry), rep)

	if calls != 1 {
		t.Fatalf("a bw-backed PAT must be probed exactly once; got %d\n%s", calls, buf.String())
	}
	// The display name must identify the bw source, not print an empty age name.
	if !strings.Contains(buf.String(), "github-bitacora-pat") {
		t.Errorf("output must name the bw source\n%s", buf.String())
	}
}

// A locked (or absent) vault is answered from the daemon's own state, never by
// attempting the resolution: a shellout against a locked vault costs ~1.5s per
// secret, which is what made shell startup hang for 45 seconds (BUG-080). The
// SKIP must name a command that exists — the message this replaced named
// `secrets_refresh`, retired with the login-time loader.
func TestCheckPATExpiry_BwLockedSkipsWithoutResolving(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: BITACORA_PAT, plane: app, backend: bw, bw: {item: github-bitacora-pat, field: api-token}, validate: github-token, expose: {env: BITACORA_PAT}}\n"

	for _, state := range []string{"locked", "absent"} {
		t.Run(state, func(t *testing.T) {
			sys := newSys(nil, nil, nil)
			sys.BWServeStatus = func() (string, error) { return state, nil }

			var resolved, probed int
			sys.ResolveSecret = func(secrets.Entry) (string, error) {
				resolved++
				return "tok", nil
			}
			sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
				probed++
				return http.StatusOK, http.Header{}, nil
			}

			var buf bytes.Buffer
			rep := capture(&buf)
			checkPATExpiry(sys, patCfg(t, registry), rep)

			if resolved != 0 {
				t.Errorf("a %s vault must not be resolved against; got %d attempt(s)", state, resolved)
			}
			if probed != 0 {
				t.Errorf("no probe without a token; got %d", probed)
			}
			if rep.Failures() != 0 {
				t.Errorf("a locked vault is not a setup failure; got %d\n%s", rep.Failures(), buf.String())
			}
			if !strings.Contains(buf.String(), "dotf secrets unlock") {
				t.Errorf("the SKIP must name a command that exists\n%s", buf.String())
			}
		})
	}
}

// A daemon that answers neither "unlocked" nor a clean lock state is a third
// case: the check cannot know whether the vault is usable, so it reports the
// uncertainty as a WARN and probes nothing. Covered because "locked" and
// "absent" being tested is not the same as the error branch being tested — the
// gap AC4 claims to close.
func TestCheckPATExpiry_BwStatusErrorWarnsWithoutProbing(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: BITACORA_PAT, plane: app, backend: bw, bw: {item: github-bitacora-pat, field: api-token}, validate: github-token, expose: {env: BITACORA_PAT}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWServeStatus = func() (string, error) { return "", errors.New("unparseable envelope") }

	var resolved, probed int
	sys.ResolveSecret = func(secrets.Entry) (string, error) { resolved++; return "tok", nil }
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		probed++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, registry), rep)

	if resolved != 0 || probed != 0 {
		t.Errorf("an indeterminate vault state must resolve and probe nothing; got %d resolve(s), %d probe(s)", resolved, probed)
	}
	if rep.Failures() != 0 {
		t.Errorf("an indeterminate vault state is not a setup failure; got %d\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "could not determine Bitwarden state") {
		t.Errorf("the WARN must name the uncertainty\n%s", buf.String())
	}
}

// The check must never read a token from the ambient environment: ADR-028
// guarantees it is empty, which is exactly why the previous implementation
// SKIPped every PAT on a correctly configured machine. Resolution is the only
// source of a token.
func TestCheckPATExpiry_IgnoresAmbientEnvironment(t *testing.T) {
	sys := newSys(map[string]string{
		"GITHUB_PERSONAL_ACCESS_TOKEN": "ambient-token-that-must-not-be-used",
	}, nil, nil)
	sys.Now = func() time.Time { return fixedTestNow }

	var seen string
	sys.HTTPGet = func(_ string, h map[string]string) (int, http.Header, error) {
		seen = h["Authorization"]
		return http.StatusOK, http.Header{}, nil
	}
	sys.ResolveSecret = resolvesTo("resolved-token")

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, patOneSecret), rep)

	if seen != "Bearer resolved-token" {
		t.Fatalf("the probed token must come from the Loader, got %q\n%s", seen, buf.String())
	}
}

// github.token is mapped by two env vars yet must be probed exactly once, and a
// secret without the marker is never probed at all. The dedupe key is the entry's
// backend-qualified SourceID: keying on File alone collapsed every bw secret into
// one, since File is "" for all of them.
func TestCheckPATExpiry_ProbesEachSourceOnce(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: github-token, plane: app, backend: age, age: github.token, validate: github-token, expose: {env: [GITHUB_PERSONAL_ACCESS_TOKEN, RELEASE_TOKEN]}}\n" + // one source, two vars → one probe
		"  - {id: dockerhub-token, plane: app, backend: age, age: dockerhub.token, expose: {env: DOCKERHUB_TOKEN}}\n" // unmarked → never probed

	sys := newSys(nil, nil, nil)
	sys.Now = func() time.Time { return fixedTestNow }
	sys.ResolveSecret = resolvesTo("tok")

	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, registry), rep)

	if calls != 1 {
		t.Fatalf("want exactly 1 probe (one source deduped, unmarked ignored); got %d\n%s", calls, buf.String())
	}
}

// Two distinct bw PATs are two secrets, not one. Before REFACTOR-012 the dedupe
// keyed on Entry.File — "" for both — so the second collapsed into the first and
// went unprobed. The direct regression test for the dedupe half of the fix.
func TestCheckPATExpiry_DistinctBwPATsProbedSeparately(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: BITACORA_PAT, plane: app, backend: bw, bw: {item: github-bitacora-pat, field: api-token}, validate: github-token, expose: {env: BITACORA_PAT}}\n" +
		"  - {id: RELEASE_TOKEN, plane: app, backend: bw, bw: {item: github-release-pat, field: api-token}, validate: github-token, expose: {env: RELEASE_TOKEN}}\n"

	sys := newSys(nil, nil, nil)
	sys.Now = func() time.Time { return fixedTestNow }
	sys.BWServeStatus = func() (string, error) { return "unlocked", nil }
	sys.ResolveSecret = resolvesTo("tok")

	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, registry), rep)

	if calls != 2 {
		t.Fatalf("two distinct bw items are two secrets; want 2 probes, got %d\n%s", calls, buf.String())
	}
}

// TestCheckPATExpiry_QuickSkipsProbe: the SessionStart hot path (--quick) must
// stay fork-free and offline, so a full Run with Quick:true makes zero HTTP
// probes even with everything resolvable.
func TestCheckPATExpiry_QuickSkipsProbe(t *testing.T) {
	dotfiles := t.TempDir()
	writeFile(t, filepath.Join(dotfiles, "secrets", "registry.yaml"), patOneSecret)

	sys := newSys(map[string]string{
		"DOTFILES_DIR": dotfiles,
		"HOME":         dotfiles,
	}, nil, nil)
	sys.ResolveSecret = resolvesTo("tok")

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

// TestCheckPATExpiry_NoSecrets covers the empty-selection branch: no secret
// carries the marker ⇒ a single SKIP and no probe.
func TestCheckPATExpiry_NoSecrets(t *testing.T) {
	sys := newSys(nil, nil, nil)
	var calls int
	sys.HTTPGet = func(string, map[string]string) (int, http.Header, error) {
		calls++
		return http.StatusOK, http.Header{}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkPATExpiry(sys, patCfg(t, "version: 1\nsecrets:\n  - {id: dockerhub-token, plane: app, backend: age, age: dockerhub.token, expose: {env: DOCKERHUB_TOKEN}}\n"), rep)

	if calls != 0 {
		t.Fatalf("no marked secrets must mean no probe; got %d\n%s", calls, buf.String())
	}
	if !strings.Contains(buf.String(), "validate: github-token") {
		t.Fatalf("expected the no-secrets SKIP to name the marker it looked for\n%s", buf.String())
	}
}
