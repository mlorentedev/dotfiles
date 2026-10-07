package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessInstructionsCmd deploys the agents' instruction files: the Go form
// of compile-harness.sh's deploy_instructions, which delegates here when the
// installed dotf carries this subcommand (PLAT-001b PR 2c).
func newHarnessInstructionsCmd() *cobra.Command {
	var (
		repoRoot, home string
		dryRun         bool
	)
	cmd := &cobra.Command{
		Use:   "instructions",
		Short: "Deploy each agents.presence instruction file from its source",
		Long: "instructions writes every harness/manifest.json agents.presence[] file under\n" +
			"$HOME from its source: the source verbatim, followed by the deploy-only regions\n" +
			"(the persona presence roster, the copilot skill catalog) the deployed file\n" +
			"already carries, so the steps that own them replace them in place. A file that\n" +
			"already holds exactly that is not rewritten. A target whose requires_command is\n" +
			"not on PATH is skipped; a symlink at the destination becomes a copy. A missing\n" +
			"source is an error, reported after the other targets are deployed.",
		Example:      "  dotf harness instructions --dry-run\n  dotf harness instructions --repo-root /path/to/checkout",
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
			has := func(name string) bool { _, err := exec.LookPath(name); return err == nil }
			outcomes, err := harness.DeployInstructions(repoRoot, home, has, dryRun)
			w := cmd.OutOrStdout()
			for _, o := range outcomes {
				switch {
				case o.Status == "skipped":
					_, _ = fmt.Fprintf(w, "[deploy] instructions target %s skipped: %s\n", o.Agent, o.Reason)
				case o.Status == "current":
					_, _ = fmt.Fprintf(w, "[deploy] instructions current: %s\n", o.File)
				case dryRun:
					_, _ = fmt.Fprintf(w, "[deploy] instructions would deploy -> %s\n", o.File)
				default:
					_, _ = fmt.Fprintf(w, "[deploy] instructions -> %s\n", o.File)
				}
			}
			return err
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's files are relative to (default: the user's home)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be deployed and write nothing")
	return cmd
}
