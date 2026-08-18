package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
	"github.com/spf13/cobra"
)

// errPending is returned (silently) when at least one pull request is awaiting
// a disposition. main.go maps any non-nil error to exit status 1, which is the
// whole exit contract this CLI has — so a caller distinguishes "pending" from
// "could not answer" by reading the message, not the code. Inventing a second
// code here would mean changing main.go's contract for one subcommand.
var errPending = errors.New("pr triage-queue: reviewer output is awaiting a disposition")

// ghPR mirrors what `gh pr list --json` returns. It is deliberately separate
// from prtriage.PR: the vendor's shape is a boundary concern, and pinning the
// domain to it would make every test carry gh's nesting for no benefit.
type ghPR struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Comments []struct {
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"createdAt"`
	} `json:"comments"`
}

func newPrCmd() *cobra.Command {
	pr := &cobra.Command{
		Use:   "pr",
		Short: "Pull-request review loop helpers",
		Long: "Helpers for the half of the review loop that happens after a reviewer speaks.\n" +
			"GUARD-002 makes a green check mean reviewed; these answer whether anyone acted on it.",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	pr.AddCommand(newTriageQueueCmd())
	return pr
}

func newTriageQueueCmd() *cobra.Command {
	var repo, registry string
	cmd := &cobra.Command{
		Use:   "triage-queue",
		Short: "List open PRs whose reviewer output has not been dispositioned",
		Long: `List open pull requests carrying reviewer output newer than their last
recorded triage.

This exists because the loop had no wake-up. A workflow_run re-evaluates the
attestation gate and GitHub notifies the human, but no push channel reaches an
agent session — so the mechanism an agent can use is a query it runs at a
checkpoint, which is this one.

It LISTS. It never applies, comments, or merges. The judgement belongs to the
pr-review-triage skill, which disposes of each item under explicit human
confirmation; nothing here may erode that.

Exit status is 0 when the queue is empty and 1 otherwise — including when the
question could not be answered, which is deliberate: a queue that cannot be
computed must not read as an empty one. The message says which.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			reg, err := prtriage.LoadRegistry(registry)
			if err != nil {
				c.PrintErrln("pr triage-queue:", err)
				return err
			}
			args := []string{"pr", "list", "--state", "open", "--limit", "100",
				"--json", "number,title,url,comments"}
			if repo != "" {
				args = append(args, "--repo", repo)
			}
			out, err := exec.Command("gh", args...).Output()
			if err != nil {
				c.PrintErrln("pr triage-queue: gh pr list:", err)
				return fmt.Errorf("gh pr list: %w", err)
			}
			var wire []ghPR
			if err := json.Unmarshal(out, &wire); err != nil {
				c.PrintErrln("pr triage-queue: parse gh output:", err)
				return fmt.Errorf("parse gh output: %w", err)
			}

			prs := make([]prtriage.PR, 0, len(wire))
			for _, w := range wire {
				p := prtriage.PR{Number: w.Number, Title: w.Title, URL: w.URL}
				for _, cm := range w.Comments {
					p.Comments = append(p.Comments, prtriage.Comment{
						Author: cm.Author.Login, Body: cm.Body, CreatedAt: cm.CreatedAt,
					})
				}
				prs = append(prs, p)
			}

			pending := prtriage.Queue(prs, reg)
			if len(pending) == 0 {
				_, _ = fmt.Fprintln(c.OutOrStdout(), "[OK] no reviewer output is awaiting a disposition")
				return nil
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "%d pull request(s) awaiting a disposition:\n\n", len(pending))
			for _, st := range pending {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "  #%-5d %s\n         %s, %s\n         %s\n",
					st.PR.Number, st.PR.Title, st.Reason,
					st.At.Local().Format("2006-01-02 15:04"), st.PR.URL)
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(),
				"\nDisposition them with the pr-review-triage skill, which records its table\n"+
					"on the PR under %q. Nothing here applies anything.\n", reg.Triage.Marker)
			return errPending
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "owner/name (default: the current repository)")
	cmd.Flags().StringVar(&registry, "registry",
		filepath.Join("harness", "review-attestation.json"),
		"path to the reviewer registry")
	return cmd
}
