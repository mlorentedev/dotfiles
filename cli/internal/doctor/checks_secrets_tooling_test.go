package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// runSecretsTooling drives checkSecretsTooling with a fake System (default:
// round-trip succeeds) and returns the captured (verbose) report plus its
// failure count.
func runSecretsTooling(t *testing.T, env map[string]string, onPath []string) (string, int) {
	return runSecretsToolingRT(t, env, onPath, nil)
}

// runSecretsToolingRT is runSecretsTooling with an injectable AgeRoundTrip seam,
// so the round-trip PASS / FAIL / not-called paths are table-testable with no
// real age binary or key. A nil rt keeps newSys's success default.
func runSecretsToolingRT(t *testing.T, env map[string]string, onPath []string, rt func(string) error) (string, int) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(env, onPath, nil)
	if rt != nil {
		sys.AgeRoundTrip = rt
	}
	checkSecretsTooling(sys, rep)
	rep.Summary()
	return buf.String(), rep.Failures()
}

func TestSecretsTooling_AllPresent(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "AGE-SECRET-KEY-1...\n")

	// A fully healthy box has bw, age AND age-keygen — so the round-trip runs and
	// the default success seam yields the verified-round-trip PASS.
	out, fails := runSecretsTooling(t, map[string]string{"HOME": home}, []string{"bw", "age", "age-keygen"})
	if fails != 0 {
		t.Fatalf("want 0 failures, got %d\n%s", fails, out)
	}
	for _, want := range []string{"bw", "age", "age identity key present", "age root-of-trust verified (round-trip)"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n%s", want, out)
		}
	}
}

func TestSecretsTooling_BwMissing(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "k\n")

	out, fails := runSecretsTooling(t, map[string]string{"HOME": home}, []string{"age"})
	if fails != 1 {
		t.Fatalf("want 1 failure (bw), got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "bw not in PATH") {
		t.Errorf("expected bw FAIL message\n%s", out)
	}
}

func TestSecretsTooling_AgeMissing(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "k\n")

	out, fails := runSecretsTooling(t, map[string]string{"HOME": home}, []string{"bw"})
	if fails != 1 {
		t.Fatalf("want 1 failure (age), got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "age not in PATH") {
		t.Errorf("expected age FAIL message\n%s", out)
	}
}

func TestSecretsTooling_AgeKeyMissingIsWarnNotFail(t *testing.T) {
	home := t.TempDir() // no key.txt created

	out, fails := runSecretsTooling(t, map[string]string{"HOME": home}, []string{"bw", "age"})
	if fails != 0 {
		t.Fatalf("missing age key must WARN, not FAIL; got %d failures\n%s", fails, out)
	}
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, "age identity key missing") {
		t.Errorf("expected a WARN about the missing age key\n%s", out)
	}
}

func TestSecretsTooling_AgeKeyPathOverride(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(t.TempDir(), "ci-age-key.txt")
	writeFile(t, custom, "k\n")

	out, fails := runSecretsTooling(t,
		map[string]string{"HOME": home, "AGE_KEY_PATH": custom},
		[]string{"bw", "age"})
	if fails != 0 {
		t.Fatalf("want 0 failures with AGE_KEY_PATH override, got %d\n%s", fails, out)
	}
	if !strings.Contains(out, custom) {
		t.Errorf("expected the report to honour AGE_KEY_PATH (%s)\n%s", custom, out)
	}
}

// A present-but-broken key (round-trip errors) is the failure #518 exists to
// surface: a red check now, not a silent surprise at recover time.
func TestSecretsTooling_RoundTripFailIsFail(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "corrupt\n")

	failing := func(string) error { return errors.New("decrypted bytes differ from the sentinel") }
	out, fails := runSecretsToolingRT(t, map[string]string{"HOME": home}, []string{"bw", "age", "age-keygen"}, failing)
	if fails != 1 {
		t.Fatalf("a broken key round-trip must FAIL exactly once, got %d\n%s", fails, out)
	}
	for _, want := range []string{"round-trip FAILED", "decrypted bytes differ", "restore a good key"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected FAIL message to contain %q\n%s", want, out)
		}
	}
}

// A missing key must WARN (fresh box) and must NEVER invoke the round-trip: there
// is no key to round-trip, and calling the seam would be a logic bug.
func TestSecretsTooling_KeyMissingSkipsRoundTrip(t *testing.T) {
	home := t.TempDir() // no key.txt

	called := false
	spy := func(string) error { called = true; return nil }
	out, fails := runSecretsToolingRT(t, map[string]string{"HOME": home}, []string{"bw", "age", "age-keygen"}, spy)
	if fails != 0 {
		t.Fatalf("missing key must WARN, not FAIL; got %d\n%s", fails, out)
	}
	if called {
		t.Errorf("round-trip must not run when the key is absent\n%s", out)
	}
	if !strings.Contains(out, "age identity key missing") {
		t.Errorf("expected the WARN about the missing key\n%s", out)
	}
}

// age present but age-keygen absent: the recipient cannot be derived, so we can't
// verify the round-trip. That is a WARN (the key may be fine) — never a FAIL —
// and the seam must not be called.
func TestSecretsTooling_AgeKeygenMissingWarnsAndSkips(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "k\n")

	called := false
	spy := func(string) error { called = true; return nil }
	out, fails := runSecretsToolingRT(t, map[string]string{"HOME": home}, []string{"bw", "age"}, spy)
	if fails != 0 {
		t.Fatalf("missing age-keygen must WARN, not FAIL; got %d\n%s", fails, out)
	}
	if called {
		t.Errorf("round-trip must not run without age-keygen\n%s", out)
	}
	if !strings.Contains(out, "age-keygen not in PATH") {
		t.Errorf("expected the WARN about the missing age-keygen\n%s", out)
	}
}
