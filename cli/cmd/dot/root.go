package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is overridden at release time by goreleaser via
// -ldflags "-X main.version=<tag>".
var version = "dev"

func newRootCmd() *cobra.Command {
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

	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dot version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "dot version %s\n", version)
		},
	}
}
