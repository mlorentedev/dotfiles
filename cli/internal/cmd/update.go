package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/doctor"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

func newUpdateCmd() *cobra.Command {
	var homes updateHomes
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-deploy: fast-forward the dotfiles repo and converge from it (opt-in, scheduler-invoked)",
		Long: `update is the opt-in self-deploy entrypoint the native scheduler invokes once a
day: a launchd agent (macOS), a systemd --user timer (Linux) or a Scheduled
Task (Windows), which the 'dotf converge' step self-deploy installs on a machine
that declares the role ('dotf env role add self-deploy'). It pulls the dotfiles
repo and runs 'dotf converge' from it (which runs the setup script last on
Linux and Windows) — but ONLY via a clean fast-forward, and ONLY when HEAD moved.

The checkout's DOTF_VERSION pin is part of being up to date. When the running
dotf is not that version (after a fast-forward, or when the checkout was
already current), update runs the checkout's installer (install.sh, or
install.ps1 under pwsh), which installs the pin and converges with the NEW
binary; update does not converge a second time with the old one. A source
build ("dev") is never replaced: the pin check is skipped and said so.

Anything non-actionable (not a git repo, dirty worktree, unreachable remote, no
upstream, ahead of the upstream, diverged history, or already current with dotf
at its pin) is logged and skipped with exit 0; a skip names a dotf that is off
its pin.
A real converge failure, or a pinned dotf that cannot be installed, is the ONLY
non-zero exit, so the scheduler's last result (launchd's log, 'systemctl --user
status dotfiles-selfupdate', the task's LastTaskResult) surfaces it. Every run,
a skip included, records its outcome in <state dir>/update/last.json, which
'dotf doctor' reads in its Self-deploy section.

--config-home and --state-home set XDG_CONFIG_HOME and XDG_STATE_HOME for the
run and everything it starts. The Windows Scheduled Task passes them, because a
task has no per-action environment; launchd and systemd set the variables
themselves.

Env:
  DOTFILES_REPO_DIR              repo to update  (default: the env-contract path)
  DOTFILES_SELFUPDATE_SETUP_CMD  setup command converge runs last (default:
                                 <repo>/setup-linux.sh or setup-windows.ps1)`,
		Example:       "  dotf update",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if err := homes.apply(); err != nil {
				return err
			}
			repo := repoForUpdate()
			out, err := update.Run(update.Config{Repo: repo, Version: dotfVersion}, update.Deps{
				Git:      gitRunner(repo),
				Converge: func() error { return runConverge(c.OutOrStdout(), repo, false, nil) },
				Pin:      func() (string, error) { return pinnedDotfVersion(repo) },
				Install:  func(pin string) error { return runInstaller(c, repo, pin) },
			})
			c.Println(out.Message)
			if err != nil {
				c.PrintErrln("update:", err)
			}
			recordUpdate(c.ErrOrStderr(), out, err)
			return err
		},
	}
	cmd.Flags().StringVar(&homes.config, "config-home", "", "absolute XDG_CONFIG_HOME for this run (the Windows task passes it)")
	cmd.Flags().StringVar(&homes.state, "state-home", "", "absolute XDG_STATE_HOME for this run (the Windows task passes it)")
	return cmd
}

// updateHomes are the XDG homes a scheduler hands `dotf update` as flags.
type updateHomes struct{ config, state string }

// apply exports each home given, before anything resolves machine.json or the
// state dir, so the converge and the installer the run starts inherit it. A
// relative one is refused: env ignores a relative XDG_STATE_HOME, and the run
// would quietly use another home.
func (h updateHomes) apply() error {
	for _, v := range []struct{ flag, env, dir string }{{"--config-home", "XDG_CONFIG_HOME", h.config}, {"--state-home", "XDG_STATE_HOME", h.state}} {
		if v.dir == "" {
			continue
		}
		if !filepath.IsAbs(v.dir) {
			return fmt.Errorf("%s %q: want an absolute path", v.flag, v.dir)
		}
		if err := os.Setenv(v.env, v.dir); err != nil {
			return err
		}
	}
	return nil
}

// recordUpdate keeps the run's outcome where doctor reads it. A record that
// cannot be written is said and leaves the exit alone: update fails only when
// the deploy broke, and doctor reads a missing record as no recent run.
func recordUpdate(w io.Writer, out update.Outcome, runErr error) {
	state, err := env.StateDir()
	if err == nil {
		err = update.WriteRecord(state, update.NewRecord(time.Now(), out, runErr))
	}
	if err != nil {
		_, _ = fmt.Fprintf(w, "warning: recording this run for doctor: %v\n", err)
	}
}

// pinnedDotfVersion reads DOTF_VERSION from the checkout's versions.conf, the
// file doctor compares the installed dotf against. "" means it is not pinned.
func pinnedDotfVersion(repo string) (string, error) {
	pins, err := doctor.ParseVersionsConf(filepath.Join(repo, env.CheckoutMarker))
	if err != nil {
		return "", err
	}
	return pins["DOTF_VERSION"], nil
}

// installerInvocation is the command that runs the checkout's installer for
// goos, handing it the checkout so its converge reads the one update just
// moved rather than resolving another. Every argument install.{sh,ps1} gets
// goes on to `dotf converge`.
func installerInvocation(goos, repo string) (name string, args []string) {
	if goos == "windows" {
		return "pwsh", []string{"-NoProfile", "-File", filepath.Join(repo, "install.ps1"), "--repo", repo}
	}
	return filepath.Join(repo, "install.sh"), []string{"--repo", repo}
}

// runInstaller installs the dotf release pin and converges with it, through the
// checkout's own installer: it stages the binary beside the target and renames
// it into place (install.ps1 parks a live dotf.exe first), so replacing the
// binary of this very process is safe. DOTF_VERSION is set explicitly because
// the installer prefers an exported one to the checkout's versions.conf, and a
// shell that sourced a stale deploy copy exports the old pin.
func runInstaller(c *cobra.Command, repo, pin string) error {
	name, args := installerInvocation(runtime.GOOS, repo)
	cmd := exec.Command(name, args...) //nolint:gosec // the checkout's own installer, at a path built from the resolved checkout
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "DOTF_VERSION="+pin)
	// Stdin stays closed, as ExecSetup leaves it: update is the unattended
	// scheduler path, so the identity epilogue of the installer's converge
	// prints what is left instead of walking a restore.
	cmd.Stdout, cmd.Stderr = c.OutOrStdout(), c.ErrOrStderr()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(name), err)
	}
	return nil
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
// stdout without its trailing newline. On failure the error is returned (stdout ignored) — callers
// treat a failed git query as a skip condition, so surfacing stderr is not
// needed here.
func gitRunner(repo string) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output() //nolint:gosec // fixed git subcommands, repo is the resolved checkout
		// Only the trailing newline goes: porcelain output starts with a status
		// column that is a space for an unstaged change, and trimming both ends
		// stripped it from the first line only (#2177).
		return strings.TrimRight(string(out), "\r\n"), err
	}
}
