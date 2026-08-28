package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// errVaultHealthFailed is returned (silently — SilenceErrors keeps the report
// itself the only output) when RunHealth reports one or more failed checks.
// Mirrors doctor.go's errChecksFailed for the same reason.
var errVaultHealthFailed = errors.New("vault health: one or more checks failed")

// newVaultHealthCmd builds `dotf vault health`, the Go port of
// scripts/vault-health.sh (CLI-021 / #490, increment 2). Built BESIDE the shell
// twin: nothing repoints at this yet (that cutover is CLI-023 / #492).
//
// vaultDirDefault mirrors the shell's OWN cascade — $VAULT_DIR, then
// $VAULT_PATH, then a literal ~/Projects/knowledge — deliberately NOT the
// ADR-025 machine.json cascade `vault.ResolveVault()` uses: the shell predates
// that cascade, and the golden corpus (tests/golden/vault-health/) pins this
// exact fallback, so matching the oracle wins over "improving" it here.
func newVaultHealthCmd() *cobra.Command {
	var (
		verbose   bool
		vaultName string
	)

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Report knowledge-vault health (Obsidian orphans, links, frontmatter, backlog)",
		Long: `health runs a 7-section report against the knowledge vault:

  1. Working tree integrity (files deleted from disk but still tracked)
  2. Obsidian CLI connectivity
  3. Orphans & dead-ends
  4. Unresolved links
  5. Frontmatter coverage (id/type/status/tags/created/owner)
  6. Tag hygiene
  7. Backlog integrity (duplicate/contradictory ticket entries, stale-merged ticks)

Requires the Obsidian GUI running (the CLI talks to it over IPC); every
GUI-dependent section degrades gracefully when it is not. Read-only: no flag
writes anything.

Exit: 0 all checks pass, 1 one or more failed (or the Obsidian CLI itself is
missing from PATH), 2 the GUI is unreachable.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			name := vaultName
			if name == "" {
				name = os.Getenv("VAULT_NAME")
			}
			if name == "" {
				name = "knowledge"
			}

			dir := os.Getenv("VAULT_DIR")
			if dir == "" {
				dir = os.Getenv("VAULT_PATH")
			}
			if dir == "" {
				if home, err := os.UserHomeDir(); err == nil {
					dir = home + "/Projects/knowledge"
				}
			}

			code, err := vault.RunHealth(c.OutOrStdout(), vault.HealthOptions{
				VaultDir:   dir,
				VaultName:  name,
				Verbose:    verbose,
				ScriptsDir: memScriptsDir(),
				BashPath:   mem.ResolveBash(),
			})
			if err != nil {
				return err
			}
			if code != 0 {
				return withExitCode(code, errVaultHealthFailed)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "list orphan/unresolved/tag detail (truncated)")
	cmd.Flags().StringVar(&vaultName, "vault", "", "Obsidian vault name (default: $VAULT_NAME or \"knowledge\")")
	return cmd
}
