package cmd

import (
	"os"
	"slices"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// prDeclareGH is how `pr declare-unreviewed` runs gh; nil is the gh on PATH. A
// variable so a test answers the API without GitHub.
var prDeclareGH prreview.GHRunner

// newPrDeclareUnreviewedCmd is `dotf pr declare-unreviewed`: put the review
// escape on a PR nothing can review, and take it off once something can.
//
// Exit status: 0 when the PR is as it should be; 1 when it is not, or the
// registry declares no escape; 2 is a usage error.
func newPrDeclareUnreviewedCmd() *cobra.Command {
	var diff string
	cmd := &cobra.Command{
		Use:   "declare-unreviewed",
		Short: "Declare a pull request PR-Agent cannot review, or clear a stale declaration",
		Long: `Declare a pull request PR-Agent cannot review, or clear a stale declaration.

--diff takes the diff output of 'dotf pr reviewable', so a run reads the PR's
files once:

  empty       every file is one PR-Agent drops. Writes the rationale under
              escape.section in the PR body, unless the section already has
              text, adds escape.label beside the PR's other labels, and reads
              the label back.
  reviewable  a file reaches PR-Agent. Removes escape.label if it is there; the
              section is left for a human.
  unread      a draft, a closed PR, a forfeited branch or a failed read. Does
              nothing, without reading anything.

The label and the section come from the escape block of
harness/review-attestation.json, read at DEFAULT_BRANCH, the file the
review-attestation gate reads. A registry with no escape is an error.

It does not look at the event: the caller runs it on pull_request and on a
/review comment, not on /improve, which never publishes a review.

It calls the GitHub API through gh (GH_TOKEN), and reads GITHUB_REPOSITORY,
PR_NUMBER and DEFAULT_BRANCH. Annotations go to stdout.

Exit status is 0 when the PR is as it should be, 1 when it is not or the
registry declares no escape, and 2 for a usage error.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "pr declare-unreviewed takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			// An empty value is a reviewable step that never ran. Reading it as
			// unread would turn a broken workflow into a green no-op.
			if !slices.Contains([]string{prreview.DiffEmpty, prreview.DiffReviewable, prreview.DiffUnread}, diff) {
				return guardUsage(c, "pr declare-unreviewed: --diff must be empty, reviewable or unread")
			}
			in := prreview.DeclareInputs{Repo: os.Getenv("GITHUB_REPOSITORY"), PR: os.Getenv("PR_NUMBER"),
				DefaultBranch: os.Getenv("DEFAULT_BRANCH"), Diff: diff}
			if in.Repo == "" || in.PR == "" || in.DefaultBranch == "" {
				return guardUsage(c, "pr declare-unreviewed: GITHUB_REPOSITORY, PR_NUMBER and DEFAULT_BRANCH are required")
			}
			return prreview.DeclareUnreviewed(c.Context(), in, prreview.DeclareEnv{Out: c.OutOrStdout(), GH: prDeclareGH})
		},
	}
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return guardUsage(c, "pr declare-unreviewed: "+err.Error())
	})
	cmd.Flags().StringVar(&diff, "diff", "", "the diff output of dotf pr reviewable: empty, reviewable or unread")
	return cmd
}
