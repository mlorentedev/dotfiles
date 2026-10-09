package converge

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// gitConfig converges the global git configuration's two dotfiles-owned
// parts (gitconfig package, #2207): ~/.gitconfig includes the deployed
// dotfiles.gitconfig, and GitHub's credential helper is gh's absolute form,
// which a process without the shell's PATH can still run. Every OS.
type gitConfig struct {
	run gitconfig.Runner  // gitconfig.ExecRunner in production
	has func(string) bool // is a command on PATH
}

func (gitConfig) Name() string        { return "git-config" }
func (gitConfig) Platforms() []string { return nil }

func (r gitConfig) machine(env Env) gitconfig.Machine {
	return gitconfig.Machine{
		Home:   env.Home,
		Run:    r.run,
		OnPath: r.has,
		Exists: func(p string) bool { _, err := os.Stat(p); return err == nil },
	}
}

func (r gitConfig) Reconcile(env Env, dryRun bool) (Result, error) {
	if !r.has("git") {
		return Result{Skip: "git is not on PATH"}, nil
	}
	if r.run == nil {
		return Result{Skip: "no git runner is wired into this registry"}, nil
	}
	m := r.machine(env)
	st, err := gitconfig.Inspect(m)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: st.Repairable(), Detail: gitDetail(st, dryRun)}
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	return res, gitconfig.Apply(m, st)
}

// Probe holds the post-condition: the include is in place, and every helper is
// gh's absolute form unless gh itself is what is missing, which converge
// cannot provide and the detail already names.
func (r gitConfig) Probe(env Env) error {
	st, err := gitconfig.Inspect(r.machine(env))
	if err != nil {
		return err
	}
	if st.IncludeMissing {
		return fmt.Errorf("~/.gitconfig still does not include %s", gitconfig.IncludePath)
	}
	if len(st.BadHelpers) > 0 && st.Blocked == "" {
		return fmt.Errorf("after gh auth setup-git, the credential helper is still not gh's absolute path: %s", strings.Join(st.BadHelpers, "; "))
	}
	return nil
}

func gitDetail(st gitconfig.State, dryRun bool) string {
	if st.Converged() {
		return "include and GitHub credential helper in place"
	}
	var parts []string
	if st.IncludeMissing {
		parts = append(parts, "include "+gitconfig.IncludePath)
	}
	if len(st.BadHelpers) > 0 {
		if st.Blocked != "" {
			parts = append(parts, "credential helper left as it is: "+st.Blocked)
		} else {
			parts = append(parts, "credential helper through gh auth setup-git")
		}
	}
	verb := "applied: "
	if dryRun {
		verb = "to apply: "
	}
	return verb + strings.Join(parts, "; ")
}
