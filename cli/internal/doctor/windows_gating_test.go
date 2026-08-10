package doctor

// Regression coverage for BUG-052 (#804): dotf doctor emitted checks that can
// never pass on Windows (terraform FAIL, bats false-negative, compile-harness
// SKIP with a misleading reason), pushing a healthy machine to exit 1. Each case
// asserts the stable status tag (PASS/SKIP/FAIL) the branch must produce, with a
// supplemental substring only to identify the reported tool or platform reason.

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// coreToolsNoTerraform is the full core set once terraform is reclassified as
// optional — every genuinely-core tool, terraform absent.
var coreToolsNoTerraform = []string{
	"git", "zsh", "bash", "curl", "wget", "jq", "eza",
	"direnv", "node", "npm", "zoxide", "docker", "kubectl",
}

// terraform must not be a core tool (its absence is not a FAIL) and must be
// reported among the optional tools (its absence is a SKIP). One case per branch.
func TestTerraformIsOptionalNotCore(t *testing.T) {
	tests := []struct {
		name       string
		run        func(*Report)
		status     Status
		want       int
		mustReport string // supplemental: identifies the tool in output
	}{
		{
			name:   "all core present, terraform absent → no core FAIL",
			run:    func(r *Report) { checkCoreTools(newSys(nil, coreToolsNoTerraform, nil), nil, r) },
			status: StatusFail,
			want:   0,
		},
		{
			name:       "terraform absent → optional SKIP, never FAIL",
			run:        func(r *Report) { checkOptionalTools(newSys(nil, nil, nil), &Config{}, nil, r) },
			status:     StatusFail,
			want:       0,
			mustReport: "terraform",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			rep := capture(&buf)
			tt.run(rep)
			if got := rep.totals[tt.status]; got != tt.want {
				t.Errorf("%v count = %d, want %d\n%s", statusTag[tt.status], got, tt.want, buf.String())
			}
			if tt.mustReport != "" && !strings.Contains(buf.String(), tt.mustReport) {
				t.Errorf("output should report %q; got:\n%s", tt.mustReport, buf.String())
			}
		})
	}
}

// On Windows exec.LookPath only resolves names carrying a PATHEXT extension, so
// an extensionless POSIX script on PATH (~/.local/bin/bats, a bash script) was
// reported missing. has() must emulate `command -v` there — but the filesystem
// fallback is Windows-only, so a POSIX host still trusts LookPath alone.
func TestSystemHas_ExtensionlessOnPath(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		lookPathHit bool
		fileOnPath  bool
		want        bool
	}{
		{"windows: LookPath miss, script on PATH → found", "windows", false, true, true},
		{"windows: LookPath miss, nothing on PATH → not found", "windows", false, false, false},
		{"windows: LookPath hit → found regardless", "windows", true, false, true},
		{"posix: LookPath miss, script on PATH → NOT found (no fs fallback)", "linux", false, true, false},
		{"posix: LookPath hit → found", "linux", true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.fileOnPath {
				writeExec(t, filepath.Join(dir, "bats"))
			}
			sys := &System{
				GOOS: tt.goos,
				Getenv: func(k string) string {
					if k == "PATH" {
						return dir
					}
					return ""
				},
				LookPath: func(string) (string, error) {
					if tt.lookPathHit {
						return filepath.Join(dir, "bats.exe"), nil
					}
					return "", errors.New("not found")
				},
			}
			if got := sys.has("bats"); got != tt.want {
				t.Errorf("has(bats) = %v, want %v", got, tt.want)
			}
		})
	}
}

// The compile-harness drift gate is Linux-only. On Windows it SKIPs with the
// platform reason; on Linux with the script absent it SKIPs with the not-found
// reason; on Linux with a passing --check it PASSes. One case per branch.
func TestCheckCompileHarnessDrift(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		withScript bool
		checkOK    bool
		status     Status
		mustReport string
	}{
		{"windows → SKIP, Linux-only reason", "windows", false, false, StatusSkip, "Linux-only"},
		{"linux, no script → SKIP, not found", "linux", false, false, StatusSkip, "not found"},
		{"linux, script, --check clean → PASS", "linux", true, true, StatusPass, "no drift"},
		{"linux, script, --check drift → FAIL", "linux", true, false, StatusFail, "drift"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dotfiles := t.TempDir()
			compile := filepath.Join(dotfiles, "scripts", "compile-harness.sh")
			var cmdOut map[string]string
			if tt.withScript {
				writeExec(t, compile)
				if tt.checkOK {
					cmdOut = map[string]string{"bash " + compile + " --check": "ok"}
				}
			}
			sys := newSys(nil, nil, cmdOut)
			sys.GOOS = tt.goos
			var buf bytes.Buffer
			rep := capture(&buf)
			checkCompileHarnessDrift(sys, &Config{DotfilesDir: dotfiles}, rep)
			if got := rep.totals[tt.status]; got != 1 {
				t.Errorf("expected one %v; totals=%v\n%s", statusTag[tt.status], rep.totals, buf.String())
			}
			if !strings.Contains(buf.String(), tt.mustReport) {
				t.Errorf("output should state %q; got:\n%s", tt.mustReport, buf.String())
			}
		})
	}
}
