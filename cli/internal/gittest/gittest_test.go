package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cleanEnv clears every variable Quiet sets, restoring them after the test.
func cleanEnv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	for _, k := range []string{countVar, "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_1", "GIT_CONFIG_VALUE_1", systemVar, "GIT_CONFIG_NOSYSTEM"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
}

// gitConfig reads key in dir through a git started with env, or with the
// process environment when env is nil.
func gitConfig(t *testing.T, dir, key string, env []string) string {
	t.Helper()
	c := exec.Command("git", "-C", dir, "config", key)
	c.Env = env
	out, err := c.Output()
	if err != nil {
		t.Fatalf("git config %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}

// TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts — the keys must reach a
// git that no fixture helper builds, so assert them through a plain exec.Command
// that only inherits the environment.
func TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts(t *testing.T) {
	cleanEnv(t)
	Quiet()
	for key, want := range map[string]string{"gc.auto": "0", "maintenance.auto": "false"} {
		if got := gitConfig(t, t.TempDir(), key, nil); got != want {
			t.Errorf("git config %s = %q, want %q", key, got, want)
		}
	}
}

// TestQuietReachesTheRemoteSideOfALocalTransport — a local push starts the
// remote's receive-pack with GIT_CONFIG_COUNT unset (#3031), so assert the keys
// through exactly that environment.
func TestQuietReachesTheRemoteSideOfALocalTransport(t *testing.T) {
	cleanEnv(t)
	Quiet()
	var remote []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, countVar+"=") {
			remote = append(remote, kv)
		}
	}
	for key, want := range map[string]string{"receive.autogc": "false", "gc.auto": "0", "maintenance.auto": "false"} {
		if got := gitConfig(t, t.TempDir(), key, remote); got != want {
			t.Errorf("git config %s on the remote side = %q, want %q", key, got, want)
		}
	}
}

// TestQuietKeepsTheSystemConfigInForce — replacing the system config would drop
// what the host sets there, such as core.autocrlf on Git for Windows.
func TestQuietKeepsTheSystemConfigInForce(t *testing.T) {
	cleanEnv(t)
	host := filepath.Join(t.TempDir(), "sys#tem;config") // comment characters git must not cut at
	if err := os.WriteFile(host, []byte("[gittestdemo]\n\tkept = yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(systemVar, host)
	Quiet()
	// Its name hashes this test's own temp path, so no other run shares it.
	written := os.Getenv(systemVar)
	t.Cleanup(func() { _ = os.Remove(written) })
	if got := gitConfig(t, t.TempDir(), "gittestdemo.kept", nil); got != "yes" {
		t.Errorf("the host's system config was dropped: gittestdemo.kept = %q", got)
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

// TestQuietRefusesNoSystem — git would skip the file that quiets a remote.
func TestQuietRefusesNoSystem(t *testing.T) {
	cleanEnv(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	defer func() {
		if recover() == nil {
			t.Fatal("Quiet accepted GIT_CONFIG_NOSYSTEM, which leaves remotes unquieted")
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

// TestQuietDoesNotIncludeItsOwnFile — a re-executed binary whose COUNT keys
// were dropped still inherits GIT_CONFIG_SYSTEM; including that file in a new
// one would nest it on every generation.
func TestQuietDoesNotIncludeItsOwnFile(t *testing.T) {
	cleanEnv(t)
	Quiet()
	first := os.Getenv(systemVar)
	for _, k := range []string{countVar, "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_1", "GIT_CONFIG_VALUE_1"} {
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	Quiet()
	if got := os.Getenv(systemVar); got != first {
		t.Errorf("a second Quiet wrote a new system config %q over its own %q", got, first)
	}
}
