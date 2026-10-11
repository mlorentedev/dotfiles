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
	run    gitconfig.Runner  // gitconfig.ExecRunner in production
	stored gitconfig.Runner  // gitconfig.StoredLoginRunner in production; nil means run
	has    func(string) bool // is a command on PATH
}

func (gitConfig) Name() string        { return "git-config" }
func (gitConfig) Platforms() []string { return nil }

func (r gitConfig) machine(env Env) gitconfig.Machine {
	return gitconfig.Machine{
		Home:       env.Home,
		Run:        r.run,
		OnPath:     r.has,
		Exists:     func(p string) bool { _, err := os.Stat(p); return err == nil },
		StoredAuth: r.stored,
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

// Probe holds the post-condition: the include is in place and names a file
// that exists, and every helper is gh's absolute form unless gh itself is what
// is missing, which converge cannot provide and the detail already names. An
// include of an undeployed file fails: git ignores it without a word, so a run
// that passed there would leave git reading none of the dotfiles' settings.
func (r gitConfig) Probe(env Env) error {
	st, err := gitconfig.Inspect(r.machine(env))
	if err != nil {
		return err
	}
	if st.IncludeMissing {
		return fmt.Errorf("~/.gitconfig still does not include %s", gitconfig.IncludePath)
	}
	if st.TargetMissing {
		return fmt.Errorf("%s is not deployed, so git reads none of the dotfiles' settings (run: dotf deploy)", gitconfig.IncludePath)
	}
	if len(st.BadHelpers) > 0 && st.Blocked == "" {
		return fmt.Errorf("after gh auth setup-git, the credential helper is still not gh's absolute path: %s", strings.Join(st.BadHelpers, "; "))
	}
	return nil
}

// gitDetail names what a run changes apart from what it leaves, so a run that
// changed nothing never reads as "applied".
func gitDetail(st gitconfig.State, dryRun bool) string {
	if st.Converged() {
		return "include and GitHub credential helper in place"
	}
	var fix, left []string
	if st.IncludeMissing {
		fix = append(fix, "include "+gitconfig.IncludePath)
	}
	if len(st.BadHelpers) > 0 {
		if st.Blocked != "" {
			left = append(left, "credential helper: "+st.Blocked)
		} else {
			fix = append(fix, "credential helper through gh auth setup-git")
		}
	}
	if st.TargetMissing {
		left = append(left, gitconfig.IncludePath+" is not deployed (run: dotf deploy)")
	}
	var parts []string
	if len(fix) > 0 {
		verb := "applied: "
		if dryRun {
			verb = "to apply: "
		}
		parts = append(parts, verb+strings.Join(fix, "; "))
	}
	if len(left) > 0 {
		parts = append(parts, "left as it is: "+strings.Join(left, "; "))
	}
	return strings.Join(parts, "; ")
}
