package converge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/pwsh"
)

// pwshModules installs the PowerShell Gallery modules versions.conf marks
// "# pwsh-module: <Name>" at their exact pin (PLAT-003): PSScriptAnalyzer for
// `dotf lint powershell` and Pester for the Windows suites, the versions CI's
// jobs install, so a local run reaches CI's verdict. Every OS where pwsh is on
// PATH; it runs after tools, which installs pwsh where packages.json names a
// manager.
type pwshModules struct {
	run pwsh.Runner       // tools.ExecRunner in production
	has func(string) bool // is a command on PATH
}

func (pwshModules) Name() string        { return "pwsh-modules" }
func (pwshModules) Platforms() []string { return nil }

func (r pwshModules) Reconcile(env Env, dryRun bool) (Result, error) {
	if !r.has(pwsh.Command) {
		return Result{Skip: "pwsh is not on PATH; the tools step installs it where packages.json names a manager for this OS"}, nil
	}
	if r.run == nil {
		return Result{Skip: "no pwsh runner is wired into this registry"}, nil
	}
	mods, missing, err := r.state(env)
	if err != nil || len(mods) == 0 {
		return Result{Detail: "versions.conf marks no PowerShell module"}, err
	}
	if len(missing) == 0 {
		return Result{Detail: fmt.Sprintf("%d module(s) at their pin: %s", len(mods), pwsh.Names(mods))}, nil
	}
	verb := "installed"
	if dryRun {
		verb = "to install"
	}
	res := Result{Changes: len(missing), Detail: verb + " from the PowerShell Gallery: " + pwsh.Names(missing)}
	if dryRun {
		return res, nil
	}
	return res, pwsh.Install(r.run, missing)
}

// Probe holds the post-condition: every marked module imports at its pin.
func (r pwshModules) Probe(env Env) error {
	if !r.has(pwsh.Command) || r.run == nil {
		return nil
	}
	_, missing, err := r.state(env)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("still not importable at their pin: %s", pwsh.Names(missing))
	}
	return nil
}

// state reads the checkout's marked modules and those pwsh cannot import.
func (r pwshModules) state(env Env) (mods, missing []pwsh.Module, err error) {
	raw, err := os.ReadFile(filepath.Join(env.RepoRoot, "versions.conf"))
	if err != nil {
		return nil, nil, err
	}
	if mods, err = pwsh.ParseModules(raw); err != nil || len(mods) == 0 {
		return nil, nil, err
	}
	missing, err = pwsh.Missing(r.run, mods)
	return mods, missing, err
}
