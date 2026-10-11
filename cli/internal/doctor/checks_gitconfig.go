package doctor

import (
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// checkGitConfig holds the global git configuration to what the dotfiles need
// from it, through the predicate `dotf converge` applies (gitconfig package,
// lesson 368): ~/.gitconfig includes the deployed dotfiles.gitconfig, and
// GitHub's credential helper runs gh by an absolute path. A bare `gh` works in
// a terminal and fails in every process without the shell's PATH: measured on
// the Mac, obsidian-git asked for a GitHub password (#2207). With fix it runs
// the same Apply as converge.
func checkGitConfig(sys *System, rep *Report, fix bool) {
	rep.Section("Git config (global)")
	if !sys.has("git") {
		rep.Skip("git not on PATH")
		return
	}
	home := sys.home()
	m := gitconfig.Machine{
		Home: home,
		Run: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandStdoutDir(home, name, args...)
			return []byte(out), err
		},
		OnPath: sys.has,
		Exists: pathExists,
		StoredAuth: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandOutputEnv(gitconfig.WithoutEnvTokens(os.Environ()), name, args...)
			return []byte(out), err
		},
	}
	st, err := gitconfig.Inspect(m)
	if err != nil {
		rep.Fail("cannot read the global git config: " + err.Error())
		return
	}
	if fix && st.Repairable() > 0 {
		if err := gitconfig.Apply(m, st); err != nil {
			rep.Fail("repairing the global git config: " + err.Error())
			return
		}
		after, err := gitconfig.Inspect(m)
		if err != nil {
			rep.Fail("cannot re-read the global git config: " + err.Error())
			return
		}
		if st.IncludeMissing && !after.IncludeMissing {
			rep.Fix("~/.gitconfig now includes " + gitconfig.IncludePath)
		}
		if len(st.BadHelpers) > 0 && len(after.BadHelpers) == 0 {
			rep.Fix("GitHub's credential helper set by gh auth setup-git")
		}
		st = after
	}
	reportGitConfig(st, rep)
}

func reportGitConfig(st gitconfig.State, rep *Report) {
	if st.IncludeMissing {
		rep.Fail("~/.gitconfig does not include " + gitconfig.IncludePath + ", where `dotf deploy` puts the dotfiles' git settings (run: dotf doctor --fix)")
	}
	if st.TargetMissing {
		rep.Fail(gitconfig.IncludePath + " is missing, so git reads none of the dotfiles' settings (run: dotf deploy)")
	}
	switch {
	case len(st.BadHelpers) == 0:
	case st.Blocked != "":
		rep.Warn("GitHub's credential helper needs the shell's PATH, so a GUI app, launchd or cron cannot authenticate (" +
			strings.Join(st.BadHelpers, "; ") + "); " + st.Blocked + ", then run: dotf doctor --fix")
	default:
		rep.Fail("GitHub's credential helper needs the shell's PATH, so a GUI app, launchd or cron cannot authenticate (" +
			strings.Join(st.BadHelpers, "; ") + ") (run: dotf doctor --fix, which runs gh auth setup-git)")
	}
	if st.Converged() {
		rep.Pass("~/.gitconfig includes " + gitconfig.IncludePath + "; GitHub's credential helper runs gh by absolute path")
	}
}
