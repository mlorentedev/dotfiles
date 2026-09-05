// Package cmd wires the dotf CLI (Cobra). One <verb>.go per subcommand;
// domain logic lives in sibling internal packages, never here.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// New builds the root command. The version and commit are injected by
// cmd/dotf/main.go, where goreleaser's -X main.version / -X main.commit ldflags
// land. commit is "" for a source build.
func New(version, commit string) *cobra.Command {
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

	root.AddCommand(newVersionCmd(version, commit))
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
	root.AddCommand(newAgentCmd())
	root.AddCommand(newDeployCmd())
	root.AddCommand(newOrcaCmd())
	root.AddCommand(newPrCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newHooksCmd())
	root.AddCommand(newWorktreeCmd())
	return root
}

func newVersionCmd(version, commit string) *cobra.Command {
	var commitOnly bool

	c := &cobra.Command{
		Use:   "version",
		Short: "Print the dotf version",
		Run: func(cmd *cobra.Command, _ []string) {
			// install-dotf.{sh,ps1} grep this for the installed semver, so it
			// must reach stdout — cmd.Printf writes to OutOrStderr().
			// Run (not RunE) — no error to return, so discard explicitly, as
			// secrets.go does. A failed write to stdout here is not actionable.
			//
			// The DEFAULT output stays exactly one line, byte-identical to what
			// it has always been. Both installers regex the merged streams for
			// `(\d+\.\d+\.\d+|dev)` against whatever dotf is already on PATH, and
			// their idempotence skip — the thing that decides whether a machine
			// reinstalls on every setup run — hangs off that match. The commit
			// goes behind a flag rather than onto a second line for that reason.
			if commitOnly {
				// Empty (source build) prints an empty line rather than a
				// placeholder: the caller is a machine, and "" is the value.
				// `dotf doctor` distinguishes empty-stamp from absent-flag, and
				// a word like "unknown" here would collide with a real hash.
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), commit)
				return
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dotf version %s\n", version)
		},
	}

	// A binary built before this flag existed answers `unknown flag: --commit`
	// on stderr and exits non-zero. That is not an error to paper over — it is
	// the provenance answer for a pre-stamp binary, and doctor reads it as such.
	c.Flags().BoolVar(&commitOnly, "commit", false,
		"print only the commit this binary was built from (empty for a source build)")

	return c
}
