package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
	"github.com/spf13/cobra"
)

// errPending is returned (silently) when at least one pull request is awaiting
// a disposition. It carries no exit code of its own, so main.go exits 1 for it,
// the same status as "could not answer": a caller tells the two apart by reading
// the message. A command that needs another status tags its error with
// withExitCode (see exit.go), as `pr route` does for usage errors.
var errPending = errors.New("pr triage-queue: reviewer output is awaiting a disposition")

// triageFetch and triageFetchPR are the queue's two questions; variables so a
// test drives the exit status without GitHub.
var (
	triageFetch   = prtriage.FetchWithRegistry
	triageFetchPR = prtriage.FetchPR
)

func newPrCmd() *cobra.Command {
	pr := &cobra.Command{
		Use:   "pr",
		Short: "Pull-request review loop helpers",
		Long: "Helpers for the half of the review loop that happens after a reviewer speaks.\n" +
			"A green review check says a review ran; these answer whether anyone acted on it.",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	pr.AddCommand(newTriageQueueCmd())
	pr.AddCommand(newPrLandCmd())
	pr.AddCommand(newPrReproduceCmd())
	pr.AddCommand(newPrRouteCmd())
	pr.AddCommand(newPrPreflightCmd())
	pr.AddCommand(newPrGuardCmd())
	pr.AddCommand(newPrGateCmd())
	pr.AddCommand(newPrReviewableCmd())
	return pr
}

func newTriageQueueCmd() *cobra.Command {
	var repo, registry string
	var pr int
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
computed must not read as an empty one. The message says which.

--pr N asks about one pull request only, so another session's untriaged PR
does not hold up "triage-queue --pr N && pr land N". It exits 0 when N has
nothing awaiting a disposition, whatever the other PRs hold, and 1 when N
cannot be read or is not open.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			reg, err := reviewerRegistry(repo, registry)(c.Context())
			if err != nil {
				c.PrintErrln("pr triage-queue:", err)
				return err
			}
			pending, err := fetchTriageQueue(c, repo, reg, pr)
			if err != nil {
				c.PrintErrln("pr triage-queue:", err)
				return err
			}
			marker := reg.Triage.Marker
			if len(pending) == 0 {
				_, _ = fmt.Fprintln(c.OutOrStdout(), "[OK] no reviewer output is awaiting a disposition")
				return nil
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "%d pull request(s) awaiting a disposition:\n\n", len(pending))
			for _, st := range pending {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "  #%-5d %s\n         %s, %s\n         %s\n",
					st.PR.Number, st.PR.Title, st.Reason,
					st.At.Local().Format("2006-01-02 15:04"), st.PR.URL)
				if line := findingsLine(st); line != "" {
					_, _ = fmt.Fprintf(c.OutOrStdout(), "         %s\n", line)
				}
				for _, u := range st.Unreproduced {
					_, _ = fmt.Fprintf(c.OutOrStdout(), "         REAL %s:%d: %s\n", u.Finding.Path, u.Finding.Line, u.Problem)
				}
			}
			if len(reg.Classification.Reviewers) == 0 {
				_, _ = fmt.Fprintln(c.OutOrStdout(),
					"\nNote: finding classes were not checked: the registry names no classifying reviewer (classification.reviewers).")
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(),
				"\nDisposition them with the pr-review-triage skill, which records its table\n"+
					"on the PR under %q. Nothing here applies anything.\n", marker)
			return errPending
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "owner/name (default: the current repository)")
	cmd.Flags().IntVar(&pr, "pr", 0, "only this pull request's reviewer output decides the exit status")
	addReviewerRegistryFlag(cmd, &registry)
	return cmd
}

// findingsLine names a pending PR's inline findings that carry no declared
// class (TOOL-017 AC3), or says they could not be read. Empty when neither.
func findingsLine(st prtriage.Status) string {
	if st.FindingsUnread {
		return "inline findings could not be read: their classes are unknown"
	}
	if len(st.Unclassified) == 0 {
		return ""
	}
	const shown = 5
	where := make([]string, 0, shown)
	for _, f := range st.Unclassified {
		if len(where) == shown {
			where = append(where, fmt.Sprintf("and %d more", len(st.Unclassified)-shown))
			break
		}
		if f.Line > 0 {
			where = append(where, fmt.Sprintf("%s:%d", f.Path, f.Line))
		} else {
			where = append(where, f.Path)
		}
	}
	return fmt.Sprintf("%d finding(s) without a declared class: %s", len(st.Unclassified), strings.Join(where, ", "))
}

// fetchTriageQueue asks the repo-wide question, or with --pr the one-PR one.
func fetchTriageQueue(c *cobra.Command, repo string, reg prtriage.Registry, pr int) ([]prtriage.Status, error) {
	if !c.Flags().Changed("pr") {
		return triageFetch(c.Context(), repo, reg)
	}
	if pr <= 0 {
		return nil, fmt.Errorf("--pr takes a pull request number, got %d", pr)
	}
	return triageFetchPR(c.Context(), repo, reg, pr)
}

// addReviewerRegistryFlag declares --registry, shared by every pr command that
// reads the reviewer registry. Read it through reviewerRegistry.
func addReviewerRegistryFlag(cmd *cobra.Command, registry *string) {
	cmd.Flags().StringVar(registry, "registry", "",
		"path to a reviewer registry file (default: "+prtriage.RegistryPath+
			" on the default branch of --repo, read through the GitHub API)")
}

// registryLoader reads the reviewer registry when a command needs it.
type registryLoader func(context.Context) (prtriage.Registry, error)

// reviewerRegistry resolves --registry: the file it names, else the named
// repository's own registry from its default branch. Never the checkout the
// command runs in: that may belong to another repository, and on a branch that
// edits the registry it would let a PR change the check it is held to, which
// is why pr land reads the merge grant from the base too.
func reviewerRegistry(repo, registry string) registryLoader {
	if registry != "" {
		return func(context.Context) (prtriage.Registry, error) { return prtriage.LoadRegistry(registry) }
	}
	return func(ctx context.Context) (prtriage.Registry, error) { return prtriage.FetchRegistry(ctx, repo) }
}
