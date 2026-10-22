package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessDeployCmd runs every harness surface in the order
// compile-harness.sh --deploy runs them, which it replaces once a release
// carries it (CLI-026).
func newHarnessDeployCmd() *cobra.Command {
	var (
		repoRoot, home string
		dryRun         bool
	)
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy every harness surface: instructions, skills, agents, presence, doctrine",
		Long: "deploy runs, in order, what `dotf harness instructions`, `skills`, `agents`,\n" +
			"`presence` and `doctrine` each do. The instruction files go first, because the\n" +
			"skill catalog and the presence roster are regions inside them. A failed\n" +
			"surface does not stop the ones after it, except agents: when an agent record\n" +
			"fails, presence and doctrine are not written, since both carry the roster those\n" +
			"records feed. Any failure makes the command exit non-zero at the end.\n" +
			"--dry-run previews every surface but presence, which has no preview.",
		Example:      "  dotf harness deploy\n  dotf harness deploy --dry-run",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo-root, set DOTFILES_REPO_DIR, or run from inside it")
			}
			if home == "" {
				home = env.Home()
			}
			return deployHarness(cmd, repoRoot, home, dryRun)
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's paths are relative to (default: the user's home)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be deployed and write nothing")
	return cmd
}

func deployHarness(cmd *cobra.Command, repoRoot, home string, dryRun bool) error {
	w, ew := cmd.OutOrStdout(), cmd.ErrOrStderr()
	has := func(name string) bool { _, err := exec.LookPath(name); return err == nil }
	d, err := harness.Deploy(repoRoot, home, has, dryRun)

	reportInstructionOutcomes(w, d.Instructions, dryRun)
	reportSkillOutcomes(w, ew, d.Skills, dryRun)
	reportAgentOutcomes(w, ew, d.Agents, dryRun)
	if d.AgentsErr != nil {
		_, _ = fmt.Fprintln(ew, "[deploy] presence and doctrine NOT written: the agents surface failed; its error follows")
		return err
	}
	if dryRun {
		_, _ = fmt.Fprintln(w, "[deploy] presence has no preview; not run under --dry-run")
	} else {
		reportPresenceOutcomes(w, ew, d.Presence)
	}
	reportDoctrineOutcomes(w, ew, d.Doctrine, dryRun)

	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(w, "[deploy] OK")
	return nil
}
