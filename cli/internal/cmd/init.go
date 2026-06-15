package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// newInitCmd builds `dotf init`, the repo-scaffolder (ADR-022) — the Go flagship
// that absorbs the init-project + init-repo-{agents,github} shell twins. It
// scaffolds a fully-practiced repo from templates embedded in the binary
// (cli/internal/initrepo), so it works on a machine with no vault checked out.
//
// This is the foundation cut (CLI-014 Step 1): the command namespace plus the
// embedded-template drift guard. The `agents`/`github` subcommands and the
// default orchestrator land in the following steps; until then `dotf init`
// prints its help.
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a fully-practiced repo from embedded templates (ADR-022)",
		Long: `init scaffolds a new repo with the cross-project practice stack baked in:
AGENTS.md + the Spec-Driven Development section, a CLAUDE.md pointer, guardrail
CI, pre-commit (gitleaks), .gitignore, stack init, git init, the env-contract,
and (when present) the vault entry and GitHub repo defaults.

Templates are embedded in the binary and drift-tested against the vault SSOT, so
init is self-contained: it works on a machine with no vault and no ~/.claude, and
the generated AGENTS.md never leaks an unexpanded $VAULT_PATH.

Re-runnable subcommands ('init agents', 'init github') and the default
orchestrator are being built out under CLI-014.`,
		// Until the orchestrator lands (Step 4), bare `dotf init` prints its
		// help — the same idiom the root command uses. This also keeps init a
		// first-class entry under "Available Commands" rather than a help topic.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newInitAgentsCmd())
	return cmd
}

// newInitAgentsCmd builds `dotf init agents`: seed or refresh the target repo's
// AGENTS.md + Spec-Driven Development section from the embedded, self-contained
// template. The Go twin of scripts/init-repo-agents.sh, and the standalone
// target for the fleet-wide AGENTS.md backfill (HARNESS-013).
func newInitAgentsCmd() *cobra.Command {
	var (
		repo  string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Bootstrap or refresh AGENTS.md + the Spec-Driven Development section",
		Long: `agents seeds (or refreshes) the target repo's AGENTS.md with the self-contained
Spec-Driven Development section, from the template embedded in the binary — no
vault required and no $VAULT_PATH leak (#248).

Idempotent: a re-run is a safe no-op when the section is already present. Pass
--force to replace an existing section in place. Without --repo it operates on
the current git repo.`,
		Example:      "  dotf init agents\n  dotf init agents --repo ../other-repo --force",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := repo
			if root == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				if root, err = spec.RepoRoot(cwd); err != nil {
					return err
				}
			} else if info, err := os.Stat(root); err != nil || !info.IsDir() {
				return fmt.Errorf("--repo is not a directory: %s", root)
			}

			res, err := initrepo.BootstrapAgents(root, force)
			if err != nil {
				return err
			}
			switch res.Action {
			case "unchanged":
				cmd.Printf("[OK] SDD section already present in %s (use --force to refresh)\n", res.Path)
			case "created":
				cmd.Printf("[OK] Created %s with the Spec-Driven Development section\n", res.Path)
			case "appended":
				cmd.Printf("[OK] Appended the Spec-Driven Development section to %s\n", res.Path)
			case "replaced":
				cmd.Printf("[OK] Replaced the Spec-Driven Development section in %s\n", res.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "target repo root (default: the current git repo)")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing SDD section in place")
	return cmd
}
