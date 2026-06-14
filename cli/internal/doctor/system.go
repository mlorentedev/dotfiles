// Package doctor implements the `dotf doctor` post-setup diagnostics domain.
// It consolidates the two retired shell twins — scripts/healthcheck.sh (the
// 12-section sweep) and scripts/doctor.sh (the env-contract verifier with a
// --fix heal path) — into one cross-compiled checker (ADR-021, the first port).
//
// Design: the process-global / external surfaces (environment variables, PATH
// lookups, running `<tool> --version`) are abstracted behind System so checks
// are table-testable; the filesystem is hit directly via os/filepath, with
// tests building a real temp tree rooted at a fake $HOME.
package doctor

import (
	"os"
	"os/exec"
	"strings"
)

// System abstracts the non-deterministic, process-global surfaces a check
// touches. The real implementation wires the os/exec stdlib; tests inject
// deterministic funcs. Filesystem access is intentionally NOT here — checks use
// os/filepath directly against paths derived from Getenv, and tests build a real
// temp tree, which is both more faithful and simpler than a virtual FS.
type System struct {
	// Getenv resolves an environment variable (os.Getenv in production).
	Getenv func(string) string
	// LookPath reports whether an executable is resolvable on PATH
	// (exec.LookPath in production); err != nil means "not found".
	LookPath func(string) (string, error)
	// CommandOutput runs name with args and returns combined stdout+stderr.
	// Used for the `<tool> --version` probes; faked in tests.
	CommandOutput func(name string, args ...string) (string, error)
}

// realSystem wires System to the live OS.
func realSystem() *System {
	return &System{
		Getenv:   os.Getenv,
		LookPath: exec.LookPath,
		CommandOutput: func(name string, args ...string) (string, error) {
			out, err := exec.Command(name, args...).CombinedOutput()
			return string(out), err
		},
	}
}

// env returns the value of key, or def when unset/empty. Mirrors the shell
// ${VAR:-default} idiom the twins lean on heavily.
func (s *System) env(key, def string) string {
	if v := s.Getenv(key); v != "" {
		return v
	}
	return def
}

// has reports whether name resolves on PATH (the `command -v` test).
func (s *System) has(name string) bool {
	_, err := s.LookPath(name)
	return err == nil
}

// home resolves the user's home directory, preferring HOME (POSIX) then
// USERPROFILE (Windows), matching the env-contract's OS-scoped vars.
func (s *System) home() string {
	if h := s.Getenv("HOME"); h != "" {
		return h
	}
	return s.Getenv("USERPROFILE")
}

// pathEntries splits PATH into its entries using the OS list separator.
func (s *System) pathEntries() []string {
	raw := s.Getenv("PATH")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, string(os.PathListSeparator))
}

// versionLine runs `<name> --version`, returning the first output line. A
// non-nil error means the probe itself failed (binary refused to run); callers
// treat that as "unparseable" rather than a hard failure, mirroring the twins.
func (s *System) versionLine(name string) (string, error) {
	out, err := s.CommandOutput(name, "--version")
	first := out
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		first = out[:i]
	}
	return strings.TrimSpace(first), err
}
