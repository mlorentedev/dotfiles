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

// claimRow is one claim as `lane claims` shows it. Closed is true when the
// issue is closed: its claim blocks nothing (the queue skips closed issues)
// and a person can release it.
type claimRow struct {
	Issue  int       `json:"issue"`
	Holder string    `json:"holder"`
	At     time.Time `json:"at"`
	Age    string    `json:"age"`
	Closed bool      `json:"closed"`
}

func newLaneClaimsCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "claims",
		Short: "List the claims held on the remote, with their holder and age",
		Long: "claims reads refs/dotf/claims/* on the lane's remote and shows who holds each\n" +
			"issue and for how long. A claim never expires on its own: one whose holder is\n" +
			"gone, or whose issue is closed, is released by a person with\n" +
			"`dotf lane release <issue> --force`. It changes nothing.",
		Example:       "  dotf lane claims\n  dotf lane claims --json",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, top, parent, err := laneRepo("claims")
			if err != nil {
				return err
			}
			claims, err := laneClaimer(cfg, top, parent).List()
			if err != nil {
				return err
			}
			rows := claimRows(claims, lane.GH{Run: laneRun}, time.Now())
			if asJSON {
				return writeClaimsJSON(c.OutOrStdout(), rows)
			}
			return writeClaimsTable(c.OutOrStdout(), rows)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "output in JSON format")
	return c
}

// claimRows joins each claim with its issue's state. An issue that cannot be
// read is left open: a claim is never called releasable on a guess.
func claimRows(claims []lane.Claim, src lane.IssueSource, now time.Time) []claimRow {
	rows := make([]claimRow, 0, len(claims))
	for _, cl := range claims {
		r := claimRow{Issue: cl.Issue, Holder: cl.Holder, At: cl.At, Age: "unknown"}
		if !cl.At.IsZero() {
			r.Age = now.Sub(cl.At).Round(time.Minute).String()
		}
		if is, err := src.Issue(cl.Issue); err == nil && is.State == "CLOSED" {
			r.Closed = true
		}
		rows = append(rows, r)
	}
	return rows
}

func writeClaimsJSON(w io.Writer, rows []claimRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeClaimsTable(w io.Writer, rows []claimRow) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no claims")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ISSUE\tHOLDER\tAGE\tNOTE")
	for _, r := range rows {
		note := "-"
		if r.Closed {
			note = "issue closed: release with `dotf lane release " + fmt.Sprint(r.Issue) + " --force`"
		}
		_, _ = fmt.Fprintf(tw, "#%d\t%s\t%s\t%s\n", r.Issue, r.Holder, r.Age, note)
	}
	return tw.Flush()
}
