package doctor

import (
	"path/filepath"
	"strings"
)

// checkProfileFiles ports the healthcheck.ps1 §4 residual the .sh→Go
// consolidation (CLI-012) left behind: existence of deployed config files that
// checkSymlinks does not already cover. CLAUDE.md and AGY.md are deployed on both
// OSes; the PowerShell profile is Windows-only ($PROFILE) — on POSIX the
// equivalent rc files (.zshrc/.bashrc) are already checked by checkSymlinks.
//
// A missing file is a FAIL, matching checkSymlinks' treatment of a missing
// deployed file (both reproduce the same healthcheck §4 Write-Fail semantics).
func checkProfileFiles(sys *System, rep *Report) {
	rep.Section("Deployed config files")
	home := sys.home()

	for _, f := range []struct{ rel, name string }{
		{".claude/CLAUDE.md", "Claude global instructions (CLAUDE.md)"},
		{".gemini/AGY.md", "Antigravity instructions (AGY.md)"},
	} {
		p := filepath.Join(home, filepath.FromSlash(f.rel))
		if pathExists(p) {
			rep.Pass(f.name + " exists")
		} else {
			rep.Fail(f.name + " missing: " + p)
		}
	}

	if sys.GOOS != "windows" {
		rep.Skip("PowerShell profile (Windows-only; POSIX uses .zshrc/.bashrc, checked above)")
		return
	}

	// $PROFILE (CurrentUserCurrentHost) lives under Documents as
	// Microsoft.PowerShell_profile.ps1, in PowerShell\ for pwsh 7 or
	// WindowsPowerShell\ for Windows PowerShell 5.1. Documents itself is often
	// OneDrive-redirected on corporate boxes, so accept that root too — checking
	// only ~\Documents would false-FAIL there. (Go has no $PROFILE intrinsic; this
	// enumerates the realistic locations setup-windows.ps1's $PROFILE resolves to.)
	const leaf = "Microsoft.PowerShell_profile.ps1"
	var checked []string
	for _, docRoot := range []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "OneDrive", "Documents"),
	} {
		for _, host := range []string{"PowerShell", "WindowsPowerShell"} {
			p := filepath.Join(docRoot, host, leaf)
			checked = append(checked, p)
			if pathExists(p) {
				rep.Pass("PowerShell profile exists (" + p + ")")
				return
			}
		}
	}
	rep.Fail("PowerShell profile missing (checked: " + strings.Join(checked, ", ") + ")")
}
