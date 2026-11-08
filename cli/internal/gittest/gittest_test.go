package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// cleanEnv clears every GIT_CONFIG_* variable Quiet reads or writes, restoring
// them after the test: a harness may pin its own pairs there, and a test that
// asserts what Quiet wrote needs a known starting point.
func cleanEnv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(name, "GIT_CONFIG_") {
			continue
		}
		t.Setenv(name, "") // registers the restore of the caller's value
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

// quietVars names every variable Quiet writes for n pairs.
func quietVars(n int) []string {
	vars := []string{countVar}
	for i := 0; i < n; i++ {
		vars = append(vars, keyVar+strconv.Itoa(i), valueVar+strconv.Itoa(i))
	}
	return vars
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

// TestQuietKeepsTheCallersConfig — a harness pins config through the same
// variables (Copilot CLI exports safe.bareRepository, credential.interactive
// and core.fsmonitor). Overwriting them would drop keys the harness relies on,
// and refusing would abort the whole test binary, which is how #3253 surfaced.
func TestQuietKeepsTheCallersConfig(t *testing.T) {
	cleanEnv(t)
	t.Setenv(countVar, "1")
	t.Setenv(keyVar+"0", "gittestdemo.caller")
	t.Setenv(valueVar+"0", "kept")
	Quiet()
	if got := gitConfig(t, t.TempDir(), "gittestdemo.caller", nil); got != "kept" {
		t.Errorf("the caller's pair was dropped: gittestdemo.caller = %q", got)
	}
	if got := gitConfig(t, t.TempDir(), "gc.auto", nil); got != "0" {
		t.Errorf("gc.auto = %q, want 0: Quiet did not append after the caller's pair", got)
	}
	if got, want := os.Getenv(countVar), strconv.Itoa(1+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q (1 caller pair + %d appended)", countVar, got, want, len(quietPairs))
	}
}

// TestQuietAppendsAfterAForeignEnvironment — the shape that aborted the test
// binary: three pairs the harness pinned, none of them ours.
func TestQuietAppendsAfterAForeignEnvironment(t *testing.T) {
	cleanEnv(t)
	foreign := [][2]string{
		{"safe.bareRepository", "explicit"},
		{"credential.interactive", "never"},
		{"core.fsmonitor", "false"},
	}
	t.Setenv(countVar, strconv.Itoa(len(foreign)))
	for i, pair := range foreign {
		t.Setenv(keyVar+strconv.Itoa(i), pair[0])
		t.Setenv(valueVar+strconv.Itoa(i), pair[1])
	}
	Quiet()
	if got, want := os.Getenv(countVar), strconv.Itoa(len(foreign)+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q", countVar, got, want)
	}
	for _, pair := range foreign {
		if got := gitConfig(t, t.TempDir(), pair[0], nil); got != pair[1] {
			t.Errorf("%s = %q, want %q: the harness pair was dropped", pair[0], got, pair[1])
		}
	}
	for _, pair := range quietPairs {
		if got := gitConfig(t, t.TempDir(), pair[0], nil); got != pair[1] {
			t.Errorf("%s = %q, want %q: Quiet's pair does not outrank the caller's", pair[0], got, pair[1])
		}
	}
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
// helper process inherits them, and its TestMain calls Quiet again. A second
// call must add nothing rather than append duplicates on every generation.
func TestQuietAcceptsItsOwnKeys(t *testing.T) {
	cleanEnv(t)
	Quiet()
	Quiet()
	if got, want := os.Getenv(countVar), strconv.Itoa(len(quietPairs)); got != want {
		t.Errorf("%s = %q after a second Quiet, want %q", countVar, got, want)
	}
	if got := gitConfig(t, t.TempDir(), "maintenance.auto", nil); got != "false" {
		t.Errorf("maintenance.auto = %q, want false", got)
	}
}

// TestQuietDoesNotIncludeItsOwnFile — a re-executed binary whose COUNT keys
// were dropped still inherits GIT_CONFIG_SYSTEM; including that file in a new
// one would nest it on every generation.
func TestQuietDoesNotIncludeItsOwnFile(t *testing.T) {
	cleanEnv(t)
	Quiet()
	first := os.Getenv(systemVar)
	for _, k := range quietVars(len(quietPairs)) {
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	Quiet()
	if got := os.Getenv(systemVar); got != first {
		t.Errorf("a second Quiet wrote a new system config %q over its own %q", got, first)
	}
}
