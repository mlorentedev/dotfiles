package converge

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/hooks"
)

// gitHooks deploys the GUARD-001 dispatcher from the checkout into the deploy
// dir and wires git's global core.hooksPath at it when it is unset (hooks
// package, CLI-072). Every OS. It runs what `dotf hooks install` runs, which
// the setup twins called and macOS never did (#2013 X1). A hooksPath that
// points elsewhere is preserved and named, never counted as a change.
type gitHooks struct {
	run gitconfig.Runner  // gitconfig.ExecRunner in production
	has func(string) bool // is a command on PATH
}

func (gitHooks) Name() string        { return "git-hooks" }
func (gitHooks) Platforms() []string { return nil }

func (r gitHooks) opts(env Env) hooks.Options {
	return hooks.Options{Source: filepath.Join(env.RepoRoot, "git-hooks"), DotfilesDir: env.DeployDir}
}

func (r gitHooks) git(ctx context.Context, args ...string) ([]byte, error) {
	return r.run("git", args...)
}

func (r gitHooks) Reconcile(env Env, dryRun bool) (Result, error) {
	if !r.has("git") {
		return Result{Skip: "git is not on PATH"}, nil
	}
	if r.run == nil {
		return Result{Skip: "no git runner is wired into this registry"}, nil
	}
	ctx := context.Background()
	st, err := hooks.Inspect(ctx, r.git, r.opts(env), env.GOOS)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: st.Changes(), Detail: hooksDetail(st, dryRun)}
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	return res, hooks.Apply(ctx, r.git, r.opts(env))
}

// Probe holds the post-condition: the mirror holds what the checkout has, and
// core.hooksPath is set. A foreign hooksPath passes: converge never takes it
// over, and the detail already says the guard is inactive.
func (r gitHooks) Probe(env Env) error {
	st, err := hooks.Inspect(context.Background(), r.git, r.opts(env), env.GOOS)
	if err != nil {
		return err
	}
	if st.MirrorStale {
		return fmt.Errorf("%s still differs from the checkout's git-hooks/", st.Dest)
	}
	if st.Wiring == hooks.WiringUnset {
		return fmt.Errorf("core.hooksPath is still unset, so the GUARD dispatcher never runs")
	}
	return nil
}

// hooksDetail names what a run changes apart from what it leaves.
func hooksDetail(st hooks.State, dryRun bool) string {
	var fix []string
	if st.MirrorStale {
		fix = append(fix, "deploy the dispatcher to "+st.Dest)
	}
	if st.Wiring == hooks.WiringUnset {
		fix = append(fix, "wire core.hooksPath")
	}
	var parts []string
	if len(fix) > 0 {
		verb := "applied: "
		if dryRun {
			verb = "to apply: "
		}
		parts = append(parts, verb+strings.Join(fix, "; "))
	}
	switch st.Wiring {
	case hooks.WiringForeign:
		parts = append(parts, fmt.Sprintf("left as it is: core.hooksPath is %s, not a GUARD dispatcher, so the guard is inactive", st.HooksPath))
	case hooks.WiringEquivalent:
		parts = append(parts, "the guard runs through "+st.HooksPath)
	}
	if len(parts) == 0 {
		return "dispatcher deployed and wired"
	}
	return strings.Join(parts, "; ")
}
