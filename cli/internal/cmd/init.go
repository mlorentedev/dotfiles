package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	var (
		stack      string
		skipVault  bool
		skipGithub bool
		skipAgents bool
	)

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a fully-practiced repo from embedded templates (ADR-022)",
		Long: `init scaffolds a repo (the current dir, or [path]) with the cross-project
practice stack baked in: the placement-model structure, AGENTS.md + the
Spec-Driven Development section, a CLAUDE.md pointer, stack-appropriate CI,
pre-commit (gitleaks), .gitignore, an env-contract skeleton, stack init, git
init, and — when present — the vault project entry and GitHub repo defaults.

Templates are embedded in the binary, so init is self-contained: it works on a
machine with no vault and no ~/.claude, and the generated AGENTS.md never leaks
an unexpanded $VAULT_PATH. Files are skip-if-present (a re-run never clobbers),
and host-coupled steps (vault, GitHub, pre-commit) degrade to a [skip] when
their dependency is absent — never fatal.

Use the subcommands to run one piece on an existing repo: 'dotf init agents'
(AGENTS.md + SDD) and 'dotf init github' (repo defaults).`,
		Example:      "  dotf init\n  dotf init my-new-repo --stack go\n  dotf init . --skip-vault --skip-github",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			root, err := filepath.Abs(target)
			if err != nil {
				return err
			}

			vault := ""
			if !skipVault {
				vault = initrepo.ResolveVault()
			}
			claudeDir := ""
			if home, herr := os.UserHomeDir(); herr == nil {
				claudeDir = filepath.Join(home, ".claude", "projects")
			}

			report, err := initrepo.Init(initrepo.InitOptions{
				Root:              root,
				Stack:             stack,
				Date:              now().Format("2006-01-02"),
				SkipAgents:        skipAgents,
				SkipGithub:        skipGithub,
				VaultPath:         vault,
				ClaudeProjectsDir: claudeDir,
			})
			if err != nil {
				return err
			}

			cmd.Printf("\n[OK] Initialized %s\n", report.Root)
			for _, s := range report.Steps {
				cmd.Printf("  %-10s [%s] %s\n", s.Name, s.Status, s.Detail)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&stack, "stack", "none", "project stack to initialize: go|python|node|none")
	cmd.Flags().BoolVar(&skipVault, "skip-vault", false, "skip the vault project entry")
	cmd.Flags().BoolVar(&skipGithub, "skip-github", false, "skip applying GitHub repo defaults")
	cmd.Flags().BoolVar(&skipAgents, "skip-agents", false, "skip writing AGENTS.md + the SDD section")

	cmd.AddCommand(newInitAgentsCmd())
	cmd.AddCommand(newInitGithubCmd())
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

// newInitGithubCmd builds `dotf init github`: apply opinionated GitHub repo
// defaults via gh. The Go twin of scripts/init-repo-github-defaults.sh.
func newInitGithubCmd() *cobra.Command {
	var (
		repo   string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "github",
		Short: "Apply GitHub repo defaults (delete_branch_on_merge)",
		Long: `github applies opinionated GitHub repo-level defaults via the gh CLI —
currently delete_branch_on_merge=true, which auto-deletes a PR's head branch on
merge (the orphan-branch hygiene fix). Idempotent: a no-op when already enabled.

Without --repo it derives owner/name from the current repo's origin remote.
Host-coupling degrades gracefully (ADR-022): a missing gh, missing remote, or an
unreadable repo is a [WARN] skip with exit 0, never fatal.`,
		Example:      "  dotf init github\n  dotf init github --repo owner/name --dry-run",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target := repo
			switch {
			case target == "":
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				root, err := spec.RepoRoot(cwd)
				if err != nil {
					cmd.PrintErrln("[WARN] not in a git repo and no --repo given; skipping GitHub defaults")
					return nil
				}
				if target, err = initrepo.OriginRepo(root); err != nil {
					cmd.PrintErrln("[WARN] no origin remote; skipping GitHub defaults (add a remote, then re-run)")
					return nil
				}
			case !initrepo.ValidRepoSlug(target):
				return fmt.Errorf("--repo must be owner/name, got %q", target)
			}

			res, err := initrepo.ApplyDeleteBranchOnMerge(target, dryRun)
			if err != nil {
				return err
			}
			switch res.Action {
			case "skipped":
				cmd.PrintErrf("[WARN] %s — skipping GitHub defaults for %s\n", res.Reason, res.Repo)
			case "already-enabled":
				cmd.Printf("[OK] delete_branch_on_merge already enabled on %s\n", res.Repo)
			case "dry-run":
				cmd.Printf("[DRY-RUN] would enable delete_branch_on_merge on %s\n", res.Repo)
			case "enabled":
				cmd.Printf("[OK] enabled delete_branch_on_merge on %s\n", res.Repo)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "target repo owner/name (default: derived from origin)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without applying")
	return cmd
}
