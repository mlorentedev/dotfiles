package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fleetlint"
	"github.com/spf13/cobra"
)

func newLintFleetCmd() *cobra.Command {
	var dir, summary, defaultBranch string
	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Check a repository's CI workflows against the fleet's CI standard",
		Long: `Check the workflows under .github/workflows and .gitea/workflows:

  fleet/timeout              every job that runs steps sets timeout-minutes
  fleet/sha-pin              every remote uses: is pinned by a full commit SHA
  fleet/persist-credentials  every actions/checkout sets persist-credentials: false
  fleet/no-secrets-inherit   no job passes secrets: inherit
  fleet/cancel-in-progress   a pull-request workflow cancels its stale runs
  fleet/os-on-pull-request  no Windows or macOS job runs on a pull request
                             without an if:
  fleet/skip-cache           no setup action or linter switches its cache off
  fleet/branch-filter        a push or pull_request branch filter includes
                             the default branch (--default-branch, else what
                             origin/HEAD names; unknown is an error)

A repository silences one rule on one file in .github/lint-exceptions.yml:

  exceptions:
    - rule: fleet/timeout
      path: .github/workflows/ci.yml
      reason: why the rule does not apply
      issue: the issue that tracks it

An exception missing any of the four fields is itself a finding
(fleet/exception) and silences nothing. Exit 0 is clean, 1 is findings,
2 is a workflow or exceptions file that cannot be read.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "lint fleet takes no arguments; name the repository with --dir")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return runLintFleet(c.OutOrStdout(), c.ErrOrStderr(), lintFleetDeps{branchOf: fleetlint.DefaultBranch}, dir, defaultBranch, summary)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "repository root to check")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "the branch every branch filter must include (default: what origin/HEAD names)")
	cmd.Flags().StringVar(&summary, "summary", "", "Markdown file to append the findings table to, e.g. $GITHUB_STEP_SUMMARY")
	return cmd
}

// lintFleetDeps is the command's one external surface: what a checkout's
// origin/HEAD names, which reads git.
type lintFleetDeps struct {
	branchOf func(dir string) string
}

func runLintFleet(stdout, stderr io.Writer, deps lintFleetDeps, dir, defaultBranch, summary string) error {
	if defaultBranch == "" {
		defaultBranch = deps.branchOf(dir)
	}
	findings, err := fleetlint.Lint(dir, defaultBranch)
	if errors.Is(err, fleetlint.ErrNoDefaultBranch) {
		err = fmt.Errorf("%w: pass --default-branch", err)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "[ERROR] %v\n", err)
		return withExitCode(2, err)
	}
	for _, f := range findings {
		_, _ = fmt.Fprintln(stdout, f)
	}
	if summary != "" {
		if err := appendText(summary, fleetTable(findings)); err != nil {
			_, _ = fmt.Fprintf(stderr, "[ERROR] write summary: %v\n", err)
			return withExitCode(2, err)
		}
	}
	if len(findings) > 0 {
		return withExitCode(1, fmt.Errorf("%d fleet lint finding(s)", len(findings)))
	}
	_, _ = fmt.Fprintln(stdout, "[OK] no fleet lint findings")
	return nil
}

func fleetTable(findings []fleetlint.Finding) string {
	var b strings.Builder
	b.WriteString("\n### Fleet lint\n\n")
	if len(findings) == 0 {
		b.WriteString("No findings.\n")
		return b.String()
	}
	b.WriteString("| Rule | Where | Finding |\n|---|---|---|\n")
	esc := func(s string) string { return strings.ReplaceAll(s, "|", `\|`) }
	for _, f := range findings {
		fmt.Fprintf(&b, "| `%s` | `%s:%d` | %s |\n", f.Rule, esc(f.Path), f.Line, esc(f.Message))
	}
	return b.String()
}
