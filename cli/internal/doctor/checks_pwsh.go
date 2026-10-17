package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/pwsh"
)

// checkPowerShell reports the local half of CI's PowerShell gates (PLAT-003):
// pwsh on PATH and each module versions.conf marks "# pwsh-module: <Name>"
// importable at its pin. pwsh missing fails only where packages.json names a
// package for it on this OS, since converge would have installed it; on linux
// nothing does, and CI's job stays the gate. A module off its pin warns rather
// than fails: CI's Windows doctor gate runs after setup-windows.ps1, which
// converges no module, and those jobs install their own pinned copy.
func checkPowerShell(sys *System, cfg *Config, rep *Report) {
	rep.Section("PowerShell (local lint-powershell)")
	if !sys.has(pwsh.Command) {
		if manager, pkg := pwshPackage(sys, cfg); pkg != "" {
			rep.Fail(fmt.Sprintf("pwsh not on PATH; packages.json installs %s through %s here: run `dotf converge --only tools`", pkg, manager))
		} else {
			rep.Skip("pwsh not on PATH, and packages.json installs it through nothing on " + sys.GOOS + ": CI's lint-powershell job is the gate here")
		}
		return
	}
	line, _ := sys.versionLine(pwsh.Command)
	rep.Pass("pwsh: " + line)
	if cfg.VersionsPath == "" {
		rep.Skip("no versions.conf found, so no module pin to check")
		return
	}
	raw, err := os.ReadFile(cfg.VersionsPath)
	if err != nil {
		rep.Fail("versions.conf unreadable: " + err.Error())
		return
	}
	mods, err := pwsh.ParseModules(raw)
	if err != nil {
		rep.Fail(err.Error())
		return
	}
	missing, err := pwsh.Missing(func(name string, args ...string) ([]byte, error) {
		out, err := sys.CommandOutput(name, args...)
		return []byte(out), err
	}, mods)
	switch {
	case err != nil:
		rep.Warn(err.Error())
	case len(missing) > 0:
		rep.Warn(fmt.Sprintf("not importable at their pin: %s; `dotf lint powershell` cannot match CI until `dotf converge --only pwsh-modules` runs", pwsh.Names(missing)))
	default:
		rep.Pass(fmt.Sprintf("%d module(s) at their pin: %s", len(mods), pwsh.Names(mods)))
	}
}

// pwshPackage is the manager and package the catalog entry providing pwsh
// names for this OS; the package is "" when it names none.
func pwshPackage(sys *System, cfg *Config) (manager, pkg string) {
	for _, t := range loadCatalog(sys, cfg).Tools {
		if t.Source.Type == "system" && strings.EqualFold(t.Source.Command, pwsh.Command) && t.SupportsOS(sys.GOOS) {
			return t.Source.SystemPackage(sys.GOOS)
		}
	}
	return "", ""
}
