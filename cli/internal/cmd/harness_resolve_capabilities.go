package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessResolveCapabilitiesCmd is capability-map.json's consumer, and the
// sibling of resolve-tier: same seam, one field over.
//
// It resolves a capability SET rather than one capability at a time, because the
// two native forms disagree about what a partial answer means. A `csv` field is
// an allow-list — naming a tool grants it and omitting one denies it — while a
// `decision-map` grants without denying. A per-capability API would push that
// distinction onto every caller, and the shell caller is exactly the one least
// able to carry it.
func newHarnessResolveCapabilitiesCmd() *cobra.Command {
	var (
		harnessName string
		repoRoot    string
	)

	cmd := &cobra.Command{
		Use:   "resolve-capabilities <capability>[,<capability>...]",
		Short: "Resolve neutral capabilities to one harness's native frontmatter line",
		Long: `resolve-capabilities answers which native tools or permissions a set of neutral
capabilities means for one harness, reading harness/capability-map.json through
the validated loader.

It prints a COMPLETE frontmatter line — field and value — because the field name
differs per harness and a caller should not have to know it:

    cap_line="$(dotf harness resolve-capabilities read,shell --harness claude)"
    # tools: Read, Glob, Bash

Both declared forms render on one line and both are valid YAML: ` + "`csv`" + ` produces
a comma list, ` + "`decision-map`" + ` a flow mapping.

Where the map cannot answer — an unmapped capability, an undeclared harness, or a
map that is absent or schema-invalid — this exits non-zero and writes nothing to
stdout. It never falls back to a default: an empty tools list is not "no opinion",
it is a definition granting nothing.`,
		Example: `  dotf harness resolve-capabilities read,search,edit,shell --harness claude
  dotf harness resolve-capabilities read,web --harness opencode`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if harnessName == "" {
				return fmt.Errorf("--harness is required: a capability means different native tools " +
					"per harness, so resolving one without naming the harness has no answer")
			}
			root := repoRoot
			if root == "" {
				root = env.ResolveHarnessRoot()
			}
			m, err := harness.LoadCapabilityMap(root)
			if err != nil {
				return err
			}
			line, err := harness.ResolveCapabilities(m, strings.Split(args[0], ","), harnessName)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "harness the capabilities are resolved for (required)")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "",
		"root containing harness/capability-map.json (default: the checkout, else DOTFILES_DIR)")
	return cmd
}
