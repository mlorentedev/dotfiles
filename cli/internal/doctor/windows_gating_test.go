package doctor

// Regression coverage for BUG-052 (#804): dotf doctor emitted checks that can
// never pass on Windows (terraform FAIL, bats false-negative, compile-harness
// SKIP with a misleading reason), pushing a healthy machine to exit 1.

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// terraform is optional (winget on Windows, Applications on Linux), so its
// absence must be a SKIP under optional tools, never a core-tools FAIL.
func TestCheckCoreTools_TerraformNotCore(t *testing.T) {
	// Every genuinely-core tool on PATH; terraform absent.
	onPath := []string{"git", "zsh", "bash", "curl", "wget", "jq", "eza", "direnv", "node", "npm", "zoxide", "docker", "kubectl"}
	rep := capture(&bytes.Buffer{})
	checkCoreTools(newSys(nil, onPath, nil), nil, rep)
	if rep.Failures() != 0 {
		t.Fatalf("terraform absence must not fail core-tools; got %d failures", rep.Failures())
	}
}

func TestCheckOptionalTools_TerraformSkipWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	checkOptionalTools(newSys(nil, nil, nil), &Config{}, nil, rep)
	if rep.Failures() != 0 {
		t.Fatalf("optional tools must never FAIL; got %d", rep.Failures())
	}
	if !strings.Contains(buf.String(), "terraform") {
		t.Error("terraform should be reported among optional tools")
	}
}

// On Windows exec.LookPath only resolves names carrying a PATHEXT extension, so
// an extensionless POSIX script on PATH (~/.local/bin/bats, a bash script) was
// reported missing. has() must emulate `command -v` and find it.
func TestSystemHas_WindowsExtensionlessOnPath(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, filepath.Join(dir, "bats"))
	sys := &System{
		GOOS: "windows",
		Getenv: func(k string) string {
			if k == "PATH" {
				return dir
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "", errors.New("PATHEXT miss") },
	}
	if !sys.has("bats") {
		t.Fatal("has() must resolve an extensionless script on PATH on Windows")
	}
}

// The fallback is Windows-only: on POSIX, LookPath is authoritative and a
// filesystem scan would mask a genuinely-missing tool.
func TestSystemHas_PosixNoFilesystemFallback(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, filepath.Join(dir, "bats"))
	sys := &System{
		GOOS: "linux",
		Getenv: func(k string) string {
			if k == "PATH" {
				return dir
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}
	if sys.has("bats") {
		t.Fatal("has() must not filesystem-scan on POSIX; LookPath is authoritative")
	}
}

func TestCheckHarnessDrift_WindowsSkipsLinuxEngine(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	cfg := &Config{DotfilesDir: t.TempDir()}
	sys := newSys(nil, nil, nil)
	sys.GOOS = "windows"
	checkHarnessDrift(sys, cfg, rep)
	if rep.Failures() != 0 {
		t.Fatalf("windows harness check must not FAIL; got %d", rep.Failures())
	}
	if !strings.Contains(buf.String(), "Windows") && !strings.Contains(buf.String(), "Linux-only") {
		t.Errorf("windows skip must state the platform reason; got:\n%s", buf.String())
	}
}
