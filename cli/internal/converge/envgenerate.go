package converge

import (
	"fmt"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// envGenerate keeps the path file (paths.sh, or paths.ps1 on Windows) current
// with the env contract and the machine overrides (#2013 P6, F-005). The rc
// files source it, and they render it themselves only when it is MISSING; a
// contract or machine.json change leaves a stale file that doctor fails and
// nothing repaired. It runs right after records-mirror, so the deploy dir
// exists, and before tools, so the paths exist before any agent is installed.
type envGenerate struct{}

func (envGenerate) Name() string        { return "env-generate" }
func (envGenerate) Platforms() []string { return nil }

func (envGenerate) options(e Env, check bool) env.Options {
	return env.Options{
		ContractPath: filepath.Join(e.RepoRoot, "env-contract.json"),
		MachinePath:  env.MachinePath(e.Home),
		GOOS:         e.GOOS,
		Home:         e.Home,
		Output:       env.DefaultOutput(e.GOOS, e.DeployDir),
		Check:        check,
	}
}

func (r envGenerate) Reconcile(e Env, dryRun bool) (Result, error) {
	res, err := env.Generate(r.options(e, true))
	if err != nil {
		return Result{}, err
	}
	if !res.Drifted {
		return Result{Detail: res.Output + " up to date"}, nil
	}
	if dryRun {
		return Result{Changes: 1, Detail: res.Output + " to write"}, nil
	}
	if _, err := env.Generate(r.options(e, false)); err != nil {
		return Result{}, err
	}
	return Result{Changes: 1, Detail: res.Output + " written"}, nil
}

// Probe holds the post-condition: the file on disk is what the contract and
// the machine overrides render to now.
func (r envGenerate) Probe(e Env) error {
	res, err := env.Generate(r.options(e, true))
	if err != nil {
		return err
	}
	if res.Drifted {
		return fmt.Errorf("%s still differs from what the env contract renders", res.Output)
	}
	return nil
}
