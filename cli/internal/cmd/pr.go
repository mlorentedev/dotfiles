package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
	"github.com/spf13/cobra"
)

// errPending is returned (silently) when at least one pull request is awaiting
// a disposition. main.go maps any non-nil error to exit status 1, which is the
// whole exit contract this CLI has — so a caller distinguishes "pending" from
// "could not answer" by reading the message, not the code. Inventing a second
// code here would mean changing main.go's contract for one subcommand.
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
			reg, err := prtriage.LoadRegistry(reviewerRegistryPath(c, registry))
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
// reads the reviewer registry. Read it through reviewerRegistryPath.
func addReviewerRegistryFlag(cmd *cobra.Command, registry *string) {
	cmd.Flags().StringVar(registry, "registry", filepath.Join("harness", "review-attestation.json"),
		"path to the reviewer registry (default: under the root of the current checkout; "+
			"for a --repo other than this checkout's, pass that repository's registry)")
}

// reviewerRegistryPath resolves --registry. A path given on the command line
// is used as given; the default is the current checkout's, found from any
// directory in it. Outside a checkout the default stays relative, and reading
// it fails with the path named.
func reviewerRegistryPath(c *cobra.Command, registry string) string {
	if c.Flags().Changed("registry") {
		return registry
	}
	root, err := worktree.ResolveWorktreeRoot(".")
	if err != nil {
		return registry
	}
	return filepath.Join(root, registry)
}
