package doctor

import (
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// checkSystemPackages reports the packages.json `system` entries that wait on a
// sudo password (#2308). dotf never asks for one (`sudo -n`), so a converge or
// `dotf tools install` skips them and prints the command; when that run was a
// scheduled one, the command is in a log nobody reads, and the package stays
// missing until something needs it (GNU parallel, on msi, until `bats --jobs`
// failed). The report carries the same single command the install prints.
//
// It plans only: the same Plan `dotf tools install --dry-run` runs, which asks
// the package manager and `sudo -n true` and never installs. Only apt
// escalates, so the check plans apt entries alone: elsewhere it could only
// spend a `brew list` or a multi-second `winget list` per entry on an answer
// that is never a sudo wait.
func checkSystemPackages(sys *System, cfg *Config, rep *Report) {
	rep.Section("System packages (packages.json)")
	if manager, _ := (tools.Source{}).SystemPackage(sys.GOOS); manager != "apt" {
		rep.Skip("only apt asks for sudo, and " + sys.GOOS + " installs through another manager")
		return
	}
	if !sys.has("apt-get") {
		// Every entry would plan as waiting on a manager, and "none waits on
		// sudo" would read as a verified answer on a box nothing could ask.
		rep.Skip("apt-get is not on PATH, so no system package could be planned")
		return
	}
	in := &tools.Installer{
		GOOS:       sys.GOOS,
		HasCommand: sys.has,
		Query: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandOutput(name, args...)
			return []byte(out), err
		},
	}
	var names, pkgs []string
	for _, t := range loadCatalog(sys, cfg).Tools {
		if t.Source.Type != "system" {
			continue
		}
		if p := in.Plan(t); p.Action == tools.PlanNeedsSudo {
			names, pkgs = append(names, t.Name), append(pkgs, p.Package)
		}
	}
	if len(names) == 0 {
		rep.Pass("no system package waits on sudo")
		return
	}
	rep.Warn(fmt.Sprintf("%d system package(s) wait on sudo, which dotf never asks for (%s) — run once: `%s`",
		len(names), strings.Join(names, ", "), tools.SudoInstallCommand(pkgs)))
}
