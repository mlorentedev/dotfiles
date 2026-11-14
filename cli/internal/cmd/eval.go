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
	c.AddCommand(newEvalCheckCmd())
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
