// Package cmd wires the dotf CLI (Cobra). One <verb>.go per subcommand;
// domain logic lives in sibling internal packages, never here.
package cmd

import (
	"fmt"

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
	root.AddCommand(newEnvCmd())
	root.AddCommand(newToolsCmd())
	root.AddCommand(newMemCmd())
	root.AddCommand(newSecretsCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newHarnessCmd())
	root.AddCommand(newDeployCmd())
	root.AddCommand(newPrCmd())
	return root
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dotf version",
		Run: func(cmd *cobra.Command, _ []string) {
			// install-dotf.{sh,ps1} grep this for the installed semver, so it
			// must reach stdout — cmd.Printf writes to OutOrStderr().
			// Run (not RunE) — no error to return, so discard explicitly, as
			// secrets.go does. A failed write to stdout here is not actionable.
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dotf version %s\n", version)
		},
	}
}
