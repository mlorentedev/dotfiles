package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// newVaultCmd builds `dotf vault`, the knowledge-vault scaffolder (ADR-021 step
// 3). It is the single home for vault-entry scaffolding, with the entry TYPE as
// the dimension: `vault work` scaffolds a work-SDK entry (CLI-015 #388); `vault
// project` scaffolds the personal-project entry under 10_projects/ — the renderer
// extracted from dotf init so init and the standalone command share one core (#395).
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
  work <family> <component>   scaffold a work-SDK entry under 50_work/45-development/
  project [path]              scaffold a personal-project entry under 10_projects/`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newVaultWorkCmd())
	cmd.AddCommand(newVaultProjectCmd())
	return cmd
}

// newVaultProjectCmd builds `dotf vault project [path]`: scaffold the personal-
// project vault entry under 10_projects/<repo>/ (context.md + roadmap.md +
// memory/MEMORY.md), the same renderer dotf init drives — here as a standalone,
// vault-only command for an existing repo. <path> defaults to the current dir;
// the entry name is its basename.
func newVaultProjectCmd() *cobra.Command {
	var (
		stack string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "project [path]",
		Short: "Scaffold a personal-project vault entry (10_projects)",
		Long: `project scaffolds the personal-project vault entry under 10_projects/<repo>/:
context.md, roadmap.md, and memory/MEMORY.md (with {{repo}}/{{stack}}/{{date}}
substituted). It is the standalone twin of the entry dotf init writes — same
embedded templates, same renderer (cli/internal/vault).

<path> defaults to the current directory; the entry name is its basename. On
non-Windows the Claude auto-memory dir is linked to the entry's memory/ (the
session hook re-creates it if absent).

Skip-if-present: a re-run never clobbers an entry that may hold real content.
Pass --force to regenerate the entry files (the memory symlink is left as-is).`,
		Example:      "  dotf vault project\n  dotf vault project ../my-repo --stack go\n  dotf vault project . --force",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := vault.ResolveVaultStrict()
			if err != nil {
				return err
			}

			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			root, err := filepath.Abs(target)
			if err != nil {
				return err
			}

			claudeDir := ""
			if home, herr := os.UserHomeDir(); herr == nil {
				claudeDir = filepath.Join(home, ".claude", "projects")
			}

			res, err := vault.WriteProjectEntry(vault.ProjectEntryOptions{
				VaultPath:         vaultPath,
				RepoRoot:          root,
				Stack:             stack,
				Date:              now().Format("2006-01-02"),
				ClaudeProjectsDir: claudeDir,
				Force:             force,
			})
			if err != nil {
				return err
			}

			cmd.Printf("[OK] Project vault entry: %s\n", res.EntryDir)
			for _, f := range res.Created {
				cmd.Printf("  created  %s\n", f)
			}
			for _, f := range res.Skipped {
				cmd.Printf("  skipped  %s (present; --force to regenerate)\n", f)
			}
			if res.Symlink != "" {
				cmd.Printf("  symlink  %s\n", res.Symlink)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&stack, "stack", "none", "project stack token for the entry: go|python|node|none")
	cmd.Flags().BoolVar(&force, "force", false, "regenerate entry files even if present")
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
50_work/45-development/<family>/<component>/: context.md (with a source_path
placeholder pointing at the real repo), memory/MEMORY.md, and the parent
<family>/context.md (created only when absent — it carries the family's repo
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
			cmd.Printf("  family   [%s] %s/context.md\n", res.Family, args[0])
			cmd.Printf("Next: fill source_path in %s/context.md with the real repo path.\n", res.EntryDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "regenerate component files even if present")
	return cmd
}
