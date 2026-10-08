package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// refreshPaths are the files `compile-harness.sh --refresh` rewrites in the
// checkout: the harness records and the two generated blocks (ADR-013).
var refreshPaths = []string{"harness/", "AGENTS.md", "ai/claude/CLAUDE.md"}

// newHarnessRefreshCmd re-renders the committed harness records from the vault,
// but only from a vault that is level with its upstream (#2162). The refresh
// reads whatever the local clone holds, so a clone one commit behind rewrote
// merged records to their older state, and the drift report below then asked
// for that revert to be committed. It replaces the block setup-linux.sh ran,
// which had no way to tell a vault that is ahead from one that is behind.
func newHarnessRefreshCmd() *cobra.Command {
	var repo, vault string
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Re-render the harness records from the vault, once the vault is level with its upstream",
		Long: "refresh fast-forwards the vault to its upstream and then runs\n" +
			"scripts/compile-harness.sh --refresh against it, which regenerates the\n" +
			"committed harness records and the generated blocks in AGENTS.md and\n" +
			"ai/claude/CLAUDE.md. Any change that leaves in the checkout is listed, with\n" +
			"the commit that records it.\n\n" +
			"A vault that cannot be confirmed current (diverged, dirty, offline, no\n" +
			"upstream) is not refreshed from: the committed records are kept, and the\n" +
			"reason is printed. A vault that is ahead of its upstream is refreshed from,\n" +
			"since its extra commits are the newer state. Neither case is an error; a\n" +
			"failed compile-harness run is.\n\n" +
			"Without a vault (no <vault>/00_meta/patterns) there is nothing to refresh\n" +
			"from and the committed records stand.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot := repo
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo, set DOTFILES_REPO_DIR, or run from inside it")
			}
			vaultDir := vault
			if vaultDir == "" {
				vaultDir = env.ResolvePath("VAULT_PATH")
			}
			return runHarnessRefresh(cmd.OutOrStdout(), cmd.ErrOrStderr(), repoRoot, vaultDir)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout whose records are refreshed")
	cmd.Flags().StringVar(&vault, "vault", "", "vault to refresh from (default: $VAULT_PATH, then machine.json, then the env-contract default)")
	return cmd
}

func runHarnessRefresh(stdout, stderr io.Writer, repo, vault string) error {
	if vault == "" || !dirExists(filepath.Join(vault, "00_meta", "patterns")) {
		_, _ = fmt.Fprintf(stdout, "harness refresh: no vault at %q; the committed harness records stand\n", vault)
		return nil
	}
	sync := update.Sync(vault, gitRunner(vault))
	switch sync.Status {
	case update.StatusCurrent, update.StatusAhead, update.StatusFastForwarded:
		_, _ = fmt.Fprintf(stdout, "harness refresh: vault %s\n", sync.Message)
	default:
		_, _ = fmt.Fprintf(stderr, "WARNING: harness refresh skipped, the vault is not confirmed current: %s\n", sync.Message)
		for _, line := range strings.Split(sync.Detail, "\n") {
			if line != "" {
				_, _ = fmt.Fprintf(stderr, "      %s\n", line)
			}
		}
		_, _ = fmt.Fprintln(stderr, "  The committed harness records stand: refreshing from a stale vault would revert merged records.")
		return nil
	}

	c := exec.Command(filepath.Join(repo, "scripts", "compile-harness.sh"), "--refresh") //nolint:gosec // the checkout's own script
	c.Dir = repo
	// The script reads $VAULT_PATH. Pin it to the clone just checked, so the
	// refresh cannot read a different one than the check vouched for.
	c.Env = append(os.Environ(), "VAULT_PATH="+vault)
	if out, err := c.CombinedOutput(); err != nil {
		_, _ = stderr.Write(out)
		return fmt.Errorf("compile-harness.sh --refresh failed: %w; the committed harness records stand", err)
	}
	_, _ = fmt.Fprintln(stdout, "harness refresh: override blocks and records re-rendered from the vault")
	return reportRefreshDrift(stdout, stderr, repo)
}

// reportRefreshDrift lists what the refresh changed in the checkout. Those
// changes are the vault's newer state and belong in a commit, so they are
// announced loudly rather than left to read as a parallel session's work
// (#295). The deploy mirror is not a repo (ADR-005), so there it says nothing.
func reportRefreshDrift(stdout, stderr io.Writer, repo string) error {
	git := gitRunner(repo)
	if _, err := git("rev-parse", "--git-dir"); err != nil {
		return nil
	}
	drift, err := git(append([]string{"status", "--porcelain", "--"}, refreshPaths...)...)
	if err != nil {
		return fmt.Errorf("cannot read the refresh's changes in %s: %w", repo, err)
	}
	if drift == "" {
		return nil
	}
	_, _ = fmt.Fprintln(stderr, "WARNING: Harness records changed by --refresh from the vault -- commit them:")
	for _, line := range strings.Split(drift, "\n") {
		_, _ = fmt.Fprintf(stderr, "      %s\n", line)
	}
	_, _ = fmt.Fprintf(stderr, "  git add %s && git commit -m 'chore(harness): refresh records from vault'\n",
		strings.Join(refreshPaths, " "))
	return nil
}
