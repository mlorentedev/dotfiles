package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-deploy: fast-forward the dotfiles repo and re-run setup (opt-in, scheduler-invoked)",
		Long: `update is the opt-in self-deploy entrypoint the systemd --user timer (Linux)
/ Scheduled Task (Windows) invokes. It pulls the dotfiles repo and re-runs the
idempotent setup — but ONLY via a clean fast-forward, and ONLY when HEAD moved.

Anything non-actionable (not a git repo, dirty worktree, unreachable remote, no
upstream, already current, diverged history) is logged and skipped with exit 0.
A real setup failure is the ONLY non-zero exit, so 'systemctl --user status
dotfiles-selfupdate' (and the journal) surface it.

It is the Go port of scripts/dotfiles-selfupdate.{sh,ps1} (ADR-020 convergence).

Env:
  DOTFILES_REPO_DIR              repo to update  (default: the env-contract path)
  DOTFILES_SELFUPDATE_SETUP_CMD  setup to re-run (default: <repo>/setup-linux.sh
                                 or setup-windows.ps1)`,
		Example:       "  dotf update",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			repo := repoForUpdate()
			out, err := update.Run(update.Config{Repo: repo}, update.Deps{
				Git:      gitRunner(repo),
				RunSetup: setupRunner(setupCmdPath(repo)),
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

// repoForUpdate resolves the dotfiles checkout to fast-forward. The seam
// (ResolvePath: env → machine.json → contract default, ADR-025) is primary; the
// literal fallback is the last resort for a bare scheduler env (systemd --user /
// Task Scheduler) that lacks DOTFILES_REPO_DIR *and* cannot discover the
// contract. It reproduces the shell twins' '${DOTFILES_REPO_DIR:-$HOME/Projects/
// dotfiles}' exactly, so self-deploy stays self-contained under a timer.
func repoForUpdate() string {
	if r := env.ResolvePath("DOTFILES_REPO_DIR"); r != "" {
		return r
	}
	return filepath.Join(env.Home(), "Projects", "dotfiles")
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

// setupRunner returns a RunSetup seam that execs the setup command with the
// process's stdout/stderr attached (so a timer run streams to the journal). The
// setup shell is OS-specific: a POSIX script runs directly; a .ps1 needs pwsh.
func setupRunner(setupCmd string) func() error {
	return func() error {
		var c *exec.Cmd
		if runtime.GOOS == "windows" {
			c = exec.Command("pwsh", "-NoProfile", "-File", setupCmd)
		} else {
			c = exec.Command(setupCmd) //nolint:gosec // the resolved setup entrypoint
		}
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
}

// setupCmdPath resolves the setup entrypoint: the DOTFILES_SELFUPDATE_SETUP_CMD
// override, else the OS-appropriate setup script under the repo. Mirrors the
// shell default '${DOTFILES_SELFUPDATE_SETUP_CMD:-<repo>/setup-linux.sh}'.
func setupCmdPath(repo string) string {
	if c := os.Getenv("DOTFILES_SELFUPDATE_SETUP_CMD"); c != "" {
		return c
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(repo, "setup-windows.ps1")
	}
	return filepath.Join(repo, "setup-linux.sh")
}
