package gittest

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts — the keys must reach a
// git that no fixture helper builds, so assert them through a plain exec.Command
// that only inherits the environment.
func TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	for _, k := range []string{countVar, "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_1", "GIT_CONFIG_VALUE_1"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	Quiet()
	for key, want := range map[string]string{"gc.auto": "0", "maintenance.auto": "false"} {
		out, err := exec.Command("git", "-C", t.TempDir(), "config", key).Output()
		if err != nil {
			t.Fatalf("git config %s: %v", key, err)
		}
		if got := strings.TrimSpace(string(out)); got != want {
			t.Errorf("git config %s = %q, want %q", key, got, want)
		}
	}
}

// TestQuietRefusesAnEnvironmentThatAlreadyCarriesConfig — overwriting another
// caller's GIT_CONFIG_COUNT would drop its keys silently.
func TestQuietRefusesAnEnvironmentThatAlreadyCarriesConfig(t *testing.T) {
	t.Setenv(countVar, "1")
	defer func() {
		if recover() == nil {
			t.Fatal("Quiet overwrote an existing " + countVar)
		}
	}()
	Quiet()
}

// TestQuietAcceptsItsOwnKeys — a test binary that re-executes itself as a
// helper process inherits them, and its TestMain calls Quiet again.
func TestQuietAcceptsItsOwnKeys(t *testing.T) {
	for k, v := range quiet {
		t.Setenv(k, v)
	}
	Quiet()
}
