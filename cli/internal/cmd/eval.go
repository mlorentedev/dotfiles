package cmd

import (
	"errors"
	"fmt"

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
			"    must match a row, one to one, and no decoy may match;\n" +
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
			out := c.OutOrStdout()
			if len(results) == 0 {
				_, err := fmt.Fprintln(out, "[OK] no eval cases in this repository")
				return err
			}
			failed := 0
			for _, r := range results {
				if len(r.Problems) == 0 {
					fmt.Fprintf(out, "[OK]   %s\n", r.Path)
					continue
				}
				failed++
				fmt.Fprintf(out, "[FAIL] %s\n", r.Path)
				for _, p := range r.Problems {
					fmt.Fprintf(out, "         %s\n", p)
				}
			}
			if failed > 0 {
				fmt.Fprintf(out, "%d of %d cases failed\n", failed, len(results))
				return errEvalCheck
			}
			_, err = fmt.Fprintf(out, "%d cases checked\n", len(results))
			return err
		},
	}
	c.Flags().StringVar(&root, "repo-root", "", "repository to check (default: the current one)")
	return c
}
