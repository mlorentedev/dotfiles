package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

func newLanePlanCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "plan",
		Short: "Show what a reconcile would do, and change nothing",
		Long: "plan reads the same rows as status and prints the actions a reconcile would\n" +
			"take, each in one class: auto (an existing verb accepts it without --force),\n" +
			"pending (only time is missing; it says when it is due) or ambiguous (it\n" +
			"carries the exact command that resolves it). It also prints the landing\n" +
			"order. Adoption and landing are never auto: a PR is landed by the lane that\n" +
			"opened it, or by the owner, with the command printed for the head.",
		Example:       "  dotf lane plan\n  dotf lane plan --json",
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
			plan := lanePlan{Actions: lane.Plan(rows, src.Queue, src.Now), Landing: lane.LandingOrder(rows)}
			if plan.Actions == nil {
				plan.Actions = []lane.Action{}
			}
			if plan.Landing == nil {
				plan.Landing = []lane.Landing{}
			}
			if asJSON {
				enc := json.NewEncoder(c.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(plan)
			}
			return writeLanePlan(c.OutOrStdout(), plan, src.Now)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "output in JSON format")
	return c
}

type lanePlan struct {
	Actions []lane.Action  `json:"actions"`
	Landing []lane.Landing `json:"landing"`
}

func writeLanePlan(w io.Writer, p lanePlan, now time.Time) error {
	if len(p.Actions) == 0 {
		_, _ = fmt.Fprintln(w, "nothing to do")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "CLASS\tACTION\tBRANCH\tWHEN / RESOLVE")
		for _, a := range p.Actions {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.Class, a.Kind, a.Branch, planDetail(a, now))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	if len(p.Landing) > 0 {
		_, _ = fmt.Fprintln(w, "\nlanding order:")
		for i, l := range p.Landing {
			line := fmt.Sprintf("  %d. #%d %s", i+1, l.PR, l.Branch)
			if l.Command != "" {
				line += "\n     " + l.Command
			}
			_, _ = fmt.Fprintln(w, line)
		}
	}
	return nil
}

// planDetail is the last column: when a pending action is due, the command
// that resolves an ambiguous one, or the reason where there is neither.
func planDetail(a lane.Action, now time.Time) string {
	switch {
	case a.Class == lane.ClassPending:
		return "due in " + a.Due.Sub(now).Round(time.Minute).String() + " (" + a.Due.Format(time.RFC3339) + ")"
	case a.Resolve != "":
		return a.Resolve
	}
	return a.Reason
}
