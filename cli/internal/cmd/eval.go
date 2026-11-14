package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/evals"
)

// errEvalCheck is returned when at least one case fails its self-check.
var errEvalCheck = errors.New("eval check: cases failed")

func newEvalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "eval",
		Short:        "Measure models on the harness's own tasks, graded without a judge model",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newEvalCheckCmd(), newEvalRunCmd())
	return c
}

func newEvalCheckCmd() *cobra.Command {
	var root string
	c := &cobra.Command{
		Use:   "check",
		Short: "Prove every eval case's key against its own source, with no model involved",
		Long: "check reads the repository's eval cases (" + evals.ReplayCaseDir + "/*.json, and\n" +
			evals.PlantedCaseDir + "/*/case.json where it exists) and, for each one:\n" +
			"  - validates it against the case schema embedded in this binary;\n" +
			"  - holds its reviewed_sha, verdict and key_author to its source's frontmatter;\n" +
			"  - grades its key against its own source: every expected and optional entry\n" +
			"    must match a row, one to one, and a planted reference may match no decoy;\n" +
			"  - matches each decoy and noise entry against its recorded sample.\n" +
			"It exits non-zero when any case fails. A repository without cases passes.",
		Example:       "  dotf eval check\n  dotf eval check --repo-root ../kubelab",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("eval check: not inside a repository — pass --repo-root")
			}
			results, err := evals.Check(root)
			if err != nil {
				c.PrintErrln("eval check:", err)
				return err
			}
			report, failed := renderEvalCheck(results)
			if _, err := io.WriteString(c.OutOrStdout(), report); err != nil {
				return err
			}
			if failed > 0 {
				return errEvalCheck
			}
			return nil
		},
	}
	c.Flags().StringVar(&root, "repo-root", "", "repository to check (default: the current one)")
	return c
}

// errEvalRunNotYet is returned by a run without --plan until the run path lands
// (HARNESS-101 slice 3).
var errEvalRunNotYet = errors.New("eval run: only --plan exists so far; launching candidates is not implemented yet")

func newEvalRunCmd() *cobra.Command {
	var (
		root       string
		candidates []string
		caseIDs    []string
		epochs     int
		maxRuns    int
		planOnly   bool
	)
	c := &cobra.Command{
		Use:   "run",
		Short: "Plan an evaluation run of the eval cases for named candidates (launching is not implemented yet)",
		Long: "run builds the matrix of every eval case × each --candidate × --epochs, and\n" +
			"refuses it above --max-runs: the cap is what authorises spend. --plan prints the\n" +
			"matrix and each candidate's estimate (NaN credits, agy runs) and launches nothing.\n" +
			"A corpus whose keys fail `dotf eval check` is not run. Candidates come from\n" +
			evals.CandidatesFile + ", never from the reviewer pool. Only --plan works so far:\n" +
			"launching and grading candidates is not implemented yet.",
		Example:       "  dotf eval run --plan --candidate nan/glm5.3-flash\n  dotf eval run --plan --candidate agy/gemini-4-hydrogen-high --case harness-088-r2 --epochs 1",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("eval run: not inside a repository — pass --repo-root")
			}
			cases, err := evals.LoadCases(root)
			if err != nil {
				return fmt.Errorf("eval run: %w", err)
			}
			if cases, err = evals.SelectCases(cases, caseIDs); err != nil {
				return fmt.Errorf("eval run: %w", err)
			}
			cs, err := evals.LoadCandidates(root)
			if err != nil {
				return fmt.Errorf("eval run: %w", err)
			}
			plan, err := evals.BuildPlan(cases, cs, candidates, epochs, maxRuns)
			if plan.Runs > 0 {
				if _, werr := io.WriteString(c.OutOrStdout(), plan.Render()); werr != nil {
					return werr
				}
			}
			if err != nil {
				return fmt.Errorf("eval run: %w", err)
			}
			if planOnly {
				return nil
			}
			return errEvalRunNotYet
		},
	}
	c.Flags().StringVar(&root, "repo-root", "", "repository holding the cases and candidates (default: the current one)")
	c.Flags().StringArrayVar(&candidates, "candidate", nil, "candidate id from "+evals.CandidatesFile+" (repeatable)")
	c.Flags().StringArrayVar(&caseIDs, "case", nil, "case id to run (repeatable; default: every case)")
	c.Flags().IntVar(&epochs, "epochs", evals.DefaultEpochs, "runs per (candidate, case)")
	c.Flags().IntVar(&maxRuns, "max-runs", evals.DefaultMaxRuns, "the most runs this invocation may launch")
	c.Flags().BoolVar(&planOnly, "plan", false, "print the matrix and the estimate, and launch nothing")
	return c
}

// renderEvalCheck formats the check report and counts the failed cases.
func renderEvalCheck(results []evals.CheckResult) (string, int) {
	var b strings.Builder
	if len(results) == 0 {
		b.WriteString("[OK] no eval cases in this repository\n")
		return b.String(), 0
	}
	failed := 0
	for _, r := range results {
		if len(r.Problems) == 0 {
			fmt.Fprintf(&b, "[OK]   %s\n", r.Path)
			continue
		}
		failed++
		fmt.Fprintf(&b, "[FAIL] %s\n", r.Path)
		for _, p := range r.Problems {
			fmt.Fprintf(&b, "         %s\n", p)
		}
	}
	if failed > 0 {
		fmt.Fprintf(&b, "%d of %d cases failed\n", failed, len(results))
	} else {
		fmt.Fprintf(&b, "%d cases checked\n", len(results))
	}
	return b.String(), failed
}
