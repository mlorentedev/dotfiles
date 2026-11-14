package cmd

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fleetlint"
	"github.com/spf13/cobra"
)

func newLintFleetCmd() *cobra.Command {
	var dir, fleet, summary, defaultBranch string
	var only []string
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

An exception missing any of the four fields, naming a rule that does not
exist, or matching no finding is itself a finding (fleet/exception).

--fleet checks every git checkout directly under a directory, each against
the branch its own origin/HEAD names, and prefixes each finding with the
checkout's name. A checkout that cannot be read is reported and the others
are still checked.

--rule <id>, repeatable, reports only the named rules: findings of the
others neither print nor count toward the exit code. Exceptions are
applied first, so a stale one shows only under --rule fleet/exception.
A selection without fleet/branch-filter or fleet/exception needs no
default branch.

fleet/sha-pin also reads the steps of each local composite action,
.github/actions/<name>/action.yml (or .gitea/actions, or action.yaml).

Exit 0 is clean, 1 is findings, 2 is a workflow or exceptions file that
cannot be read, or an unknown --rule.`,
		Args: func(c *cobra.Command, args []string) error {
			switch {
			case len(args) > 0:
				return guardUsage(c, "lint fleet takes no arguments; name the repository with --dir")
			case fleet != "" && (c.Flags().Changed("dir") || defaultBranch != ""):
				return guardUsage(c, "--fleet reads each checkout's own default branch; it takes neither --dir nor --default-branch")
			}
			for _, r := range only {
				if !slices.Contains(fleetlint.Rules(), r) {
					return guardUsage(c, fmt.Sprintf("--rule %s names no rule; the rules are %s", r, strings.Join(fleetlint.Rules(), ", ")))
				}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			deps := lintFleetDeps{branchOf: fleetlint.DefaultBranch, only: only}
			if fleet != "" {
				return runLintFleetAll(c.OutOrStdout(), c.ErrOrStderr(), deps, fleet, summary)
			}
			return runLintFleet(c.OutOrStdout(), c.ErrOrStderr(), deps, dir, defaultBranch, summary)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "repository root to check")
	cmd.Flags().StringVar(&fleet, "fleet", "", "directory whose git checkouts are each checked")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "the branch every branch filter must include (default: what origin/HEAD names)")
	cmd.Flags().StringSliceVar(&only, "rule", nil, "report only this rule's findings; repeat for more (default: every rule)")
	cmd.Flags().StringVar(&summary, "summary", "", "Markdown file to append the findings table to, e.g. $GITHUB_STEP_SUMMARY")
	return cmd
}

// lintFleetDeps is the command's one external surface: what a checkout's
// origin/HEAD names, which reads git. only is the --rule selection, empty for
// every rule.
type lintFleetDeps struct {
	branchOf func(dir string) string
	only     []string
}

// lintRepo lints one checkout against defaultBranch, else the one its
// origin/HEAD names, and keeps the selected rules' findings.
func lintRepo(deps lintFleetDeps, dir, defaultBranch string) ([]fleetlint.Finding, error) {
	if defaultBranch == "" {
		defaultBranch = deps.branchOf(dir)
	}
	return fleetlint.LintRules(dir, defaultBranch, deps.only)
}

func runLintFleet(stdout, stderr io.Writer, deps lintFleetDeps, dir, defaultBranch, summary string) error {
	findings, err := lintRepo(deps, dir, defaultBranch)
	if errors.Is(err, fleetlint.ErrNoDefaultBranch) {
		err = fmt.Errorf("%w: pass --default-branch", err)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "[ERROR] %v\n", err)
		return withExitCode(2, err)
	}
	return reportFleet(stdout, stderr, findings, summary, true)
}

// runLintFleetAll lints every checkout under dir. One that cannot be read
// is an error for the run, but not a reason to leave the rest unchecked.
func runLintFleetAll(stdout, stderr io.Writer, deps lintFleetDeps, dir, summary string) error {
	repos, err := fleetlint.Repos(dir)
	if err == nil && len(repos) == 0 {
		err = fmt.Errorf("no git checkout under %s", dir)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "[ERROR] %v\n", err)
		return withExitCode(2, err)
	}
	var all []fleetlint.Finding
	var failed []string
	for _, r := range repos {
		findings, err := lintRepo(deps, filepath.Join(dir, r), "")
		if errors.Is(err, fleetlint.ErrNoDefaultBranch) {
			// --fleet takes no --default-branch: name the remedies that apply.
			err = fmt.Errorf("%w: run git remote set-head origin --auto in it, or lint it alone with --dir and --default-branch", err)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "[ERROR] %s: %v\n", r, err)
			failed = append(failed, r)
			continue
		}
		for _, f := range findings {
			f.Path = r + "/" + f.Path
			all = append(all, f)
		}
	}
	reportErr := reportFleet(stdout, stderr, all, summary, len(failed) == 0)
	if len(failed) > 0 {
		return withExitCode(2, fmt.Errorf("%d checkout(s) could not be read: %s", len(failed), strings.Join(failed, ", ")))
	}
	return reportErr
}

// reportFleet prints the findings and their table. complete is false when a
// checkout went unread: no findings then is not a clean fleet.
func reportFleet(stdout, stderr io.Writer, findings []fleetlint.Finding, summary string, complete bool) error {
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
	if complete {
		_, _ = fmt.Fprintln(stdout, "[OK] no fleet lint findings")
	}
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
