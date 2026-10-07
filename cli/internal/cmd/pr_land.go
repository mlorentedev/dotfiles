package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/prland"
	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
)

// errNotLanded is returned when a PR fails a landing condition; the reasons
// are printed first. Same exit contract as errPending.
var errNotLanded = errors.New("pr land: not merged")

// prLandOptions builds the GitHub wiring; tests replace it.
var prLandOptions = func(repo, registry string) prland.Options {
	return prland.Options{
		Repo: repo,
		Run: func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "gh", args...).Output() //nolint:gosec // gh with arguments this command builds
		},
		Untriaged: func(ctx context.Context) ([]int, error) {
			reg, err := prtriage.LoadRegistry(registry)
			if err != nil {
				return nil, err
			}
			pending, err := prtriage.FetchWithRegistry(ctx, repo, reg)
			if err != nil {
				return nil, err
			}
			numbers := make([]int, 0, len(pending))
			for _, st := range pending {
				numbers = append(numbers, st.PR.Number)
			}
			return numbers, nil
		},
	}
}

func newPrLandCmd() *cobra.Command {
	var (
		repo, registry string
		wait, update   bool
	)
	cmd := &cobra.Command{
		Use:   "land <number>",
		Short: "Squash-merge a PR only when CI, review triage and freshness all hold",
		Long: `land merges a pull request only when, on one head commit:

  - the PR is open, not a draft, and not a release-please PR (releases are
    cut by a human);
  - every check passed or was skipped;
  - its merge state is CLEAN: up to date with its base, with no conflicts;
  - the triage queue does not list it: reviewer output has been dispositioned.

The head is read again before the merge, and a head that moved is a refusal.
Every PR based on this one's branch is pointed at its base first, because
GitHub closes a PR whose base branch is deleted. The merge is a squash with
--match-head-commit, never --auto.

--wait waits for the checks before reading them. --update-branch handles the
one condition that time alone cannot fix: when BEHIND is the only reason
against the PR, it merges the base into the branch (a merge, not a rebase, so
the reviewer's push gate does not re-review), waits for the new CI and decides
again on the new head. A refusal lists every failed condition and exits 1.`,
		Example:       "  dotf pr land 2030 --wait",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			number, err := strconv.Atoi(strings.TrimPrefix(args[0], "#"))
			if err != nil {
				return fmt.Errorf("pr land: %q is not a PR number", args[0])
			}
			o := prLandOptions(repo, registry)
			o.Wait, o.UpdateBranch = wait, update
			res, err := prland.Land(c.Context(), o, number)
			if err != nil {
				c.PrintErrln("pr land:", err)
				return err
			}
			return printLandResult(c, number, res)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "owner/name (default: the current repository)")
	cmd.Flags().StringVar(&registry, "registry", filepath.Join("harness", "review-attestation.json"), "path to the reviewer registry")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for the checks before reading them")
	cmd.Flags().BoolVar(&update, "update-branch", false, "when BEHIND is the only failing condition, merge the base in, wait for CI and decide again")
	return cmd
}

func printLandResult(c *cobra.Command, number int, res prland.Result) error {
	w := c.OutOrStdout()
	if !res.Merged {
		_, _ = fmt.Fprintf(w, "[NOT MERGED] #%d at %.7s:\n", number, res.HeadSHA)
		for _, r := range res.Reasons {
			_, _ = fmt.Fprintf(w, "  - %s\n", r)
		}
		return errNotLanded
	}
	if res.Updated {
		_, _ = fmt.Fprintf(w, "merged the base into #%d first\n", number)
	}
	for _, d := range res.Retargeted {
		_, _ = fmt.Fprintf(w, "retargeted #%d before the merge deleted its base branch\n", d)
	}
	_, _ = fmt.Fprintf(w, "[MERGED] #%d at %.7s\n", number, res.HeadSHA)
	return nil
}
