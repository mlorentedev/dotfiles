package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// newVaultCmd builds `dotf vault`, the knowledge-vault scaffolder (ADR-021 step
// 3). It is the single home for vault-entry scaffolding, with the entry TYPE as
// the dimension: `vault work` scaffolds a work-SDK entry (this cut, CLI-015 #388);
// `vault project` (the 10_projects/ entry extracted from dotf init) lands in #395.
//
// The parent is runnable via RunE: cmd.Help so it is a first-class "Available
// Command" rather than a cobra help-topic (the demotion a non-runnable, childless
// parent suffers — see the CLI-014 lessons).
func newVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Scaffold knowledge-vault entries (ADR-021 step 3)",
		Long: `vault scaffolds entries in the knowledge vault from templates embedded in the
binary (drift-tested against the vault SSOT, like dotf spec / dotf init).

Unlike dotf init, vault is a vault-only command: a missing vault is an error
(set $VAULT_PATH or create ~/Projects/knowledge), not a silent skip.

Subcommands:
  work <family> <component>   scaffold a work-SDK entry under 50_work/45-development/`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newVaultWorkCmd())
	return cmd
}

// newVaultWorkCmd builds `dotf vault work <family> <component>`: scaffold the
// work-SDK vault entry under 50_work/45-development/<family>/<component>/. The Go
// home of the capability removed from init-project.sh --work-sdk in CLI-014.
func newVaultWorkCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "work <family> <component>",
		Short: "Scaffold a work-SDK vault entry (50_work/45-development)",
		Long: `work scaffolds a vault-only entry for a work-SDK component under
50_work/45-development/<family>/<component>/: 00-context.md (with a source_path
placeholder pointing at the real repo), memory/MEMORY.md, and the parent
<family>/00-context.md (created only when absent — it carries the family's repo
table across components).

Skip-if-present: a re-run never clobbers an entry that may have accumulated real
content. Pass --force to regenerate the component files (the family context is
never overwritten).`,
		Example:      "  dotf vault work acme-sensors edge-fw\n  dotf vault work acme-sensors edge-fw --force",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := vault.ResolveVaultStrict()
			if err != nil {
				return err
			}

			res, err := vault.WriteWorkEntry(vault.WorkEntryOptions{
				VaultPath: vaultPath,
				Family:    args[0],
				Component: args[1],
				Date:      now().Format("2006-01-02"),
				Force:     force,
			})
			if err != nil {
				return err
			}

			cmd.Printf("[OK] Work-SDK vault entry: %s\n", res.EntryDir)
			for _, f := range res.Created {
				cmd.Printf("  created  %s\n", f)
			}
			for _, f := range res.Skipped {
				cmd.Printf("  skipped  %s (present; --force to regenerate)\n", f)
			}
			cmd.Printf("  family   [%s] %s/00-context.md\n", res.Family, args[0])
			cmd.Printf("Next: fill source_path in %s/00-context.md with the real repo path.\n", res.EntryDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "regenerate component files even if present")
	return cmd
}
