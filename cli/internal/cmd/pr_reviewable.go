package cmd

import (
	"os"
	"strconv"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// prReviewableGH is how `pr reviewable` runs gh; nil is the gh on PATH. A
// variable so a test answers the API without GitHub.
var prReviewableGH prreview.GHRunner

// newPrReviewableCmd is `dotf pr reviewable`: should PR-Agent run on this PR?
//
// Exit status: 0 with a decision; 1 when the audited PR-Agent commit is unknown;
// 2 is a usage error, an unwritable --output included.
func newPrReviewableCmd() *cobra.Command {
	var o reviewableOpts
	cmd := &cobra.Command{
		Use:   "reviewable",
		Short: "Decide whether PR-Agent should run on this pull request",
		Long: `Decide whether PR-Agent should run on this pull request. On a pull_request run it
is skipped when the live PR is merged, closed or a draft, or when its branch
starts with a --skip-branch-prefixes entry. On any run it is skipped when every
changed file is one PR-Agent would drop: .pr_agent.toml's [ignore], read at
DEFAULT_BRANCH, or a type its own filters refuse, read at the audited commit.

Only a definite answer skips. A read that fails keeps the review, or narrows the
skip. A skip is never a review: it forfeits the reviewer and grants nothing, and
a /review comment is not subject to the state or the branch.

The audited commit comes from --contract, or from --audited-ref where the
caller holds it beside the pinned image. Exactly one is required.

It calls the GitHub API through gh (GH_TOKEN), and reads GITHUB_REPOSITORY,
PR_NUMBER, GITHUB_EVENT_NAME and DEFAULT_BRANCH. Prints reviewable=true|false,
appended to --output or written to stdout. Annotations go to stdout.

Exit status is 0 with a decision, 1 when the audited commit is unknown, and 2
for a usage error.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "pr reviewable takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          o.run,
	}
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error { return guardUsage(c, "pr reviewable: "+err.Error()) })
	cmd.Flags().StringVar(&o.contract, "contract", "", "the upstream contract whose audited_ref names PR-Agent's commit")
	cmd.Flags().StringVar(&o.auditedRef, "audited-ref", "", "PR-Agent's audited commit, 40 hex characters")
	cmd.Flags().StringVar(&o.skipPrefixes, "skip-branch-prefixes", "", "branch prefixes that forfeit the review, separated by commas or whitespace")
	cmd.Flags().StringVar(&o.output, "output", "", "append reviewable=true|false to this file (default: stdout)")
	return cmd
}

type reviewableOpts struct {
	contract, auditedRef, skipPrefixes, output string
}

func (o *reviewableOpts) run(c *cobra.Command, _ []string) error {
	if (o.contract == "") == (o.auditedRef == "") {
		return guardUsage(c, "pr reviewable: exactly one of --contract and --audited-ref is required")
	}
	in := prreview.ReviewableInputs{Repo: os.Getenv("GITHUB_REPOSITORY"), PR: os.Getenv("PR_NUMBER"),
		Event: os.Getenv("GITHUB_EVENT_NAME"), DefaultBranch: os.Getenv("DEFAULT_BRANCH"),
		AuditedRef: o.auditedRef, SkipPrefixes: o.skipPrefixes}
	if in.Repo == "" || in.PR == "" || in.Event == "" || in.DefaultBranch == "" {
		return guardUsage(c, "pr reviewable: GITHUB_REPOSITORY, PR_NUMBER, GITHUB_EVENT_NAME and DEFAULT_BRANCH are required")
	}
	if o.contract != "" {
		in.AuditedRef = prreview.AuditedRefFromContract(o.contract)
	}
	ok, err := prreview.Reviewable(c.Context(), in,
		prreview.ReviewableEnv{Out: c.OutOrStdout(), GH: prReviewableGH})
	if err != nil {
		return err
	}
	line := "reviewable=" + strconv.FormatBool(ok) + "\n"
	if o.output == "" {
		_, err = c.OutOrStdout().Write([]byte(line))
		return err
	}
	if err := appendText(o.output, line); err != nil {
		return guardUsage(c, "pr reviewable: "+err.Error())
	}
	return nil
}
