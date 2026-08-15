package doctor

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedTestNow is the deterministic clock the default test System reports, so
// any check reading sys.Now() in a full-sweep test is reproducible.
var fixedTestNow = time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

// newSys builds a System with deterministic env, PATH membership and command
// output — the three process-global surfaces a check touches. The filesystem is
// left real; tests root it at a temp $HOME via the env map.
// sysCommandOutput is the fake exec table shared by the CommandOutput and
// CommandOutputBounded seams, so both resolve identically and a test that stubs
// a command does not have to know which seam the check happens to call.
func sysCommandOutput(cmdOut map[string]string, name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if out, ok := cmdOut[key]; ok {
		return out, nil
	}
	if out, ok := cmdOut[name]; ok {
		return out, nil
	}
	return "", errors.New("no such command: " + key)
}

func newSys(env map[string]string, onPath []string, cmdOut map[string]string) *System {
	if env == nil {
		env = map[string]string{}
	}
	pathset := map[string]bool{}
	for _, p := range onPath {
		pathset[p] = true
	}
	return &System{
		Getenv: func(k string) string { return env[k] },
		LookPath: func(n string) (string, error) {
			if pathset[n] {
				return "/usr/bin/" + n, nil
			}
			return "", errors.New("not found: " + n)
		},
		CommandOutput: func(name string, args ...string) (string, error) {
			return sysCommandOutput(cmdOut, name, args...)
		},
		// Safe defaults so full-sweep tests never nil-panic on the network/clock
		// seams: the clock is fixed, and HTTPGet reports "offline" (which the
		// PAT check degrades to a WARN, not a panic). Tests exercising the PAT
		// check inject their own HTTPGet/Now.
		HTTPGet: func(string, map[string]string) (int, http.Header, error) {
			return 0, nil, errors.New("no network in tests")
		},
		Now: func() time.Time { return fixedTestNow },
		// Default: the age key round-trips cleanly (a healthy box). Tests
		// exercising the FAIL / not-called paths inject their own AgeRoundTrip.
		AgeRoundTrip: func(string) error { return nil },
		// Default: no registry entry resolves through Bitwarden — the state of
		// the repo before the #585 migration, and the one that keeps an
		// unreachable vault advisory. Tests exercising the exposed severity
		// inject their own count.
		BWBackedSecrets: func() (int, error) { return 0, nil },
		// Bounded exec resolves to the same fake table as CommandOutput; the
		// deadline and the stream split are production-only behaviour, each
		// exercised by its own test against realSystem().
		CommandOutputBounded: func(_ time.Duration, name string, args ...string) (string, string, error) {
			out, err := sysCommandOutput(cmdOut, name, args...)
			return out, "", err
		},
		// Default: no daemon running — the common case (nothing has run `dotf
		// secrets unlock`). Tests exercising the daemon states inject their own.
		BWServeStatus: func() (string, error) { return "absent", nil },
	}
}

// capture returns a verbose Report writing into buf (so passing checks are
// visible to substring assertions).
func capture(buf *bytes.Buffer) *Report { return NewReport(buf, true) }

// writeExec writes an "executable" file (0o755) at path, creating parents.
func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// mkdirAll creates a directory tree.
// writeExecFile writes content at path with the executable bit set. Distinct
// from writeExec (fixed stub body) and writeFile (no exec bit): a hook needs
// both its real content AND the bit, because git silently ignores a hook file it
// cannot execute — a fixture with the bit off models a gate that does not fire.
func writeExecFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeFile writes a regular file, creating parents.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustSymlink creates a symlink, skipping the test when the OS forbids it
// (unprivileged Windows runners) rather than failing it.
func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(newname), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink unsupported in this environment: %v", err)
	}
}
