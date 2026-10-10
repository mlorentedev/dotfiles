package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessMirrorCmd mirrors the harness inputs from the checkout into the
// deploy dir (WIN-007/#1288). One implementation for both OSes: it replaces
// the bash+jq block setup-linux.sh carried (with its zsh word-split hazard and
// its "jq not on PATH yet" race) and the block setup-windows.ps1 never had —
// which is why `dotf doctor` on Windows failed the routing registry and the
// model-pin checks after every setup, printing "re-run setup" as the remedy.
func newHarnessMirrorCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror harness/ and every manifest target from the checkout into the deploy dir",
		Long: "mirror copies the harness inputs the deploy-dir consumers read — the whole\n" +
			"harness/ tree and every file harness/manifest.json declares as an injection\n" +
			"target — from the dotfiles checkout into $DOTFILES_DIR, preserving paths.\n\n" +
			"Idempotent: a file whose bytes and permission bits already match is left\n" +
			"untouched, so a converged re-run reports 0 updated and 0 pruned.\n\n" +
			"In .zsh/, ssh/ and scripts/, which hold only checkout copies, it removes a\n" +
			"file the checkout's git history deleted. A file git never tracked there is\n" +
			"left in place and named. harness/ and the secrets are never pruned here;\n" +
			"`dotf doctor --fix` owns their orphans.\n\n" +
			"A declared target the checkout does not have is named and the command exits 1\n" +
			"after mirroring everything else — the gap is the finding, not a reason to\n" +
			"leave the rest of the harness stale.",
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
			deployDir := env.DotfilesDir(env.Home())

			res, err := harness.Mirror(repoRoot, deployDir)
			switch {
			case errors.Is(err, harness.ErrCheckoutIsDeployDir):
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "harness mirror: %s is the checkout; nothing to mirror\n", deployDir)
				return nil
			case errors.Is(err, harness.ErrMissingTargets):
				for _, rel := range res.Missing {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"harness/manifest.json declares a target the checkout does not have: %s — not mirrored; dotf doctor will report harness drift\n", rel)
				}
			case err != nil:
				return err
			}
			reportPrune(cmd, res)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "harness mirror: harness/ + %d target(s) → %s (%d updated, %d unchanged, %d pruned)\n",
				len(res.Targets), deployDir, res.Updated, res.Unchanged, len(res.Pruned))
			return err
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout to mirror from")
	return cmd
}

// reportPrune names every leftover the mirror removed and every orphan it had
// to leave, so a prune is never silent and an unproven orphan is never hidden.
func reportPrune(cmd *cobra.Command, res harness.MirrorResult) {
	for _, rel := range res.Pruned {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "harness mirror: pruned %s (deleted from the checkout)\n", rel)
	}
	if len(res.Unpruned) == 0 {
		return
	}
	if res.PruneSkipped != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "harness mirror: %s\n", res.PruneSkipped)
	}
	for _, rel := range res.Unpruned {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"harness mirror: left %s in the deploy dir: the checkout does not have it and its history does not show it deleted\n", rel)
	}
}
