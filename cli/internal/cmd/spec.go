package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// now is the clock used to stamp created: in scaffolded specs. It is a package
// var so tests can pin it deterministically.
var now = func() time.Time { return time.Now().UTC() }

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Spec-driven development scaffolding (ADR-020)",
		Long: "spec scaffolds and manages per-feature SDD spec folders.\n" +
			"The Go twin of scripts/init-spec.sh + scripts/archive-spec.sh; templates\n" +
			"are embedded, so init works without the vault checked out.",
	}
	cmd.AddCommand(newSpecInitCmd())
	cmd.AddCommand(newSpecArchiveCmd())
	return cmd
}

func newSpecInitCmd() *cobra.Command {
	var (
		issueNum     int
		forceNoGate  bool
		bitacoraRepo string
	)

	cmd := &cobra.Command{
		Use:   "init <feature-id>",
		Short: "Scaffold a per-feature spec folder",
		Long: `Scaffold specs/<feature-id>/{proposal,tasks,verification}.md from the
embedded SDD templates.

Work-gate (ADR-018): the spec must be downstream of an OPEN GitHub issue.
Pass --issue <N>; the issue is verified via 'gh issue view' and its title is
recorded in the proposal's frontmatter (issue: owner/repo#N) and ## Why comment.
The issue's repo defaults to the current repo's origin; override with
--bitacora-repo owner/repo or $DOTF_BITACORA_REPO for a cross-repo work-gate.
Use --force-no-gate to scaffold without an issue (NOT RECOMMENDED).

Mechanical only: fill the proposal interactively afterwards ("/spec fill" in an
agent) or by hand. Do not skip the Why.`,
		Example:      "  dotf spec init AI-001-ollama-public --issue 42",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := spec.ValidateID(id); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}

			var issueTitle, repoSlug string
			if !forceNoGate {
				if issueNum == 0 {
					return fmt.Errorf("no work-gate given: pass --issue <number>.\n" +
						"Per ADR-018 every spec is downstream of an OPEN GitHub issue on the\n" +
						"bitácora Project. Options: (a) open/find the issue, re-run with --issue;\n" +
						"(b) re-run with --force-no-gate (NOT RECOMMENDED)")
				}
				// Resolve the repo that hosts the work-gate issue (HARNESS-023):
				// explicit --bitacora-repo, then DOTF_BITACORA_REPO, else the
				// current repo's origin slug. Used for BOTH the gate check and the
				// proposal's issue: frontmatter, so a cross-repo gate (e.g. a
				// kubelab spec gated by a knowledge issue) resolves correctly
				// instead of silently checking the current repo.
				repoSlug = bitacoraRepo
				if repoSlug == "" {
					repoSlug = os.Getenv("DOTF_BITACORA_REPO")
				}
				if repoSlug == "" {
					if s, e := initrepo.OriginRepo(repoRoot); e == nil {
						repoSlug = s
					}
				}
				if repoSlug == "" {
					return fmt.Errorf("cannot determine the repo hosting issue #%d: no "+
						"--bitacora-repo, no DOTF_BITACORA_REPO, and no origin remote in %s.\n"+
						"Pass --bitacora-repo owner/repo (e.g. mlorentedev/knowledge) or set "+
						"DOTF_BITACORA_REPO", issueNum, repoRoot)
				}
				if !initrepo.ValidRepoSlug(repoSlug) {
					return fmt.Errorf("invalid bitácora repo %q: want owner/name", repoSlug)
				}
				issueTitle, err = spec.Gate(issueNum, repoSlug)
				if err != nil {
					return err
				}
				cmd.Printf("[INFO] Work-gate OK: %s#%d is open — %s\n", repoSlug, issueNum, issueTitle)
			}

			date := now().Format("2006-01-02")
			warning, err := spec.Scaffold(repoRoot, id, date, repoSlug, issueNum, issueTitle)
			if warning != "" {
				cmd.PrintErrf("[WARN] %s\n", warning)
			}
			if err != nil {
				return err
			}

			cmd.Printf("\n[OK] Created: specs/%s\n", id)
			cmd.Printf("     proposal.md, tasks.md, verification.md\n")
			if issueNum > 0 {
				cmd.Printf("     Work-gate linked: issue #%d\n", issueNum)
			}
			cmd.Printf("\nNext: fill proposal.md interactively (\"/spec fill %s\" in an agent)\n", id)
			cmd.Printf("      or edit by hand. Do not skip the Why.\n")
			return nil
		},
	}

	cmd.Flags().IntVar(&issueNum, "issue", 0, "GitHub issue number that gates this work (must exist and be OPEN)")
	cmd.Flags().StringVar(&bitacoraRepo, "bitacora-repo", "", "owner/repo hosting the work-gate issue (default: current repo's origin, or $DOTF_BITACORA_REPO)")
	cmd.Flags().BoolVar(&forceNoGate, "force-no-gate", false, "skip the open-issue work-gate (NOT RECOMMENDED — the gate is the SSOT)")
	return cmd
}

func newSpecArchiveCmd() *cobra.Command {
	var (
		prURL       string
		abandoned   bool
		forceDrafts bool
	)

	cmd := &cobra.Command{
		Use:   "archive <feature-id>",
		Short: "Archive (or abandon) a per-feature spec folder",
		Long: `Move specs/<feature-id>/ into specs/archive/ (or specs/archive/_abandoned/
under --abandoned) and rewrite the proposal status to archived/abandoned.

Mechanical only — the Go twin of scripts/archive-spec.sh. A pre-flight refuses to
archive while unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags remain (override
with --force-with-drafts). Vault promotion (lessons/ADR/pattern) and any backlog
tick stay interactive via "/spec archive" in an agent.`,
		Example:      "  dotf spec archive AI-001-ollama-public --pr https://github.com/owner/repo/pull/42",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}

			target, err := spec.Archive(repoRoot, id, spec.ArchiveOptions{
				Abandoned:       abandoned,
				ForceWithDrafts: forceDrafts,
				PRURL:           prURL,
				Date:            now().Format("2006-01-02"),
			})
			if err != nil {
				return err
			}

			rel, relErr := filepath.Rel(repoRoot, target)
			if relErr != nil {
				rel = target
			}
			status := "archived"
			if abandoned {
				status = "abandoned"
			}
			cmd.Printf("\n[OK] Archived: specs/%s -> %s\n", id, rel)
			cmd.Printf("     status: %s\n", status)
			if prURL != "" {
				cmd.Printf("     PR: %s\n", prURL)
			}
			cmd.Printf("\nVault promotion (lessons/ADR/pattern) and backlog tick must be done\n")
			cmd.Printf("separately (via \"/spec archive\" in an agent, or by hand).\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&prURL, "pr", "", "record this PR URL in proposal.md (informational)")
	cmd.Flags().BoolVar(&abandoned, "abandoned", false, "route to specs/archive/_abandoned/ and set status abandoned")
	cmd.Flags().BoolVar(&forceDrafts, "force-with-drafts", false, "archive even with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags")
	return cmd
}
