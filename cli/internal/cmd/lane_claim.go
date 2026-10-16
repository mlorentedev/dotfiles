package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneClaimer is the claim store of the repository the command runs in: the
// remote of the lane's base, from the parent checkout, as the calling
// checkout. The holder is the checkout and not the process, so `lane claim`
// and a later `lane release` from the same worktree are the same claimant,
// while two sessions in two worktrees are not.
func laneClaimer(cfg lane.Config, top, parent string) lane.Claimer {
	remote, _ := cfg.Remote()
	return lane.Claimer{Run: laneRun, Dir: parent, Remote: remote, As: claimHolder(top), Now: time.Now}
}

func claimHolder(top string) string {
	name := "unknown"
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return name + "@" + host + ":" + top
}

// laneRepo resolves the checkout, its parent and the lane config, the way
// every lane verb that talks to the remote needs them.
func laneRepo(verb string) (cfg lane.Config, top, parent string, err error) {
	top, err = gitOut("rev-parse", "--show-toplevel")
	if err != nil {
		return cfg, "", "", fmt.Errorf("lane %s: not in a git checkout: %w", verb, err)
	}
	common, err := gitOut("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return cfg, "", "", fmt.Errorf("lane %s: %w", verb, err)
	}
	cfg, err = lane.LoadConfig(top)
	if err != nil {
		return cfg, "", "", err
	}
	return cfg, top, filepath.Dir(common), nil
}

func newLaneClaimCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claim <issue>",
		Short: "Claim an issue for this checkout with one atomic push",
		Long: "claim takes refs/dotf/claims/<issue> on the lane's remote with a push that\n" +
			"only succeeds while the ref does not exist, so of two sessions that claim\n" +
			"the same issue at once exactly one wins. The loser exits non-zero naming the\n" +
			"holder. `lane next` claims through the same call; this verb is for a session\n" +
			"that picked its issue by hand.",
		Example:      "  dotf lane claim 2551",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			issue, cl, err := claimTarget("claim", args[0])
			if err != nil {
				return err
			}
			if _, err := cl.Acquire(issue); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "claimed #%d as %s\n", issue, cl.As)
			return nil
		},
	}
}

func newLaneReleaseCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "release <issue>",
		Short: "Release this checkout's claim on an issue",
		Long: "release deletes the claim on <issue> if this checkout holds it, and does\n" +
			"nothing when nobody does. Another holder's claim is refused unless --force\n" +
			"says a person decided it is stale: a claim never expires on its own.",
		Example:      "  dotf lane release 2551\n  dotf lane release 2551 --force",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			issue, cl, err := claimTarget("release", args[0])
			if err != nil {
				return err
			}
			if err := cl.Release(issue, force); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "#%d is not claimed\n", issue)
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "release a claim another checkout holds")
	return c
}

func claimTarget(verb, arg string) (int, lane.Claimer, error) {
	issue, err := strconv.Atoi(arg)
	if err != nil || issue <= 0 {
		return 0, lane.Claimer{}, fmt.Errorf("lane %s: %q is not an issue number", verb, arg)
	}
	cfg, top, parent, err := laneRepo(verb)
	if err != nil {
		return 0, lane.Claimer{}, err
	}
	return issue, laneClaimer(cfg, top, parent), nil
}
