package converge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// toolsSync installs the pinned CLIs through mise (`dotf tools sync`,
// ADR-044): the versions.conf pins marked "# mise: cli". It runs after the
// records, so an agent that starts once a tool lands already has its
// instructions. Every OS: mise and the aqua backend are native on all three.
type toolsSync struct {
	run, stdout tools.Runner      // tools.HomeRunners in production
	has         func(string) bool // is a command on PATH
	getenv      func(string) string
}

func (toolsSync) Name() string        { return "tools" }
func (toolsSync) Platforms() []string { return nil }

func (r toolsSync) Reconcile(env Env, dryRun bool) (Result, error) {
	if skip := r.unavailable(); skip != "" {
		return Result{Skip: skip}, nil
	}
	s, pins, err := r.sync(env)
	if err != nil {
		return Result{}, err
	}
	plan := s.Plan
	if !dryRun {
		plan = s.Apply
	}
	p, err := plan(pins)
	res := Result{Changes: len(p.Missing) + len(p.MissingPackages), Detail: syncDetail(p, len(pins), dryRun)}
	if p.ConfigChanged {
		res.Changes++
	}
	return res, err
}

// Probe holds the post-condition: the rendered config is current and every
// pinned CLI runs at or above its pin through mise.
func (r toolsSync) Probe(env Env) error {
	s, pins, err := r.sync(env)
	if err != nil {
		return err
	}
	p, err := s.Plan(pins)
	if err != nil {
		return err
	}
	if p.ConfigChanged || len(p.Missing) > 0 || len(p.MissingPackages) > 0 {
		return fmt.Errorf("after the sync, config current=%v, not at their pin: %s", !p.ConfigChanged, strings.Join(append(p.Missing, p.MissingPackages...), ", "))
	}
	return nil
}

// unavailable names the prerequisite this machine lacks, or "".
func (r toolsSync) unavailable() string {
	switch {
	case !r.has("mise"):
		return "mise is not on PATH; `dotf tools install mise` installs it once the catalog carries it"
	case r.run == nil || r.stdout == nil:
		return "no mise runner is wired into this registry"
	}
	return ""
}

func (r toolsSync) sync(env Env) (tools.MiseSync, []tools.MiseTool, error) {
	raw, err := os.ReadFile(filepath.Join(env.RepoRoot, "versions.conf")) //nolint:gosec // the checkout's own pin file
	if err != nil {
		return tools.MiseSync{}, nil, err
	}
	all, err := tools.ParseMisePins(raw)
	if err != nil {
		return tools.MiseSync{}, nil, err
	}
	getenv := r.getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	s := tools.MiseSync{ConfigDir: tools.MiseConfigDir(env.Home, getenv), Run: r.run, Stdout: r.stdout, PythonPackages: all.PythonPackages}
	return s, all.Tools, nil
}

func syncDetail(p tools.SyncPlan, total int, dryRun bool) string {
	if !p.ConfigChanged && len(p.Missing) == 0 && len(p.MissingPackages) == 0 {
		return fmt.Sprintf("%d pinned CLI(s) at their pin", total)
	}
	verb := "installed"
	if dryRun {
		verb = "to install"
	}
	parts := []string{}
	if p.ConfigChanged {
		parts = append(parts, "mise config to write")
	}
	if len(p.Missing) > 0 {
		parts = append(parts, verb+": "+strings.Join(p.Missing, ", "))
	}
	if len(p.MissingPackages) > 0 {
		parts = append(parts, "python packages "+verb+": "+strings.Join(p.MissingPackages, ", "))
	}
	return strings.Join(parts, "; ")
}
