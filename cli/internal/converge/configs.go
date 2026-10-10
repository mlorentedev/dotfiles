package converge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

// configsDeploy converges every ai/deploy.json entry that applies to this
// machine (#1843 B13): the agent settings, shell rc files, git and ssh configs
// `dotf deploy` installs. It runs deploy.Run, the loop behind `dotf deploy`, so
// a template change merged to main reaches the machine on the next converge
// with no manual step. Every OS.
//
// Not here: Orca's hooks and Claude Code's MCP servers and plugins, which a
// bare `dotf deploy` also converges; their reconciler is #1843 B10.
type configsDeploy struct {
	render  deploy.Renderer     // strict: ErrRenderIncomplete when the store is locked
	resolve func(string) string // env.ResolvePath in production
	has     func(string) bool   // is a command on PATH
}

func (configsDeploy) Name() string        { return "configs-deploy" }
func (configsDeploy) Platforms() []string { return nil }

func (r configsDeploy) run(env Env, dryRun bool) (deploy.RunResult, error) {
	if r.render == nil || r.resolve == nil {
		return deploy.RunResult{}, errors.New("no secrets renderer or path resolver is wired into this registry")
	}
	raw, err := os.ReadFile(filepath.Join(env.RepoRoot, filepath.FromSlash(deploy.ManifestRel))) //nolint:gosec // checkout-relative, fixed name
	if err != nil {
		return deploy.RunResult{}, fmt.Errorf("reading %s: %w", deploy.ManifestRel, err)
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		return deploy.RunResult{}, err
	}
	return deploy.Run(man, man.Configs, deploy.RunOptions{
		RepoRoot:  env.RepoRoot,
		Home:      env.Home,
		GOOS:      env.GOOS,
		Resolve:   r.resolve,
		Render:    r.render,
		Available: r.has,
		DryRun:    dryRun,
	})
}

func (r configsDeploy) Reconcile(env Env, dryRun bool) (Result, error) {
	res, err := r.run(env, dryRun)
	if err != nil {
		return Result{}, err
	}
	changed := res.Changed()
	return Result{Changes: len(changed) + len(res.Tightened), Detail: configsDetail(res, changed, dryRun)}, nil
}

// Probe re-plans: after an apply, no entry may still differ from the checkout.
// An entry skipped because the secret store was locked is not a difference; the
// detail already names it.
func (r configsDeploy) Probe(env Env) error {
	res, err := r.run(env, true)
	if err != nil {
		return err
	}
	if changed := res.Changed(); len(changed) > 0 {
		return fmt.Errorf("config(s) still differ from the checkout after the deploy: %s", strings.Join(changed, ", "))
	}
	if len(res.Tightened) > 0 {
		return fmt.Errorf("%d directory(ies) holding a private config are still too open", len(res.Tightened))
	}
	return nil
}

// configsDetail names what changed and what was kept back, and counts the
// entries that do not apply here, so a run that skipped a config never reads
// as one that converged it.
func configsDetail(res deploy.RunResult, changed []string, dryRun bool) string {
	var kept, absent []string
	notHere := 0
	for _, s := range res.Steps {
		switch {
		case s.Kept:
			kept = append(kept, s.Name)
		case s.Absent != "":
			absent = append(absent, fmt.Sprintf("%s (%s not installed)", s.Name, s.Absent))
		case s.Skipped != "":
			notHere++
		}
	}
	verb := "deployed"
	if dryRun {
		verb = "to deploy"
	}
	parts := []string{fmt.Sprintf("%d config(s) in sync", len(res.Steps)-len(changed)-len(kept)-len(absent)-notHere)}
	if len(changed) > 0 {
		parts = append(parts, fmt.Sprintf("%d %s: %s", len(changed), verb, strings.Join(changed, ", ")))
	}
	if len(kept) > 0 {
		parts = append(parts, "kept (secrets locked): "+strings.Join(kept, ", "))
	}
	if len(absent) > 0 {
		parts = append(parts, "skipped, command absent: "+strings.Join(absent, ", "))
	}
	if notHere > 0 {
		parts = append(parts, fmt.Sprintf("%d not for this machine", notHere))
	}
	return strings.Join(parts, "; ")
}
