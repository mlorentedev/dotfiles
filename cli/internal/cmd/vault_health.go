package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
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
// The vault directory resolves $VAULT_DIR first — the shell's own internal
// handoff variable, and what every golden case in tests/golden/vault-health/
// sets — then falls through to the ADR-025 cascade (env.ResolvePath, which
// already covers $VAULT_PATH, the machine.json override, and the env-contract
// default). The shell twin only ever falls back to a literal
// $HOME/Projects/knowledge, oblivious to machine.json — but no golden exercises
// that fallback (VAULT_DIR is always set), and `dotf vault health` is a NEW,
// directly-invokable entry point unlike the shell's, so a relocated vault
// should resolve the same way it does for every other `dotf vault` subcommand
// (vault.ResolveVault) rather than reproducing a gap the oracle only had
// because nothing called it standalone before.
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
			code, err := vault.RunHealth(c.OutOrStdout(), healthOptions(vaultName, verbose))
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

// healthOptions builds the HealthOptions both `vault health` and `vault
// maintain` need. Extracted rather than duplicated when increment 3 landed: the
// $VAULT_DIR-then-ADR-025 cascade documented above is a contract, and two
// copies of a resolution cascade drift in exactly the way ADR-020 §5 is about.
func healthOptions(vaultName string, verbose bool) vault.HealthOptions {
	name := vaultName
	if name == "" {
		name = os.Getenv("VAULT_NAME")
	}
	if name == "" {
		name = "knowledge"
	}

	dir := os.Getenv("VAULT_DIR")
	if dir == "" {
		dir = env.ResolvePath("VAULT_PATH")
	}
	if dir == "" {
		if home, herr := os.UserHomeDir(); herr == nil {
			dir = home + "/Projects/knowledge"
		}
	}

	return vault.HealthOptions{
		VaultDir:   dir,
		VaultName:  name,
		Verbose:    verbose,
		ScriptsDir: memScriptsDir(),
		BashPath:   mem.ResolveBash(),
	}
}
