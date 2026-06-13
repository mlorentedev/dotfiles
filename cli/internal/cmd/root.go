// Package cmd wires the dot CLI (Cobra). One <verb>.go per subcommand;
// domain logic lives in sibling internal packages, never here.
package cmd

import (
	"github.com/spf13/cobra"
)

// New builds the root command. The version is injected by cmd/dot/main.go,
// where goreleaser's -X main.version ldflags lands.
func New(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "dot",
		Short: "Single entry point for the dotfiles tooling",
		Long: "dot is the cross-platform dotfiles tooling CLI (ADR-020).\n" +
			"Shell script twins under scripts/ converge here one subcommand at a time.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newReviewCmd())
	root.AddCommand(newSpecCmd())
	return root
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dot version",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("dot version %s\n", version)
		},
	}
}
