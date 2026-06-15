package cmd

import (
	"github.com/spf13/cobra"
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
	return cmd
}
