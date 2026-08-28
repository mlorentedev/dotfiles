package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The dotfiles section of the PowerShell profile is delimited by these two
// markers — verbatim the strings setup-windows.ps1's splice writes and
// scripts/profile-heal.ps1 counts. The SSOT (powershell/profile.ps1) contains
// neither, so a healthy deployed profile carries exactly one of each.
const (
	profileStartMarker = "# >>> DOTFILES PROFILE >>>"
	profileEndMarker   = "# <<< DOTFILES PROFILE <<<"

	// profileMaxBytes is the size signal profile-heal.ps1 heals at (`-gt 1MB`).
	// A healthy profile is ~7 KB; BUG-020's had compounded to 26 MB / 689K lines
	// of duplicated sections before anything noticed.
	profileMaxBytes = 1 << 20

	// profileHealScript is the BUG-020 repair, deployed under SCRIPTS_DIR by
	// setup-windows.ps1: it backs the corrupted profile up beside itself and
	// rewrites the dotfiles section from the SSOT.
	profileHealScript = "profile-heal.ps1"
)

// checkProfileFiles ports the healthcheck.ps1 §4 residual the .sh→Go
// consolidation (CLI-012) left behind: existence of deployed config files that
// checkSymlinks does not already cover. CLAUDE.md and AGY.md are deployed on both
// OSes; the PowerShell profile is Windows-only ($PROFILE) — on POSIX the
// equivalent rc files (.zshrc/.bashrc) are already checked by checkSymlinks.
//
// A missing file is a FAIL, matching checkSymlinks' treatment of a missing
// deployed file (both reproduce the same healthcheck §4 Write-Fail semantics).
//
// The Windows profile is also checked for BUG-020 corruption (#531): the
// retired doctor.ps1 -Fix used to invoke profile-heal.ps1, and after the
// consolidation a corrupted profile was neither detected nor healed from
// doctor — existence was the whole test. Under --fix the heal runs through
// the System seam and is verified by consequence, never by its exit code.
func checkProfileFiles(sys *System, c *Contract, rep *Report, fix bool) {
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

	profile, checked := findPowerShellProfile(home)
	if profile == "" {
		rep.Fail("PowerShell profile missing (checked: " + strings.Join(checked, ", ") + ")")
		return
	}
	reasons := profileCorruption(profile)
	if len(reasons) == 0 {
		rep.Pass("PowerShell profile exists (" + profile + ")")
		return
	}
	heal := profileHealPath(sys, c)
	if !fix {
		rep.Fail(fmt.Sprintf("PowerShell profile corrupted (%s) — BUG-020; run `pwsh -NoProfile -File %s` or `dotf doctor --fix` (backs the profile up, then rebuilds the dotfiles section from powershell/profile.ps1)",
			strings.Join(reasons, "; "), heal))
		return
	}
	repairProfile(sys, rep, profile, heal)
}

// findPowerShellProfile returns the first existing $PROFILE
// (CurrentUserCurrentHost) location and the list it searched.
//
// $PROFILE lives under Documents as Microsoft.PowerShell_profile.ps1, in
// PowerShell\ for pwsh 7 or WindowsPowerShell\ for Windows PowerShell 5.1.
// Documents itself is often OneDrive-redirected on corporate boxes, so accept
// that root too — checking only ~\Documents would false-FAIL there. (Go has no
// $PROFILE intrinsic; this enumerates the realistic locations
// setup-windows.ps1's $PROFILE resolves to.)
func findPowerShellProfile(home string) (string, []string) {
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
				return p, checked
			}
		}
	}
	return "", checked
}

