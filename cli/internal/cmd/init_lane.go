package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// resolveInitRepo is the target of `dotf init agents|lane --repo`: the named
// directory, or the current git repo when none is named.
func resolveInitRepo(repo string) (string, error) {
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return spec.RepoRoot(cwd)
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return "", fmt.Errorf("--repo is not a directory: %s", repo)
	}
	return repo, nil
}

// newInitLaneCmd builds `dotf init lane`: scaffold the lane contract into a
// repository, from the templates embedded in the binary.
func newInitLaneCmd() *cobra.Command {
	var (
		repo   string
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "lane",
		Short: "Scaffold the lane contract: .dotf/lane.json, the lane prompt and the AGENTS.md Lanes section",
		Long: `lane writes the lane contract into the target repo: an empty-queue
.dotf/lane.json, the shared .dotf/lane-prompt.md and a "## Lanes" section in
AGENTS.md, from templates embedded in the binary.

Idempotent: existing files are left alone and a re-run reports them unchanged.
--force replaces the prompt and the AGENTS.md section in place; it never
touches .dotf/lane.json, whose queue belongs to the repository. Without --repo
it operates on the current git repo.`,
		Example:      "  dotf init lane\n  dotf init lane --repo ../other-repo --dry-run\n  dotf init lane --force",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := resolveInitRepo(repo)
			if err != nil {
				return err
			}
			res, err := initrepo.BootstrapLane(root, force, dryRun)
			if err != nil {
				return err
			}
			prefix := "[OK]"
			if dryRun {
				prefix = "[DRY-RUN]"
			}
			cmd.Printf("%s %s: %s\n", prefix, root, res)
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "target repo root (default: the current git repo)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report intended changes without modifying disk")
	cmd.Flags().BoolVar(&force, "force", false, "replace the lane prompt and the AGENTS.md section in place")
	return cmd
}
