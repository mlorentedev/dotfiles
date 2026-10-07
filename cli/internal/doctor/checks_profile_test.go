package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckProfileFiles drives the healthcheck.ps1 §4 residual port:
// AGY.md existence (cross-OS) and the Windows-only PowerShell profile, whose
// $PROFILE resolves under Documents (pwsh 7 / WinPS 5.1) including the
// OneDrive-redirected root. One row per branch.
func TestCheckProfileFiles(t *testing.T) {
	cases := []struct {
		name         string
		goos         string
		files        []string // paths (slash-separated, relative to home) to create
		wantFailures int
		commands     []string
		wantSubstr   string
	}{
		{
			name:         "posix: both configs present → pass, profile skipped",
			goos:         "linux",
			files:        []string{".claude/CLAUDE.md", ".gemini/AGY.md"},
			wantFailures: 0,
			wantSubstr:   "Windows-only",
		},
		{
			// CLAUDE.md belongs to checkInstructionDrift, which fails it only
			// when claude is installed (#2016); this check no longer reports it.
			name:         "posix: CLAUDE.md missing → not this check's to fail",
			goos:         "linux",
			files:        []string{".gemini/AGY.md"},
			wantFailures: 0,
			wantSubstr:   "AGY.md) exists",
		},
		{
			name:         "posix: AGY.md missing, agy installed → fail naming it and its remedy",
			goos:         "linux",
			files:        []string{".claude/CLAUDE.md"},
			commands:     []string{"agy"},
			wantFailures: 1,
			wantSubstr:   "(agy is installed; re-run setup-linux.sh)",
		},
		{
			name:         "posix: AGY.md missing, agy not installed → skip, not fail (#843)",
			goos:         "linux",
			files:        []string{".claude/CLAUDE.md"},
			wantFailures: 0,
			wantSubstr:   "agy is not installed",
		},
		{
			name:         "windows: pwsh 7 profile location → pass",
			goos:         "windows",
			files:        []string{".claude/CLAUDE.md", ".gemini/AGY.md", "Documents/PowerShell/Microsoft.PowerShell_profile.ps1"},
			wantFailures: 0,
			wantSubstr:   "PowerShell profile exists",
		},
		{
			name:         "windows: WinPS 5.1 profile location → pass",
			goos:         "windows",
			files:        []string{".claude/CLAUDE.md", ".gemini/AGY.md", "Documents/WindowsPowerShell/Microsoft.PowerShell_profile.ps1"},
			wantFailures: 0,
			wantSubstr:   "PowerShell profile exists",
		},
		{
			name:         "windows: OneDrive-redirected Documents → pass",
			goos:         "windows",
			files:        []string{".claude/CLAUDE.md", ".gemini/AGY.md", "OneDrive/Documents/PowerShell/Microsoft.PowerShell_profile.ps1"},
			wantFailures: 0,
			wantSubstr:   "PowerShell profile exists",
		},
		{
			name:         "windows: profile missing → fail",
			goos:         "windows",
			files:        []string{".claude/CLAUDE.md", ".gemini/AGY.md"},
			wantFailures: 1,
			wantSubstr:   "PowerShell profile missing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			for _, rel := range tc.files {
				writeFile(t, filepath.Join(home, filepath.FromSlash(rel)), "x")
			}
			sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home}, tc.commands, nil)
			sys.GOOS = tc.goos

			var buf bytes.Buffer
			rep := capture(&buf)
			checkProfileFiles(sys, nil, rep, false)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

func TestSetupRemedy_NamesTheSetupScriptForTheOS(t *testing.T) {
	if got := setupRemedy("darwin"); got != "re-run setup-linux.sh" {
		t.Errorf("darwin: %q", got)
	}
	if got := setupRemedy("windows"); got != "re-run setup-windows.ps1" {
		t.Errorf("windows: %q", got)
	}
}
