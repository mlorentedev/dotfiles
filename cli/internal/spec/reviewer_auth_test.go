package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoginAuthReviewerSkipsTheSecretsWrapper is #1156, and the defect was not a
// missing credential — it was a pool entry declaring the WRONG auth mechanism.
//
// `agy` authenticates by login: its OAuth credentials live in
// ~/.gemini/antigravity-cli/google_credentials.json, and the binary answers with
// no API key present at all (measured 2026-08-22). The pool nevertheless declared
// `secret_id: GEMINI_API_KEY`, so the launcher appended `--only GEMINI_API_KEY`
// and `dotf secrets run` refused with `unknown id or env var "GEMINI_API_KEY"`.
//
// So the provider-diverse fallback arm never worked, and nothing noticed because
// the primary always answered — which is the whole reason a fallback exists.
//
// Dropping the secret_id alone would NOT be the fix. An empty SecretID falls
// through to an unscoped `dotf secrets run --`, injecting every credential into a
// process that needs none, and one broken unrelated registry entry would then
// block the review — exactly what BUG-089 introduced scoping to prevent.
func TestLoginAuthReviewerSkipsTheSecretsWrapper(t *testing.T) {
	e := ReviewerEntry{
		ID:     "agy/gemini-3.1-pro-high",
		Runner: "agy",
		Model:  "gemini-3.1-pro-high",
		Auth:   "login",
	}
	argv, err := ReviewerCommand(e, "review this", time.Minute, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "secrets") {
		t.Errorf("a login-authenticated runner must not be wrapped in `dotf secrets run`: "+
			"it needs no credential injected, and an unrelated broken registry entry would block it.\ngot: %s",
			joined)
	}
	if len(argv) == 0 || argv[0] != "agy" {
		t.Errorf("expected the runner to be invoked directly, got %v", argv)
	}
	// The model pin is an independent property and must survive the change.
	if i := argvIndex(argv, "--model"); i < 0 || argv[i+1] != "gemini-3.1-pro-high" {
		t.Errorf("model must stay explicitly pinned: %s", joined)
	}
}

// TestSecretAuthReviewerKeepsScopedInjection pins the other direction, so the
// fix above cannot quietly widen every reviewer's credential exposure.
func TestSecretAuthReviewerKeepsScopedInjection(t *testing.T) {
	e := ReviewerEntry{
		ID:       "nan/deepseek-v4-flash",
		Runner:   "pi",
		Provider: "nan",
		Model:    "deepseek-v4-flash",
		SecretID: "NAN_API_KEY",
	}
	argv, err := ReviewerCommand(e, "review this", time.Minute, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "dotf secrets run --only NAN_API_KEY") {
		t.Errorf("a secret-authenticated runner must keep SCOPED injection (BUG-089): %s", joined)
	}
}

// TestEveryPoolEntryDeclaresItsAuth guards the CLASS rather than the instance.
//
// The failure was an absence read as a default: the entry named a credential
// that does not exist, and nothing in the file said whether "no working secret"
// meant "authenticates by login" or "someone forgot". Every entry must now say
// which, so a half-declared member cannot be added and then discovered only when
// a review refuses to start.
func TestEveryPoolEntryDeclaresItsAuth(t *testing.T) {
	entries, err := LoadReviewerPoolEntries(shippedRepoRoot(t))
	if err != nil {
		t.Fatalf("load pool: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("the shipped pool is empty, so this guard checked nothing")
	}
	for _, e := range entries {
		hasSecret := strings.TrimSpace(e.SecretID) != ""
		isLogin := strings.TrimSpace(e.Auth) == "login"
		if hasSecret == isLogin {
			t.Errorf("pool entry %q declares secret_id=%q auth=%q — it must declare exactly one: "+
				"a `secret_id` to inject, or `auth: login` to say it needs none. Declaring neither "+
				"is how the fallback arm came to be decorative (#1156)", e.ID, e.SecretID, e.Auth)
		}
	}
}

// TestShippedPoolHasAnIndependentProviderArm pins the property the pool exists
// for. ADR-035 leans on provider diversity for the adversarial review's
// independence claim, and until #1156 that claim rested on an arm that could not
// start. A pool where every member shares one provider is a single point of
// failure wearing a fallback's name.
func TestShippedPoolHasAnIndependentProviderArm(t *testing.T) {
	entries, err := LoadReviewerPoolEntries(shippedRepoRoot(t))
	if err != nil {
		t.Fatalf("load pool: %v", err)
	}
	runners := map[string]bool{}
	for _, e := range entries {
		runners[e.Runner] = true
	}
	if len(runners) < 2 {
		t.Errorf("every pool member runs on the same runner (%v) — the fallback is a second model, "+
			"not a second provider family, and shares its failure modes", runners)
	}
}

// shippedRepoRoot walks up to the repository root so the guards above read the
// pool this repo actually ships, rather than a fixture that can drift from it.
func shippedRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "harness", "reviewer-pool.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find harness/reviewer-pool.json walking up from %s", dir)
		}
		dir = parent
	}
}
