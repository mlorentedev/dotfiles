package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/reproduce"
	"github.com/spf13/cobra"
)

// errNoVerdict is returned (silently) when the check ran but its exit says
// nothing about the defect. The row is printed either way; the exit status
// tells a script whether the row carries a verdict.
var errNoVerdict = errors.New("pr reproduce: no verdict")

// newPrReproduceCmd is TOOL-017 AC5, under the owner's rule on #1062
// (option A, and could-not-run as its own state).
func newPrReproduceCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "reproduce -- <command> [args...]",
		Short: "Run the check that decides a reviewer's finding, and print the row for the triage",
		Long: `reproduce runs one check against the committed tree and classifies its exit.
A reproduction is a committed failing test, or a command whose non-zero exit
is recorded; anything else, a model's judgement included, goes to a human.

  reproduced         the check exited non-zero: the defect exists
  did-not-reproduce  the check exited 0: the finding is refuted
  could-not-run      the check did not start, exited 126 or 127, was killed
                     or timed out: no verdict, never a reproduction

A tree with uncommitted changes, untracked files included, is refused: the
record names the commit the check ran on, and that is only true of a clean
tree. It prints one line to paste into the PR's ## Review triage comment and
exits 0 when the line carries a verdict, 1 when it does not. It applies
nothing and decides nothing on its own.`,
		Example: `  dotf pr reproduce -- go test ./internal/prtriage -run TestClassify
  dotf pr reproduce --timeout 2m -- bats tests/pr-agent-config.bats`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			res, err := reproduce.Run(c.Context(), reproduce.Options{Command: args, Timeout: timeout})
			if err != nil {
				c.PrintErrln("pr reproduce:", err)
				return err
			}
			c.Println(reproductionRow(res))
			if res.State == reproduce.CouldNotRun {
				if res.Output != "" {
					c.PrintErrln(strings.TrimRight(res.Output, "\n"))
				}
				return errNoVerdict
			}
			return nil
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "kill the check after this long; a killed check is could-not-run")
	return cmd
}

// reproductionRow is the line pasted under ## Review triage.
func reproductionRow(r reproduce.Result) string {
	head := r.Head
	if len(head) > 12 {
		head = head[:12]
	}
	return fmt.Sprintf("Reproduction: %s (exit %d) on %s: `%s`", r.State, r.Exit, head, strings.Join(r.Command, " "))
}
