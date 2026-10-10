package cmd

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-deploy: fast-forward the dotfiles repo and converge from it (opt-in, scheduler-invoked)",
		Long: `update is the opt-in self-deploy entrypoint the systemd --user timer (Linux)
/ Scheduled Task (Windows) invokes. It pulls the dotfiles repo and runs
'dotf converge' from it (which runs the setup script last on Linux and
Windows) — but ONLY via a clean fast-forward, and ONLY when HEAD moved.

Anything non-actionable (not a git repo, dirty worktree, unreachable remote, no
upstream, already current, diverged history) is logged and skipped with exit 0.
A real converge failure is the ONLY non-zero exit, so 'systemctl --user status
dotfiles-selfupdate' (and the journal) surface it.

Env:
  DOTFILES_REPO_DIR              repo to update  (default: the env-contract path)
  DOTFILES_SELFUPDATE_SETUP_CMD  setup command converge runs last (default:
                                 <repo>/setup-linux.sh or setup-windows.ps1)`,
		Example:       "  dotf update",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			repo := repoForUpdate()
			out, err := update.Run(update.Config{Repo: repo}, update.Deps{
				Git:      gitRunner(repo),
				Converge: func() error { return runConverge(c.OutOrStdout(), repo, false, nil) },
			})
			c.Println(out.Message)
			if err != nil {
				c.PrintErrln("update:", err)
			}
			return err
		},
	}
	return cmd
}

// repoForUpdate resolves the dotfiles checkout to fast-forward. The cascade
// (ResolvePath: env → machine.json → contract default, ADR-025) is primary, but
// only when it points at a real directory: on a fresh machine with no
// machine.json the contract default is the phantom ~/Projects/dotfiles, which
// must not win over a discoverable checkout (#696). It then falls back to the
// .git walk-up when it lands on a dotfiles checkout (interactive `dotf update`
// from inside one; another project's repository is not fast-forwarded), and finally to
// the literal default for a bare scheduler env (systemd --user / Task Scheduler)
// with neither a seeded machine.json nor a discoverable repo — reproducing the
// shell twins' '${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}'.
func repoForUpdate() string {
	if r := env.ResolvePath("DOTFILES_REPO_DIR"); r != "" && dirExists(r) {
		return r
	}
	if r := env.RepoDir(); r != "" && env.IsDotfilesCheckout(r) {
		return r
	}
	return env.DefaultCheckoutDir(env.Home())
}

// dirExists reports whether p exists and is a directory. Shared by the repo-dir
// resolvers in this package (update, mem) to reject a cascade value that names a
// path that is not actually there.
func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// gitRunner returns a Git seam that runs `git -C <repo> <args...>` and returns
// trimmed stdout. On failure the error is returned (stdout ignored) — callers
// treat a failed git query as a skip condition, so surfacing stderr is not
// needed here.
func gitRunner(repo string) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output() //nolint:gosec // fixed git subcommands, repo is the resolved checkout
		return strings.TrimSpace(string(out)), err
	}
}
