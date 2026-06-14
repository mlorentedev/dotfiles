package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newSys builds a System with deterministic env, PATH membership and command
// output — the three process-global surfaces a check touches. The filesystem is
// left real; tests root it at a temp $HOME via the env map.
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
		},
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
