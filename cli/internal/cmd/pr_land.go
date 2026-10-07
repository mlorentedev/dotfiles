package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/prland"
	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
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
		Use:   "land <number> [<number>...]",
		Short: "Squash-merge PRs, one at a time, only when CI, review triage and freshness all hold",
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
again on the new head. A base that moves again during that wait is merged in
again, up to three times. A refusal lists every failed condition and exits 1.

Several numbers land as one queue, in the order given and one at a time. A PR is
updated only during its own turn: landing PRs from separate processes makes
each of them update every PR after every merge, so n PRs cost n squared CI runs
and reviewer re-reviews. A PR that stops (a failed check, conflicts, reviewer
output awaiting triage, an error reading it) is reported and the queue moves on
to the next. At the end one summary lists the PRs merged and, for each one not
merged, why; the exit status is 1 when any PR was not merged.

One pr land runs per repository at a time. A second one started while the first
runs refuses and names the first one's PID. The lock is released when the
command exits however it exits; the lock of a process that died is taken over,
with a note.`,
		Example: `  dotf pr land 2030 --wait
  dotf pr land 2031 2032 2033 --wait --update-branch`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			numbers, err := parseLandNumbers(args)
			if err != nil {
				return err
			}
			o := prLandOptions(repo, registry)
			o.Wait, o.UpdateBranch = wait, update
			release, err := lockLand(c, o)
			if err != nil {
				c.PrintErrln("pr land:", err)
				return err
			}
			defer release()
			if len(numbers) == 1 {
				res, err := prland.Land(c.Context(), o, numbers[0])
				if err != nil {
					c.PrintErrln("pr land:", err)
					return err
				}
				return writeLandResult(c.OutOrStdout(), res)
			}
			return landQueue(c, o, numbers)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "owner/name (default: the current repository)")
	cmd.Flags().StringVar(&registry, "registry", filepath.Join("harness", "review-attestation.json"), "path to the reviewer registry")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for the checks before reading them")
	cmd.Flags().BoolVar(&update, "update-branch", false, "when BEHIND is the only failing condition, merge the base in, wait for CI and decide again")
	return cmd
}

// parseLandNumbers reads the PR numbers, refusing one that is not a number and
// one named twice: the second landing could only report the first one's merge
// as a refusal.
func parseLandNumbers(args []string) ([]int, error) {
	numbers := make([]int, 0, len(args))
	seen := map[int]bool{}
	for _, a := range args {
		n, err := strconv.Atoi(strings.TrimPrefix(a, "#"))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("pr land: %q is not a PR number", a)
		}
		if seen[n] {
			return nil, fmt.Errorf("pr land: #%d is listed twice", n)
		}
		seen[n] = true
		numbers = append(numbers, n)
	}
	return numbers, nil
}

// lockLand takes the per-repository lock under the state dir, so two landers do
// not race each other into updating every PR after every merge. A take-over of
// a dead holder's lock is noted on stderr.
func lockLand(c *cobra.Command, o prland.Options) (func(), error) {
	repo, err := prland.RepoName(c.Context(), o)
	if err != nil {
		return nil, err
	}
	state, err := env.StateDir()
	if err != nil {
		return nil, fmt.Errorf("cannot place the pr land lock: %w", err)
	}
	return prland.AcquireLock(filepath.Join(state, "pr-land"), repo, secrets.ProcessAlive, c.ErrOrStderr())
}

// landQueue lands the PRs in order, printing each one's outcome as it is known
// and one summary at the end. It returns errNotLanded when any was not merged.
func landQueue(c *cobra.Command, o prland.Options, numbers []int) error {
	w := c.OutOrStdout()
	results := prland.LandQueue(c.Context(), o, numbers, func(res prland.Result) {
		_ = writeLandResult(w, res)
	})
	return printLandSummary(w, results)
}

// printLandSummary lists the PRs merged and, for each one not merged, its
// reasons.
func printLandSummary(w io.Writer, results []prland.Result) error {
	var merged []string
	var stopped []prland.Result
	for _, r := range results {
		if r.Merged {
			merged = append(merged, fmt.Sprintf("#%d", r.Number))
		} else {
			stopped = append(stopped, r)
		}
	}
	_, _ = fmt.Fprintf(w, "\nsummary: %d merged, %d not merged\n", len(merged), len(stopped))
	if len(merged) > 0 {
		_, _ = fmt.Fprintf(w, "  merged: %s\n", strings.Join(merged, ", "))
	}
	if len(stopped) > 0 {
		_, _ = fmt.Fprintln(w, "  not merged:")
		for _, r := range stopped {
			at := ""
			if r.HeadSHA != "" { // a PR that could not be read has no head
				at = fmt.Sprintf(" at %.7s", r.HeadSHA)
			}
			_, _ = fmt.Fprintf(w, "    #%d%s:\n", r.Number, at)
			for _, reason := range r.Reasons {
				_, _ = fmt.Fprintf(w, "      - %s\n", reason)
			}
		}
		return errNotLanded
	}
	return nil
}

// writeLandResult prints one PR's outcome and returns errNotLanded when it was
// not merged.
func writeLandResult(w io.Writer, res prland.Result) error {
	number := res.Number
	if !res.Merged {
		at := ""
		if res.HeadSHA != "" {
			at = fmt.Sprintf(" at %.7s", res.HeadSHA)
		}
		_, _ = fmt.Fprintf(w, "[NOT MERGED] #%d%s:\n", number, at)
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
