package cmd

import (
	"errors"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/prreview"
	"github.com/spf13/cobra"
)

// prGuardGH is how `pr guard` runs gh; nil is the gh on PATH. A variable so a
// test answers the API without GitHub.
var prGuardGH prreview.GHRunner

// newPrGuardCmd is `dotf pr guard`: did this run of the pr-agent workflow
// publish a review?
//
// Exit status: the probe exits 0 whatever it answers; the final guard exits 0
// when a review was published and 1 when not; 2 is a usage error, tagged with
// withExitCode. A flag cobra itself rejects still exits 1.
func newPrGuardCmd() *cobra.Command {
	var o guardOpts
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Check whether this run of the pr-agent workflow published a review",
		Long: `Check whether this run of the pr-agent workflow published a review. A review is
a comment that is all three of: authored by github-actions[bot], created or
edited at or after this run's start (STARTED), and carrying a marker that
harness/review-attestation.json declares for github-actions, read at BASE_REF.
A PR that introduces the entry is checked with PR-Agent's own heading.

With --probe --output FILE it sits between the first attempt and the second: it
appends published=true, published=false or published=unknown to FILE and exits 0.
Only a measured false lets the second attempt run; unknown (the API failed, or no
marker is declared) never does.

Without --probe it is the final guard: it exits 0 when a review was published,
and otherwise 1, naming the last attempt that ran (ATTEMPTS, one
"<outcome> <model>" line per attempt) and why.

It calls the GitHub API through gh (GH_TOKEN). A 404 is an answer; any other
failure is retried three times, GUARD_RETRY_SECONDS apart (default 5), and then
reported as an outage, never as a missing review. It reads GITHUB_REPOSITORY,
PR_NUMBER, BASE_REF, STARTED, HEAD_SHA, ATTEMPTS, ROUTE_NOTE and ROUTE_OUTCOME.
Annotations go to stdout.

Exit status is 0 when a review was published (always, with --probe), 1 when
not, and 2 for a usage error.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "pr guard takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          o.run,
	}
	cmd.Flags().BoolVar(&o.probe, "probe", false, "answer published=true|false|unknown into --output, and exit 0")
	cmd.Flags().StringVar(&o.output, "output", "", "with --probe, append the answer to this file (required)")
	return cmd
}

type guardOpts struct {
	probe  bool
	output string
}

func (o *guardOpts) run(c *cobra.Command, _ []string) error {
	if o.probe && o.output == "" {
		return guardUsage(c, "pr guard: --probe needs --output FILE")
	}
	in := prreview.GuardInputs{Repo: os.Getenv("GITHUB_REPOSITORY"), PR: os.Getenv("PR_NUMBER"),
		BaseRef: os.Getenv("BASE_REF"), Started: os.Getenv("STARTED"), HeadSHA: os.Getenv("HEAD_SHA"),
		Attempts: os.Getenv("ATTEMPTS"), RouteNote: os.Getenv("ROUTE_NOTE"),
		RouteOutcome: os.Getenv("ROUTE_OUTCOME"), RetrySeconds: os.Getenv("GUARD_RETRY_SECONDS")}
	env := prreview.GuardEnv{Out: c.OutOrStdout(), GH: prGuardGH}
	if !o.probe {
		return prreview.Guard(c.Context(), in, env)
	}
	answer := prreview.ProbePublished(c.Context(), in, env)
	if err := appendText(o.output, "published="+answer+"\n"); err != nil {
		return guardUsage(c, "pr guard: "+err.Error())
	}
	return nil
}

// guardUsage reports a usage error on stderr and exits 2.
func guardUsage(c *cobra.Command, msg string) error {
	c.PrintErrln(msg)
	return withExitCode(2, errors.New(msg))
}
