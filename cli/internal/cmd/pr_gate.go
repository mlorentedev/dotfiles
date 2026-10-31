package cmd

import (
	"os"
	"strconv"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// newPrGateCmd is `dotf pr gate`: is this push to a pull request reviewed?
//
// Exit status: 0 with a decision, an unreadable input included; 2 is a usage
// error, a flag cobra itself rejects among them.
func newPrGateCmd() *cobra.Command {
	var o gateOpts
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Decide whether a push to a pull request is reviewed",
		Long: `Decide whether a push to a pull request is reviewed. A push is reviewed
incrementally once --threshold non-merge commits are newer than the previous
review by github-actions[bot], and not reviewed below that. It is reviewed in full
when there is no previous review, when someone else's review marker follows it,
or when the commit it reviewed has left the PR.

It reaches nothing itself: the workflow fetches the two files. Input it cannot
read means a full review, never an error: a skip has to be justified by the data.

Prints run=true|false, reason=<text> and, when run=true, mode=full|incremental,
ready for $GITHUB_OUTPUT.

Exit status is 0 with a decision and 2 for a usage error.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "pr gate takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          o.run,
	}
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error { return guardUsage(c, "pr gate: "+err.Error()) })
	cmd.Flags().StringVar(&o.comments, "comments", "", "the PR's issue comments, as GET /issues/N/comments returns them (required)")
	cmd.Flags().StringVar(&o.commits, "commits", "", "the PR's commits, as GET /pulls/N/commits returns them (required)")
	// A string, so that a bad value is this command's usage error, not cobra's.
	cmd.Flags().StringVar(&o.threshold, "threshold", "3", "new non-merge commits needed to review a push")
	cmd.Flags().StringVar(&o.senderType, "sender-type", "", `the push event's sender.type; "Bot" is skipped, as PR-Agent skips it`)
	return cmd
}

type gateOpts struct {
	comments, commits, threshold, senderType string
}

func (o *gateOpts) run(c *cobra.Command, _ []string) error {
	if o.comments == "" || o.commits == "" {
		return guardUsage(c, "pr gate: --comments and --commits are required")
	}
	// Not Atoi: it accepts a sign, and a whole number has none.
	threshold, err := strconv.ParseUint(o.threshold, 10, 31)
	if err != nil {
		return guardUsage(c, "pr gate: --threshold must be a whole number")
	}
	// A read error leaves nil, which Gate takes as unreadable: a full review.
	comments, _ := os.ReadFile(o.comments)
	commits, _ := os.ReadFile(o.commits)
	d := prreview.Gate(prreview.GateInputs{Comments: comments, Commits: commits,
		Threshold: int(threshold), SenderType: o.senderType})
	_, err = c.OutOrStdout().Write([]byte(d.Output()))
	return err
}
