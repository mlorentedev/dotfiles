// Package cmd wires the dotf CLI (Cobra). One <verb>.go per subcommand;
// domain logic lives in sibling internal packages, never here.
package cmd

import (
	"github.com/spf13/cobra"
)

// New builds the root command. The version is injected by cmd/dotf/main.go,
// where goreleaser's -X main.version ldflags lands.
func New(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "dotf",
		Short: "Single entry point for the dotfiles tooling",
		Long: "dotf is the cross-platform dotfiles tooling CLI (ADR-020).\n" +
			"Shell script twins under scripts/ converge here one subcommand at a time.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newReviewCmd())
	root.AddCommand(newSpecCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newVaultCmd())
	return root
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dotf version",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("dotf version %s\n", version)
		},
	}
}
