package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// errAuditAttention is returned (silently) when the audit found a zombie, an
// unresolvable link, or a lookup nobody could answer. Same exit contract as
// errPending: non-zero both when work is pending and when the question could
// not be answered, and the printed report says which.
var errAuditAttention = errors.New("spec audit: at least one active spec needs attention")

// specAuditTimeout bounds one gh call, so a hung network degrades to an
// unanswerable finding instead of a hung audit.
const specAuditTimeout = 20 * time.Second

// specAuditRun runs gh with a deadline. A package var so tests never reach
// the network.
var specAuditRun = func(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specAuditTimeout)
	defer cancel()
	var out, errOut bytes.Buffer
	c := exec.CommandContext(ctx, "gh", args...)
	c.Stdout, c.Stderr = &out, &errOut
	err := c.Run()
	return out.String(), errOut.String(), err
}

func newSpecAuditCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Check every active spec against the state of the issue it tracks",
		Long: `Grade every active spec (specs/*/, never specs/archive/) of the current
checkout by the state of the issue its proposal.md tracks.

This exists because archive-on-merge only sees an issue closed by a PR's
closing keyword. An issue closed any other way leaves its spec active forever
(#1087), and only asking the forge can find it.

  FAIL          the issue is CLOSED (a zombie), or the link resolves to no
                issue, to a pull request, or is malformed
  WARN          the spec is linked only in prose, or not at all
  UNANSWERABLE  the forge could not be asked (auth, network, rate limit)

Exit status is 0 only when nothing is FAIL or UNANSWERABLE: a question that
could not be answered must not read as a clean audit.`,
		Example:       "  dotf spec audit\n  dotf spec audit --repo mlorentedev/hive",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}
			home := repo
			if home == "" {
				home, _ = initrepo.OriginRepo(root)
			}
			if !initrepo.ValidRepoSlug(home) {
				err := fmt.Errorf("spec audit: cannot tell which repository owns these specs (no origin remote in %s); pass --repo owner/name", root)
				c.PrintErrln(err)
				return err
			}
			findings, err := spec.AuditIssueState(root, home, spec.GHIssueStateLookup(specAuditRun))
			if err != nil {
				c.PrintErrln("spec audit:", err)
				return err
			}
			printSpecAudit(c.OutOrStdout(), findings)
			if spec.AuditNeedsAttention(findings) {
				return errAuditAttention
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "owner/name that owns these specs (default: the origin remote)")
	return cmd
}

func printSpecAudit(w io.Writer, findings []spec.AuditFinding) {
	counts := map[spec.AuditSeverity]int{}
	for _, f := range findings {
		counts[f.Severity]++
		if f.Severity != spec.SeverityOK {
			_, _ = fmt.Fprintf(w, "  [%s] %s: %s\n", strings.ToUpper(string(f.Severity)), f.SpecID, f.Detail)
		}
	}
	if len(findings) > 0 && counts[spec.SeverityOK] == len(findings) {
		_, _ = fmt.Fprintf(w, "[OK] %d active spec(s), every one tracking an open issue\n", len(findings))
		return
	}
	_, _ = fmt.Fprintf(w, "\n%d active spec(s): %d ok, %d warn, %d fail, %d unanswerable\n",
		len(findings), counts[spec.SeverityOK], counts[spec.SeverityWarn],
		counts[spec.SeverityFail], counts[spec.SeverityUnanswerable])
}
