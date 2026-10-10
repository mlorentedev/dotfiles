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
	waiting  []string // "hive (uv)": the manager arrives later, or never on this machine
	needSudo []string // "gh (run: sudo apt-get …)": dotf never asks for a password
	failed   []error
}

func (p *catalogPass) add(o catalogPass) {
	p.changed = append(p.changed, o.changed...)
	p.failed = append(p.failed, o.failed...)
	p.waiting, p.needSudo = o.waiting, o.needSudo // the later pass is the current state
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
			p.waiting = append(p.waiting, t.Name+" ("+strings.TrimPrefix(plan.Note, "waits on ")+")")
		case tools.PlanNeedsSudo:
			p.needSudo = append(p.needSudo, t.Name+" ("+plan.Note+")")
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
		parts = append(parts, "waiting on a manager: "+strings.Join(p.waiting, ", "))
	}
	if len(p.needSudo) > 0 {
		parts = append(parts, "needs sudo: "+strings.Join(p.needSudo, ", "))
	}
	return strings.Join(parts, "; ")
}
