package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// runSecretsTooling drives checkSecretsTooling with a fake System and returns the
// captured (verbose) report plus its failure count.
func runSecretsTooling(t *testing.T, env map[string]string, onPath []string) (string, int) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(env, onPath, nil)
	checkSecretsTooling(sys, rep)
	rep.Summary()
	return buf.String(), rep.Failures()
}

func TestSecretsTooling_AllPresent(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "age", "key.txt"), "AGE-SECRET-KEY-1...\n")

	out, fails := runSecretsTooling(t, map[string]string{"HOME": home}, []string{"bw", "age"})
	if fails != 0 {
		t.Fatalf("want 0 failures, got %d\n%s", fails, out)
	}
	for _, want := range []string{"bw", "age", "age identity key present"} {
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
