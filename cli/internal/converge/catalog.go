package converge

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// CatalogInstaller is the slice of tools.Installer the tools step drives: the
// same Plan and Install `dotf tools install [--dry-run]` runs, so the step's
// plan, apply and probe cannot disagree with the command.
type CatalogInstaller interface {
	Plan(tools.Tool) tools.Plan
	Install(tools.Tool) (tools.Result, error)
}

// catalogPass is one walk over packages.json: what it installed (or, planned,
// would install), and what it left for a reason converge cannot change.
type catalogPass struct {
	changed  []string // installed or upgraded; in a plan, to install or upgrade
	waiting  []wait   // the manager arrives later in the run, or never on this machine
	needSudo []string // tool names: dotf never asks for a password
	sudoPkgs []string // their apt packages, for one SudoInstallCommand
	failed   []error
}

// wait is an entry whose package manager is not on PATH.
type wait struct {
	tool    tools.Tool
	manager string
}

// retry is what a second pass walks: only the entries that waited, so a
// failure is attempted and reported once.
func (p catalogPass) retry() []tools.Tool {
	entries := make([]tools.Tool, 0, len(p.waiting))
	for _, w := range p.waiting {
		entries = append(entries, w.tool)
	}
	return entries
}

// then folds a second pass into the first: its installs and failures add up,
// and its waits are what is still waiting.
func (p *catalogPass) then(o catalogPass) {
	p.changed = append(p.changed, o.changed...)
	p.failed = append(p.failed, o.failed...)
	p.needSudo = append(p.needSudo, o.needSudo...)
	p.sudoPkgs = append(p.sudoPkgs, o.sudoPkgs...)
	p.waiting = o.waiting
}

func loadCatalog(env Env) ([]tools.Tool, error) {
	cat, err := tools.Load(filepath.Join(env.RepoRoot, "packages.json"))
	return cat.Tools, err
}

// walkCatalog plans every entry and, when apply is set, installs the ones the
// plan says to. An entry Install refuses fails the pass, as it fails
// `dotf tools install`; the others still run.
func walkCatalog(in CatalogInstaller, entries []tools.Tool, apply bool) catalogPass {
	var p catalogPass
	for _, t := range entries {
		plan := in.Plan(t)
		switch plan.Action {
		case tools.PlanInstall, tools.PlanUpgrade:
			p.record(in, t, apply)
		case tools.PlanMissingManager:
			p.waiting = append(p.waiting, wait{t, strings.TrimPrefix(plan.Note, "waits on ")})
		case tools.PlanNeedsSudo:
			p.needSudo = append(p.needSudo, t.Name)
			p.sudoPkgs = append(p.sudoPkgs, plan.Package)
		case tools.PlanRefused:
			p.failed = append(p.failed, fmt.Errorf("%s: %s", t.Name, plan.Note))
		}
	}
	return p
}

func (p *catalogPass) record(in CatalogInstaller, t tools.Tool, apply bool) {
	if !apply {
		p.changed = append(p.changed, t.Name)
		return
	}
	res, err := in.Install(t)
	if err != nil {
		p.failed = append(p.failed, err)
	}
	if res == tools.Installed || res == tools.Upgraded {
		p.changed = append(p.changed, t.Name)
	}
}

func (p catalogPass) err() error {
	if len(p.failed) == 0 {
		return nil
	}
	return fmt.Errorf("catalog: %w", errors.Join(p.failed...))
}

func (p catalogPass) detail(dryRun bool) string {
	verb := "installed"
	if dryRun {
		verb = "to install"
	}
	parts := []string{}
	if len(p.changed) > 0 {
		parts = append(parts, "catalog "+verb+": "+strings.Join(p.changed, ", "))
	}
	if len(p.waiting) > 0 {
		waits := make([]string, 0, len(p.waiting))
		for _, w := range p.waiting {
			waits = append(waits, w.tool.Name+" ("+w.manager+")")
		}
		parts = append(parts, "waiting on a manager: "+strings.Join(waits, ", "))
	}
	if len(p.needSudo) > 0 {
		parts = append(parts, "needs sudo: "+strings.Join(p.needSudo, ", ")+
			" (run once: "+tools.SudoInstallCommand(p.sudoPkgs)+")")
	}
	return strings.Join(parts, "; ")
}

// unreachable names the managers this machine's own tool layer installs (a
// catalog entry for this OS, or a mise pin) that a pass still waited on, or
// that the sync could not find. After an apply that is not a wait but a PATH
// that does not reach what converge placed, and the probe fails on it.
func unreachable(p catalogPass, provided map[string]bool, miseMissing bool) []string {
	var out []string
	if miseMissing && provided["mise"] {
		out = append(out, "mise")
	}
	for _, w := range p.waiting {
		if provided[w.manager] {
			out = append(out, w.manager+" (for "+w.tool.Name+")")
		}
	}
	return out
}

// provided is every tool name this OS's tool layer installs.
func provided(entries []tools.Tool, pins []tools.MiseTool, goos string) map[string]bool {
	names := map[string]bool{}
	for _, t := range entries {
		if t.SupportsOS(goos) {
			names[t.Name] = true
		}
	}
	for _, t := range pins {
		names[t.Name] = true
	}
	return names
}
