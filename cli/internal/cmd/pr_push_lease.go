package cmd

import (
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/pushlease"
	"github.com/spf13/cobra"
)

// newPrPushLeaseCmd is `dotf pr push-lease`: force-update the checked-out
// branch on its remote with one explicit lease (HARNESS-203, #3612).
//
// Exit status: 0 pushed; 1 refused, or git rejected the push; 2 usage error.
func newPrPushLeaseCmd() *cobra.Command {
	var expect, remote string
	cmd := &cobra.Command{
		Use:   "push-lease --expect <sha> [--remote <name>]",
		Short: "Force-update the checked-out branch on its remote with an explicit lease",
		Long: `Force-update the checked-out branch on its remote with an explicit lease.

push-lease replaces the branch of the same name on the remote with the commit
checked out in the current directory:

  git push --force-with-lease=refs/heads/<b>:<expect> <remote> <HEAD sha>:refs/heads/<b>

The arguments are built here and git runs without a shell. This is how an agent
force-updates its own branch after a rebase; a raw forced git push stays denied
by the permission rules.

--expect is the full sha (40 or 64 hex) of the remote branch as the caller last
integrated it, typically the commit it pushed before rebasing. It is never
derived here: a lease read from the remote-tracking ref holds after any fetch
and protects nothing. The remote is read first, and a branch whose sha is not
--expect is refused, also when it already equals HEAD.

Refused before git pushes, with exit 1 and the reason on stderr:

  - a detached HEAD;
  - main or master, in any case;
  - --expect missing, abbreviated or not a sha;
  - --remote that is not a configured remote name (a URL or a path);
  - a remote whose push URL differs from its fetch URL (pushurl, or
    url.<base>.pushInsteadOf): every check reads the fetch URL;
  - a remote that cannot be read;
  - the remote's default branch, by what the remote reports or what
    refs/remotes/<remote>/HEAD recorded; neither known is a refusal too.

--remote defaults to the branch's push remote (branch.<b>.pushRemote, then
remote.pushDefault), else its upstream remote (branch.<b>.remote), else origin.

Pre-push hooks run. git never prompts on the terminal (GIT_TERMINAL_PROMPT=0),
so a remote asking for credentials fails instead of waiting. git's own output
goes to stderr; on success stdout gets one line, "<branch> <old>..<new>".

Exit status is 0 when pushed, 1 when refused or git rejected the push, and 2
for a usage error.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "pr push-lease takes no arguments: the branch is the one checked out")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			plan, err := pushlease.Prepare(pushlease.Request{Dir: dir, Remote: remote, Expect: expect})
			if err == nil {
				err = pushlease.Push(plan, c.OutOrStdout(), c.ErrOrStderr())
			}
			if err != nil {
				c.PrintErrln("pr push-lease: " + err.Error())
				return withExitCode(1, err)
			}
			return nil
		},
	}
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return guardUsage(c, "pr push-lease: "+err.Error())
	})
	cmd.Flags().StringVar(&expect, "expect", "", "full sha the remote branch holds, as you last integrated it (required)")
	cmd.Flags().StringVar(&remote, "remote", "", "configured remote to push to (default: push remote, else upstream, else origin)")
	return cmd
}
