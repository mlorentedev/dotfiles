package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessResolveTierCmd is model-map.json's first compile-time consumer.
//
// ADR-035 splits consumption of one file into two cadences: `tiers` resolves at
// compile time and feeds each harness's render, `chains` resolves at run time
// and is read only by a level-2 dispatcher. This command is the whole of the
// first half, and deliberately exposes no way to reach the second — a
// compile-time caller that can accidentally read a run-time field is exactly the
// drift the `version` key exists to survive.
//
// The body is thin on purpose. Resolution lives in harness.ResolveTier and the
// contract lives in model-map.schema.json; duplicating either here would create
// a second place for the routing rules to be true, which is the failure this
// registry was built to end.
func newHarnessResolveTierCmd() *cobra.Command {
	var (
		harnessName string
		repoRoot    string
	)

	cmd := &cobra.Command{
		Use:   "resolve-tier <tier>",
		Short: "Resolve a neutral model tier to one harness's model id",
		Long: `resolve-tier answers which model id the neutral tier top|mid|low means for
one harness, reading harness/model-map.json through the validated loader.

The model id is written to stdout and nothing else is, so a caller can capture it
directly:

    model_id="$(dotf harness resolve-tier top --harness claude)"

Where the map cannot answer — an undeclared tier, a harness with no entry in it,
or a map that is absent or schema-invalid — this exits non-zero and writes
nothing to stdout. It never falls back to a default and never resolves to an
empty id: a definition naming no model is the silent degrade the routing
registry exists to prevent.`,
		Example: `  dotf harness resolve-tier top --harness claude
  dotf harness resolve-tier mid --harness opencode
  dotf harness resolve-tier top --harness claude --repo-root ~/.dotfiles`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if harnessName == "" {
				return fmt.Errorf("--harness is required: a tier means a different model id per harness, " +
					"so resolving one without naming the harness has no answer")
			}
			root := repoRoot
			if root == "" {
				root = env.ResolveHarnessRoot()
			}
			m, err := harness.LoadModelMap(root)
			if err != nil {
				return err
			}
			// ResolveTier already phrases both failure cases — an undeclared tier
			// and a harness with no entry in one — naming the tier, the harness
			// and the file. Wrapping it would bury the remedy it spells out.
			model, err := harness.ResolveTier(m, args[0], harnessName)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), model)
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "harness the tier is resolved for (required)")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "",
		"root containing harness/model-map.json (default: the checkout, else DOTFILES_DIR)")
	return cmd
}
