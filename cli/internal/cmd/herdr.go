package cmd

import (
	"github.com/mlorentedev/dotfiles/cli/internal/herdr"
	"github.com/spf13/cobra"
)

// newHerdrCmd is wiring only: the behaviour lives in internal/herdr, next to the
// rest of this CLI's herdr knowledge (ADR-020, and the package already there).
func newHerdrCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "herdr",
		Short: "Glue between dotf and the herdr terminal workspace manager",
		Long: "herdr provides the parts of a herdr session that herdr's own config cannot\n" +
			"express: commands that reshape a tab.\n\n" +
			"  dotf herdr explorer   # toggle a yazi file explorer docked to the right\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(newHerdrExplorerCmd())
	return c
}

func newHerdrExplorerCmd() *cobra.Command {
	var ratio float64
	c := &cobra.Command{
		Use:   "explorer",
		Short: "Toggle a yazi file explorer docked to the right of this herdr pane",
		Long: "explorer opens this tab's worktree as a real split on the right, running\n" +
			"yazi in this pane's directory, and closes it again on the next call: a\n" +
			"toggle, so a second press does not stack a second explorer. The pane it\n" +
			"creates is labelled, which is how it is recognised later: a yazi you opened\n" +
			"yourself is never closed.\n\n" +
			"yazi is not installed here: YAZI_VERSION in versions.conf pins it and\n" +
			"`dotf tools sync` provisions it on every OS. The command refuses rather\n" +
			"than opening an empty pane when it is missing.\n\n" +
			"Inside a herdr pane it reads HERDR_ACTIVE_* (a custom command) or\n" +
			"HERDR_PANE_ID (the pane's own environment), so it works both from a herdr\n" +
			"keybinding and typed into the pane.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := herdr.ExplorerContext()
			if err != nil {
				return err
			}
			if err := herdr.ValidateExplorerRatio(ratio); err != nil {
				return err
			}
			line, err := herdr.ToggleExplorer(herdrRunner, ctx, ratio)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(line + "\n"))
			return err
		},
	}
	c.Flags().Float64Var(&ratio, "ratio", herdr.ExplorerRatioDefault, "fraction of the tab the explorer takes, as herdr's split ratio")
	return c
}
