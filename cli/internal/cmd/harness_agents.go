package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessAgentsCmd deploys the committed agent records: the Go form of
// compile-harness.sh's deploy_agents, which moves onto it once a release
// carries it (CLI-026).
func newHarnessAgentsCmd() *cobra.Command {
	var (
		repoRoot, home string
		dryRun         bool
	)
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Render each committed agent record to every agents.deploy target",
		Long: "agents renders every harness/agents/<name>/AGENT.md record to each\n" +
			"harness/manifest.json agents.deploy[] target under $HOME as <dir>/<name>.md: the\n" +
			"record's name and description, provenance, and the model and capability lines\n" +
			"resolved for that harness from harness/model-map.json and\n" +
			"harness/capability-map.json. A record's targets: list is matched by exact\n" +
			"harness name. An output that already holds its render is not rewritten. A\n" +
			"record whose tier or capabilities cannot be resolved fails on its own, keeps\n" +
			"its previous output, and makes the command exit non-zero after the rest deploy.\n" +
			"The persona roster is `dotf harness presence`, not this.",
		Example:      "  dotf harness agents --dry-run\n  dotf harness agents --repo-root /path/to/checkout",
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
			outcomes, err := harness.DeployAgents(repoRoot, home, dryRun)
			reportAgentOutcomes(cmd.OutOrStdout(), cmd.ErrOrStderr(), outcomes, dryRun)
			return err
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's dirs are relative to (default: the user's home)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be deployed and write nothing")
	return cmd
}

// reportAgentOutcomes prints one line per output written, the capabilities a
// harness dropped, and a count, so a converged run reads as "0 deployed".
func reportAgentOutcomes(w, ew io.Writer, outcomes []harness.SkillOutcome, dryRun bool) {
	verb := map[bool]string{false: "", true: "would be "}[dryRun]
	counts := map[string]int{}
	for _, o := range outcomes {
		counts[o.Status]++
		if o.Status == harness.SkillDeployed {
			_, _ = fmt.Fprintf(w, "[deploy] agent %s-> %s\n", verb, o.Path)
		}
		if o.Reason != "" {
			_, _ = fmt.Fprintf(ew, "[capabilities] %s: %s\n", o.Path, o.Reason)
		}
	}
	_, _ = fmt.Fprintf(w, "[deploy] agents: %d %sdeployed, %d current\n", counts[harness.SkillDeployed], verb, counts[harness.SkillCurrent])
}
