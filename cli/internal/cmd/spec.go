package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

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
			"The Go twin of scripts/init-spec.sh; templates are embedded, so it\n" +
			"works without the vault checked out.",
	}
	cmd.AddCommand(newSpecInitCmd())
	return cmd
}

func newSpecInitCmd() *cobra.Command {
	var (
		issueNum    int
		forceNoGate bool
	)

	cmd := &cobra.Command{
		Use:   "init <feature-id>",
		Short: "Scaffold a per-feature spec folder",
		Long: `Scaffold specs/<feature-id>/{proposal,tasks,verification}.md from the
embedded SDD templates.

Work-gate (ADR-018): the spec must be downstream of an OPEN GitHub issue.
Pass --issue <N>; the issue is verified via 'gh issue view' and its title is
recorded in the proposal's frontmatter (issue: dotfiles#N) and ## Why comment.
Use --force-no-gate to scaffold without an issue (NOT RECOMMENDED).

Mechanical only: fill the proposal interactively afterwards ("/spec fill" in an
agent) or by hand. Do not skip the Why.`,
		Example:      "  dot spec init AI-001-ollama-public --issue 42",
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

			var issueTitle string
			if !forceNoGate {
				if issueNum == 0 {
					return fmt.Errorf("no work-gate given: pass --issue <number>.\n" +
						"Per ADR-018 every spec is downstream of an OPEN GitHub issue on the\n" +
						"bitácora Project. Options: (a) open/find the issue, re-run with --issue;\n" +
						"(b) re-run with --force-no-gate (NOT RECOMMENDED)")
				}
				issueTitle, err = spec.Gate(issueNum)
				if err != nil {
					return err
				}
				cmd.Printf("[INFO] Work-gate OK: issue #%d is open — %s\n", issueNum, issueTitle)
			}

			date := now().Format("2006-01-02")
			warning, err := spec.Scaffold(repoRoot, id, date, issueNum, issueTitle)
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
	cmd.Flags().BoolVar(&forceNoGate, "force-no-gate", false, "skip the open-issue work-gate (NOT RECOMMENDED — the gate is the SSOT)")
	return cmd
}
