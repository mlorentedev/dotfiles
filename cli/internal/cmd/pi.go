package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/pi"
)

// errPiDrift is returned (silently) by check when live differs from the
// manifest, and by apply when a call failed: non-zero, like every other
// check/apply pair here, both when something is wrong and when it could not
// be made right.
var errPiDrift = errors.New("pi packages: live state differs from the manifest")

// Seams, so tests never run pi or depend on what this machine has installed.
var (
	piRun      pi.Runner = pi.ExecRunner
	piLookPath           = exec.LookPath
)

// skipEnv is CI-002's contract (#1478): a throwaway runner whose diff cannot
// change what the reconcile does sets it, and the reconcile then skips loudly.
const skipEnv = "DOTFILES_SKIP_PI_PACKAGES"

func newPiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pi",
		Short:        "pi agent state declared in this repository",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	packages := &cobra.Command{
		Use:          "packages",
		Short:        "pi packages declared in " + pi.ManifestFile,
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	packages.AddCommand(newPiPackagesCheckCmd(), newPiPackagesApplyCmd())
	cmd.AddCommand(packages)
	return cmd
}

// piTarget is what both subcommands read: the declaration and pi's live state.
type piTarget struct {
	manifest pi.Manifest
	agentDir string
	plan     pi.Plan
}

// loadPiTarget reads the manifest of repo, or of the checkout the cwd is in
// when repo is empty. Setup always passes repo: a cwd inside another checkout
// (a worktree on another branch) would otherwise reconcile pi against THAT
// manifest, removing whatever it does not declare.
func loadPiTarget(repo, agentDir string) (piTarget, error) {
	var t piTarget
	root := repo
	if root == "" {
		root = env.RepoDir()
	}
	if root == "" {
		return t, fmt.Errorf("cannot locate the dotfiles checkout; set DOTFILES_REPO_DIR or run from inside it")
	}
	m, err := pi.LoadManifest(root)
	if err != nil {
		return t, err
	}
	if agentDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return t, err
		}
		agentDir = filepath.Join(home, ".pi", "agent")
	}
	live, err := pi.LiveSources(filepath.Join(agentDir, "settings.json"))
	if err != nil {
		return t, err
	}
	return piTarget{manifest: m, agentDir: agentDir, plan: pi.NewPlan(m, live)}, nil
}

func newPiPackagesCheckCmd() *cobra.Command {
	var agentDir, repo string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report pi packages the manifest declares and pi lacks, and the reverse",
		Long: `Compare ` + pi.ManifestFile + ` with the packages pi records in
~/.pi/agent/settings.json, and list what apply would change: undeclared
packages to remove and declared ones to install. It changes nothing. Exit status is non-zero when anything would change, and
when the manifest or the live settings cannot be read.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			t, err := loadPiTarget(repo, agentDir)
			if err != nil {
				c.PrintErrln("pi packages check:", err)
				return err
			}
			printPiPlan(c.OutOrStdout(), t)
			if !t.plan.Empty() {
				return errPiDrift
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agentDir, "agent-dir", "", "pi's agent dir (default ~/.pi/agent)")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout whose manifest to read (default: the one containing the cwd)")
	return cmd
}

func newPiPackagesApplyCmd() *cobra.Command {
	var agentDir, piBin, repo string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Converge pi's packages on the manifest, both ways",
		Long: `Converge pi on ` + pi.ManifestFile + ` (HARNESS-139): remove each
package the manifest does not declare at any version, install each declared one
that is not live at exactly its source. Every change goes through pi's own
CLI; pi owns its settings file. A second run reports changed=0.

` + skipEnv + ` set skips everything, loudly. A missing pi or npm is a warning
and exit 0, so a setup run degrades instead of breaking.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return runPiApply(c.OutOrStdout(), c.ErrOrStderr(), repo, agentDir, piBin, dryRun)
		},
	}
	cmd.Flags().StringVar(&agentDir, "agent-dir", "", "pi's agent dir (default ~/.pi/agent)")
	cmd.Flags().StringVar(&piBin, "pi", "", "pi binary (default ~/.local/bin/pi, then pi on PATH)")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout whose manifest to apply (default: the one containing the cwd)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would change and change nothing")
	return cmd
}

func runPiApply(out, errOut io.Writer, repo, agentDir, piBin string, dryRun bool) error {
	// First, before any probe: a skipped run must not depend on what the
	// runner has installed.
	if os.Getenv(skipEnv) != "" {
		_, _ = fmt.Fprintln(out, "[WARN] "+skipEnv+" set — pi package reconcile skipped (nothing installed, nothing verified)")
		return nil
	}
	t, err := loadPiTarget(repo, agentDir)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "pi packages apply:", err)
		return err
	}
	if dryRun || t.plan.Empty() {
		printPiPlan(out, t)
		return nil
	}
	bin, ok := resolvePi(piBin)
	if !ok {
		_, _ = fmt.Fprintln(out, "[WARN] pi not installed — skipping pi package reconcile (re-run setup after pi installs)")
		return nil
	}
	if _, err := piLookPath("npm"); err != nil && len(t.plan.Install) > 0 {
		_, _ = fmt.Fprintln(out, "[WARN] npm not found — skipping pi package reconcile (install Node.js, then re-run setup)")
		return nil
	}
	res := pi.Apply(t.plan, pi.Options{PiBin: bin, Log: out}, piRun)
	_, _ = fmt.Fprintf(out, "pi packages: changed=%d (%d removed, %d installed), %d failed\n",
		res.Changed(), res.Removed, res.Installed, res.Failed)
	if res.Failed > 0 {
		return errPiDrift
	}
	return nil
}

// resolvePi finds the pi binary: an explicit path, then the prefix setup
// installs into, then PATH. Never the shell function named pi, which wraps
// `dotf secrets run` and fails on a locked vault; a binary lookup cannot see it.
func resolvePi(explicit string) (string, bool) {
	if explicit != "" {
		_, err := os.Stat(explicit)
		return explicit, err == nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		local := filepath.Join(home, ".local", "bin", "pi")
		if info, err := os.Stat(local); err == nil && !info.IsDir() {
			return local, true
		}
	}
	p, err := piLookPath("pi")
	return p, err == nil
}

func printPiPlan(w io.Writer, t piTarget) {
	if t.plan.Empty() {
		_, _ = fmt.Fprintf(w, "pi packages already reconciled (%d declared, 0 changed)\n", len(t.manifest.Packages))
		return
	}
	for _, s := range t.plan.Remove {
		_, _ = fmt.Fprintf(w, "  remove   %s (not declared)\n", s)
	}
	for _, s := range t.plan.Install {
		_, _ = fmt.Fprintf(w, "  install  %s\n", s)
	}
	_, _ = fmt.Fprintf(w, "pi packages: %d to remove, %d to install\n", len(t.plan.Remove), len(t.plan.Install))
}
