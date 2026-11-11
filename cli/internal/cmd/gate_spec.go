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

type gateSpecOpts struct {
	event, base, head  string
	adjacency, summary string
	threshold          int
	explain            bool
}

func newGateSpecCmd() *cobra.Command {
	var o gateSpecOpts
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Check that a large pull request carries a spec, and that it archives the specs it closes",
		Long: `Check the Discipline Gate on a pull request:

  - archive-on-merge: a pull request whose body or commits close an issue
    archives the active spec that tracks it, unless it carries the
    "skip-archive" label and a "## Archive skip rationale" section;
  - a production diff of --threshold lines or more touches an active spec
    folder by at least 10 lines, or archives the spec of an issue it closes,
    unless it carries "skip-sdd" and a "## SDD skip rationale" section.

A dependency bot's pull request labelled "dependencies" skips the second
check. The diff is git diff base...head in the current directory.

With --adjacency, it first lists the other open issues that name a production
file the pull request changes. The list is advisory and never changes the
exit status.`,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return guardUsage(c, "gate spec takes no arguments")
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if o.event == "" || o.base == "" || o.head == "" {
				return guardUsage(c, "gate spec: --event, --base and --head are required")
			}
			return runGateSpec(c.OutOrStdout(), c.ErrOrStderr(), o)
		},
	}
	cmd.Flags().StringVar(&o.event, "event", "", "pull-request event file (required)")
	cmd.Flags().StringVar(&o.base, "base", "", "base of the pull request, e.g. origin/main (required)")
	cmd.Flags().StringVar(&o.head, "head", "", "head of the pull request, e.g. HEAD (required)")
	cmd.Flags().IntVar(&o.threshold, "threshold", 50, "production lines from which a spec is required")
	cmd.Flags().BoolVar(&o.explain, "explain", false, "print the line count per file")
	cmd.Flags().StringVar(&o.adjacency, "adjacency", "", "open-issue feed, one \"<number><TAB><title and body>\" per line, for the advisory adjacency report")
	cmd.Flags().StringVar(&o.summary, "summary", "", "Markdown file to append the adjacency table to, e.g. $GITHUB_STEP_SUMMARY")
	return cmd
}

// specGate is one run of the gate: the options, the pull request, and where
// its verdict goes.
type specGate struct {
	o        gateSpecOpts
	pr       gate.PR
	slug     string
	out, err io.Writer
}

func runGateSpec(stdout, stderr io.Writer, o gateSpecOpts) error {
	pr, err := readGateEvent(stderr, o.event)
	if err != nil {
		return err
	}
	if exec.Command("git", "rev-parse", "--git-dir").Run() != nil {
		_, _ = fmt.Fprintln(stderr, "[ERROR] Not in a git repo")
		return withExitCode(2, errors.New("not in a git repo"))
	}
	g := &specGate{o: o, pr: pr, slug: repoSlug(), out: stdout, err: stderr}
	g.reportAdjacency()
	if err := g.archiveOnMerge(); err != nil {
		return err
	}
	if done, err := g.escapes(); done || err != nil {
		return err
	}
	return g.judgeDiff()
}

// readGateEvent reads the event, an event without a pull request being a
// wiring error.
func readGateEvent(stderr io.Writer, file string) (gate.PR, error) {
	pr, err := gate.ReadEvent(file)
	if err != nil {
		if errors.Is(err, gate.ErrNoPR) {
			err = fmt.Errorf("no PR context: %w", err)
		}
		_, _ = fmt.Fprintf(stderr, "[ERROR] %v\n", err)
		return gate.PR{}, withExitCode(2, err)
	}
	return pr, nil
}

// repoSlug is owner/name of this repository, to reject another repository's
// closing references: the runner's GITHUB_REPOSITORY, then origin. Empty
// means ownership cannot be proven, and qualified references are ignored.
func repoSlug() string {
	if s := os.Getenv("GITHUB_REPOSITORY"); s != "" {
		return s
	}
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSuffix(strings.TrimSpace(string(out)), ".git")
	i := strings.LastIndex(url, "github.com")
	if i < 0 || i+len("github.com") >= len(url) {
		return ""
	}
	rest := url[i+len("github.com"):]
	if rest[0] != ':' && rest[0] != '/' {
		return ""
	}
	return rest[1:]
}

// closingText is every text GitHub reads closing keywords from when the pull
// request merges: its body and its own commits' messages, which a squash
// merge carries into the merge commit (#1878).
func (g *specGate) closingText() (string, error) {
	log, err := exec.Command("git", "log", "--format=%B", g.o.base+".."+g.o.head).Output()
	if err != nil {
		return "", err
	}
	return g.pr.Body + "\n" + string(log), nil
}

func (g *specGate) closedIssues() ([]string, error) {
	text, err := g.closingText()
	if err != nil {
		return nil, err
	}
	return gate.SortedUnique(gate.ClosingIssueNumbers(g.slug, text)), nil
}

func (g *specGate) fail(code int, msg string) error {
	_, _ = io.WriteString(g.err, msg)
	return withExitCode(code, errors.New(strings.SplitN(strings.TrimSpace(msg), "\n", 2)[0]))
}