// profileCorruption returns the BUG-020 signals the profile at path trips, or
// nil for a healthy one. The two signals are profile-heal.ps1's own, so doctor
// never flags a profile the heal would leave alone: a size over 1 MiB, and more
// than one START or END marker (setup's splice replaces the FIRST pair only, so
// a second pair is the beginning of the accumulation that reached 26 MB).
//
// An oversized profile is reported on size alone — counting markers in a
// 26 MB file proves nothing size has not already proved, and reading it is
// what the heal's own preflight refuses to do in place.
func profileCorruption(path string) []string {
	fi, err := os.Stat(path)
	if err != nil {
		return []string{"unreadable: " + err.Error()}
	}
	if fi.Size() > profileMaxBytes {
		return []string{fmt.Sprintf("size %d bytes > 1 MiB", fi.Size())}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{"unreadable: " + err.Error()}
	}
	starts := strings.Count(string(raw), profileStartMarker)
	ends := strings.Count(string(raw), profileEndMarker)
	if starts > 1 || ends > 1 {
		return []string{fmt.Sprintf("duplicate dotfiles markers (start=%d, end=%d; a healthy profile has one pair)", starts, ends)}
	}
	return nil
}

// profileHealPath resolves <SCRIPTS_DIR>\profile-heal.ps1 through the env
// contract — the variable when set, else the contract's default for this OS
// dialect — never a literal path, because where scripts deploy is the
// contract's decision (ADR-025) and it is changing under WIN-013 (#1310).
// When neither resolves, the token is left visible so the message stays honest
// about what it could not name.
func profileHealPath(sys *System, c *Contract) string {
	dir := sys.Getenv("SCRIPTS_DIR")
	if dir == "" {
		dir = contractDefault(sys, c, "SCRIPTS_DIR")
	}
	if dir == "" {
		return `$env:SCRIPTS_DIR\` + profileHealScript
	}
	// Host-native join: in production this only runs on Windows (the check is
	// GOOS-gated), and the tests that exercise it on Linux CI build their
	// expectations with the same filepath.Join, so the separator never lies.
	return filepath.Join(dir, profileHealScript)
}

// contractDefault returns the contract's default for the named env var on this
// OS dialect, home-expanded, or "" when the contract is absent or declares none.
func contractDefault(sys *System, c *Contract, name string) string {
	if c == nil {
		return ""
	}
	for _, e := range c.EnvVars {
		if e.Name == name {
			return expandHome(sys, e.Default[contractOS(sys)])
		}
	}
	return ""
}

// repairProfile runs profile-heal.ps1 against the corrupted profile and reports
// the OUTCOME, not the invocation: the heal always exits 0 by design (it never
// blocks a session on a transient failure), so its exit status says nothing.
// The profile is re-measured afterwards, and only a profile that now passes
// profileCorruption is a Fix. The heal backs the corrupted file up beside itself
// before writing, so nothing is lost by running it.
func repairProfile(sys *System, rep *Report, profile, heal string) {
	if !isRegularFile(heal) {
		rep.Fail(fmt.Sprintf("PowerShell profile corrupted, and %s is not deployed at %s — run setup-windows.ps1 to deploy scripts/, then `dotf doctor --fix` again (BUG-020)",
			profileHealScript, heal))
		return
	}
	if !sys.has("pwsh") {
		rep.Fail("PowerShell profile corrupted, and pwsh is not in PATH to run " + profileHealScript + " (BUG-020)")
		return
	}
	out, err := sys.CommandOutput("pwsh", "-NoProfile", "-File", heal)
	if err != nil {
		rep.Fail(fmt.Sprintf("%s did not run (%s) — profile left as is", profileHealScript, firstLineOr(out, err)))
		return
	}
	if reasons := profileCorruption(profile); len(reasons) > 0 {
		rep.Fail(fmt.Sprintf("%s ran but the profile is still corrupted (%s) — heal said: %s",
			profileHealScript, strings.Join(reasons, "; "), firstLineOr(out, nil)))
		return
	}
	rep.Fix("PowerShell profile rebuilt from powershell/profile.ps1 by " + profileHealScript + " (the corrupted copy is backed up beside it; restart PowerShell)")
}

// firstLineOr renders a subprocess result as one line: its first line of
// output when it produced one, else the error, else "no output".
func firstLineOr(out string, err error) string {
	if l := firstLine(out); l != "" {
		return l
	}
	if err != nil {
		return err.Error()
	}
	return "no output"
}
