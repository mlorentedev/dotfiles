// Package pwsh is the PowerShell layer of the local cross-OS gates (PLAT-003):
// the Gallery modules versions.conf pins for them, checked and installed
// through pwsh, and the lint that runs CI's lint-powershell job on this
// machine. The scripts it builds are fixed text plus values the parser has
// already validated, so nothing read from a file reaches pwsh as code.
package pwsh

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ModuleMarker marks the versions.conf pin on the next line as a PowerShell
// Gallery module that `dotf converge` installs at that exact version. It
// carries the Gallery name, whose casing the variable cannot: "# pwsh-module:
// PSScriptAnalyzer" over PSSCRIPTANALYZER_VERSION=1.25.0.
const ModuleMarker = "# pwsh-module:"

// Command is the PowerShell 7 executable on every OS.
const Command = "pwsh"

// What CI's lint-powershell job analyzes, and how (TestLintMatchesCIsLintPowershellJob).
const (
	PinKey       = "PSSCRIPTANALYZER_VERSION" // the analyzer's pin in versions.conf
	Analyzer     = "PSScriptAnalyzer"
	SettingsFile = ".PSScriptAnalyzerSettings.psd1"
	Severity     = "Error,Warning"
	LintFilesEnv = "DOTF_PWSH_LINT_FILES" // newline-separated paths, relative to the checkout
)

// Extensions are the file types the lint analyzes, matched case-insensitively
// as PowerShell's -Include matches them.
var Extensions = []string{".ps1", ".psm1"}

// Module is one pinned Gallery module.
type Module struct {
	Name    string
	Version string
}

var (
	nearMissRE = regexp.MustCompile(`(?i)^\s*#\s*pwsh-?\s*module\s*:`)
	nameRE     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]*$`)
	versionRE  = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}$`)
	pinKeyRE   = regexp.MustCompile(`^[A-Z][A-Z0-9_]*_VERSION$`)
)

// ParseModules returns the modules versions.conf marks, in file order. A
// marker must be followed by a NAME_VERSION=<version> line, and a comment that
// reads like a marker but is not exactly one is an error, so a pin can never
// be silently left uninstalled. Names and versions are validated here because
// they are later quoted into PowerShell.
func ParseModules(versionsConf []byte) ([]Module, error) {
	lines := strings.Split(strings.ReplaceAll(string(versionsConf), "\r\n", "\n"), "\n")
	var mods []Module
	for i, l := range lines {
		l = strings.TrimSpace(l)
		name, ok := strings.CutPrefix(l, ModuleMarker)
		if !ok {
			if nearMissRE.MatchString(l) {
				return nil, fmt.Errorf("versions.conf line %d: %q looks like %q but is not it", i+1, l, ModuleMarker)
			}
			continue
		}
		if name = strings.TrimSpace(name); !nameRE.MatchString(name) {
			return nil, fmt.Errorf("versions.conf line %d: %q is not a module name", i+1, name)
		}
		var key, version string
		if i+1 < len(lines) {
			key, version, ok = strings.Cut(strings.TrimSpace(lines[i+1]), "=")
		}
		if !ok || !pinKeyRE.MatchString(key) {
			return nil, fmt.Errorf("versions.conf line %d: %q must be followed by a NAME_VERSION=<version> line", i+1, l)
		}
		if version = strings.TrimSpace(version); !versionRE.MatchString(version) {
			return nil, fmt.Errorf("versions.conf line %d: %s=%q is not a version", i+2, key, version)
		}
		mods = append(mods, Module{Name: name, Version: version})
	}
	return mods, nil
}

// Runner runs a command and returns its stdout then stderr (tools.ExecRunner).
type Runner func(name string, args ...string) ([]byte, error)

func script(run Runner, body string) ([]byte, error) {
	return run(Command, "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference = 'Stop'\n"+body)
}

func importLine(m Module) string {
	return fmt.Sprintf("Import-Module -Name '%s' -RequiredVersion '%s' -ErrorAction Stop", m.Name, m.Version)
}

// Missing returns the modules pwsh cannot import at their exact pin, from one
// pwsh process. Importing is the check because it is the consequence the
// gates need: a module listed at the version but broken still reads missing.
// A pwsh that fails to run is an error, never an empty answer.
func Missing(run Runner, mods []Module) ([]Module, error) {
	if len(mods) == 0 {
		return nil, nil
	}
	var b strings.Builder
	for _, m := range mods {
		fmt.Fprintf(&b, "try { %s } catch { 'MISSING %s' }\n", importLine(m), m.Name)
	}
	out, err := script(run, b.String())
	if err != nil {
		return nil, fmt.Errorf("pwsh could not check the modules: %w: %s", err, bytes.TrimSpace(out))
	}
	missing := map[string]bool{}
	for l := range strings.SplitSeq(string(out), "\n") {
		if name, ok := strings.CutPrefix(strings.TrimSpace(l), "MISSING "); ok {
			missing[name] = true
		}
	}
	var left []Module
	for _, m := range mods {
		if missing[m.Name] {
			left = append(left, m)
		}
	}
	return left, nil
}

// Install installs each module at its pin into the CurrentUser scope with the
// flags CI's jobs use (-SkipPublisherCheck lets Pester replace the copy
// Windows ships signed by another publisher).
func Install(run Runner, mods []Module) error {
	var b strings.Builder
	for _, m := range mods {
		fmt.Fprintf(&b, "Install-Module -Name '%s' -RequiredVersion '%s' -Scope CurrentUser -Force -SkipPublisherCheck -ErrorAction Stop\n", m.Name, m.Version)
	}
	if out, err := script(run, b.String()); err != nil {
		return fmt.Errorf("installing %s from the PowerShell Gallery: %w: %s", Names(mods), err, bytes.TrimSpace(out))
	}
	return nil
}

// Names lists modules as name@version.
func Names(mods []Module) string {
	n := make([]string, len(mods))
	for i, m := range mods {
		n[i] = m.Name + "@" + m.Version
	}
	return strings.Join(n, ", ")
}

// Lintable keeps the paths the lint analyzes.
func Lintable(paths []string) []string {
	var out []string
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		for _, e := range Extensions {
			if ext == e {
				out = append(out, p)
			}
		}
	}
	return out
}

// LintScript is CI's "Run PSScriptAnalyzer" step over the files named in
// LintFilesEnv, with the analyzer imported at version. It exits 1 on any
// finding and 2 when the analyzer is not installed at the pin.
func LintScript(version string) string {
	return fmt.Sprintf(`try { %s } catch {
  [Console]::Error.WriteLine("%s %s is not installed: run 'dotf converge --only pwsh-modules'")
  exit 2
}
$failed = $false
$files = @($env:%s -split "`+"`n"+`" | Where-Object { $_ })
foreach ($f in $files) {
  Write-Host "Analyzing $f..." -ForegroundColor Cyan
  $results = Invoke-ScriptAnalyzer -Path $f -Settings '%s' -Severity %s
  if ($results) { $results | Format-Table -AutoSize; $failed = $true }
}
Write-Host "Analyzed $($files.Count) PowerShell files." -ForegroundColor Cyan
if ($failed) { exit 1 }
`, importLine(Module{Name: Analyzer, Version: version}), Analyzer, version, LintFilesEnv, SettingsFile, Severity)
}
