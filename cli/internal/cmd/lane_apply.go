package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// laneVerbs builds the verbs apply may run; a variable so a test answers
// without sweeping or starting anything.
var laneVerbs = realLaneVerbs

func newLaneApplyCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "apply",
		Short: "Run the auto actions of the plan, and nothing else",
		Long: "apply runs what plan classes auto, through the existing verbs: worktree sweep\n" +
			"for reaps (under its own fail-closed gates) and lane next for the next queued\n" +
			"item. It never adopts a worktree and never lands a PR, and a pending or\n" +
			"ambiguous action is left to the time or the person it names. It prints\n" +
			"changed=N; a second run over the result prints changed=0.",
		Example:       "  dotf lane apply\n  dotf lane apply --json",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			src, err := laneSources()
			if err != nil {
				return err
			}
			rows, err := lane.Status(src)
			if err != nil {
				return err
			}
			done, err := lane.Apply(lane.Plan(rows, src.Queue, src.Now), laneVerbs(c, src))
			if asJSON {
				if encErr := json.NewEncoder(c.OutOrStdout()).Encode(done); encErr != nil && err == nil {
					err = encErr
				}
				return err
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "changed=%d\n", done.Changed)
			return err
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "output in JSON format")
	return c
}

// realLaneVerbs wires apply to worktree sweep and lane next, run from the
// repository the command is in. The sweep asks the same forge, through the same
// cache, that status and plan asked: one answer to "is it merged?", so apply
// never reaps what plan handed to a person.
func realLaneVerbs(c *cobra.Command, src lane.Sources) lane.Verbs {
	return lane.Verbs{
		Sweep: func() (int, error) {
			root, err := resolveCommandRepoRoot("")
			if err != nil {
				return 0, err
			}
			runner := lane.ForgeSweep{RealSweepRunner: &worktree.RealSweepRunner{}, Forge: src.Forge}
			report, err := worktree.SweepWithRunner(worktree.SweepOptions{RepoRoot: root, Sessions: sessionStores()}, runner, src.Now)
			if err != nil {
				return 0, err
			}
			return len(report.Reaped), nil
		},
		Next: func() (bool, error) {
			out, err := doLaneNext(c, "", false)
			if err != nil {
				return false, err
			}
			printOutcome(c.ErrOrStderr(), out)
			return out.Started, nil
		},
	}
}
