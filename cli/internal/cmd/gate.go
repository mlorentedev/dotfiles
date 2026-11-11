package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gate"
	"github.com/spf13/cobra"
)

// newGateCmd is `dotf gate`: the pull-request gates CI runs, read from an
// event file with no API call (CI-018, mlorentedev/ci-workflows CI-008).
func newGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Pull-request gates CI runs, read from an event file",
		Long: `Pull-request gates CI runs. Each reads the pull request from a
pull-request event file and makes no API call, so the same command judges a
pull request on GitHub and on Gitea. Refreshing the event file before a re-run
is the caller's job (the shared pr-checks workflow does it).

Exit status: 0 passes, 1 fails, 2 is a usage or environment error.`,
	}
	cmd.AddCommand(newGateKnowledgeCmd(), newGateSpecCmd())
	return cmd
}

type gateKnowledgeOpts struct {
	event, base, head, summary string
	explain                    bool
}

func newGateKnowledgeCmd() *cobra.Command {
	var o gateKnowledgeOpts
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Check the PR body's Knowledge section",
		Long: `Check that the pull request body has a "## Knowledge" section whose
Lesson, ADR and Runbook lines each name a path the pull request adds, changes
or renames under docs/lessons/, docs/adr/ or docs/runbooks/ (not _index.md), or
say "none: <reason>". A dependency bot's pull request labelled "dependencies"
is skipped.

The diff is git diff base...head in the current directory.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "gate knowledge takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if o.event == "" || o.base == "" || o.head == "" {
				return guardUsage(c, "gate knowledge: --event, --base and --head are required")
			}
			return runGateKnowledge(c.OutOrStdout(), c.ErrOrStderr(), o)
		},
	}
	cmd.Flags().StringVar(&o.event, "event", "", "pull-request event file (required)")
	cmd.Flags().StringVar(&o.base, "base", "", "base of the pull request, e.g. origin/main (required)")
	cmd.Flags().StringVar(&o.head, "head", "", "head of the pull request, e.g. HEAD (required)")
	cmd.Flags().StringVar(&o.summary, "summary", "", "append a Markdown section for the job summary to this file")
	cmd.Flags().BoolVar(&o.explain, "explain", false, "on failure, print the expected shape")
	return cmd
}

func runGateKnowledge(stdout, stderr io.Writer, o gateKnowledgeOpts) error {
	pr, err := gate.ReadEvent(o.event)
	if err != nil {
		if errors.Is(err, gate.ErrNoPR) {
			err = fmt.Errorf("no PR context: %w", err)
		}
		_, _ = fmt.Fprintf(stderr, "[ERROR] %v\n", err)
		return withExitCode(2, err)
	}
	if pr.IsDependencyBotPR() {
		_, _ = fmt.Fprintf(stdout, "[OK] knowledge-gate skipped: \"dependencies\" label on a bot-authored PR (%s)\n", pr.Author)
		return appendSummary(o.summary, "knowledge-gate", "skipped: a dependency bot's pull request", nil)
	}
	r, err := judgeKnowledgeBody(stderr, pr.Body, o.base, o.head)
	if err != nil {
		return err
	}
	if len(r.Problems) == 0 {
		_, _ = fmt.Fprintln(stdout, "[OK] knowledge-gate: every line is answered")
		_, _ = fmt.Fprintln(stdout, strings.Join(r.Summary, "\n"))
		return appendSummary(o.summary, "knowledge-gate", "passed", nil)
	}
	_, _ = fmt.Fprintf(stdout, "[FAIL] knowledge-gate: %d problem(s) in the PR body\n", len(r.Problems))
	for _, p := range r.Problems {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", p)
	}
	if o.explain {
		_, _ = fmt.Fprintln(stdout, gate.KnowledgeExplain)
	}
	if err := appendSummary(o.summary, "knowledge-gate", "failed", r.Problems); err != nil {
		return err
	}
	return withExitCode(1, errors.New("knowledge-gate failed"))
}

// judgeKnowledgeBody finds the section and judges it against the diff. A
// missing section is a problem, not an error; a range git cannot diff is an
// error, since the gate will not judge paths it cannot see.
func judgeKnowledgeBody(stderr io.Writer, body, base, head string) (gate.KnowledgeResult, error) {
	// CR first: a body saved from GitHub's web editor has CRLF line endings.
	text, open := gate.StripMarkup(strings.ReplaceAll(body, "\r", ""))
	section, found := gate.KnowledgeSection(text)
	if !found {
		p := `the PR body has no "## Knowledge" section`
		if open > 0 {
			p = fmt.Sprintf(`the PR body has no "## Knowledge" section outside code: the code fence opened at line %d is never closed, so everything after it is code`, open)
		}
		return gate.KnowledgeResult{Problems: []string{p}}, nil
	}
	changed, err := changedPaths(base, head)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "[ERROR] cannot diff %s...%s\n", base, head)
		_, _ = fmt.Fprintln(stderr, "[ERROR] The gate fails closed: it will not judge paths it cannot see.")
		return gate.KnowledgeResult{}, withExitCode(2, err)
	}
	return gate.JudgeKnowledge(section, changed), nil
}

// changedPaths is what the pull request adds, modifies or renames (under the
// new name), from the merge base.
func changedPaths(base, head string) ([]string, error) {
	// -z: a path is NUL-terminated and never quoted, spaces and non-ASCII included.
	out, err := exec.Command("git", "diff", "-z", "--name-only", "--diff-filter=AMR", "-M", base+"..."+head).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff %s...%s: %w", base, head, err)
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == 0 }), nil
}

// appendSummary appends a gate's section to the job summary file, when one
// was asked for.
func appendSummary(file, name, verdict string, problems []string) error {
	if file == "" {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### %s: %s\n\n", name, verdict)
	for _, p := range problems {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return withExitCode(2, fmt.Errorf("summary: %w", err))
	}
	defer func() { _ = f.Close() }()
	if _, err := io.WriteString(f, b.String()+"\n"); err != nil {
		return withExitCode(2, fmt.Errorf("summary: %w", err))
	}
	return nil
}
