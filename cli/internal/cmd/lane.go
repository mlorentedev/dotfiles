package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneExec starts the agent; a variable so a test records the argv instead of
// replacing the test process. laneRun is the seam over git, gh and herdr.
var (
	laneExec = lane.Exec
	laneRun  = lane.ExecRunner
)

func newLaneCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "lane",
		Short:        "Hand a work lane from one agent session to the next",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newLaneExecCmd(), newLaneNextCmd(), newLaneStatusCmd())
	return c
}

func newLaneExecCmd() *cobra.Command {
	var agent, promptFile string
	c := &cobra.Command{
		Use:   "exec",
		Short: "Start an agent on a prompt read from a file",
		Long: "exec replaces this process with the named agent's interactive command and\n" +
			"hands it the prompt as one argument, so no shell quoting is involved and the\n" +
			"same call works from zsh, bash and PowerShell. An agent dotf has no launch\n" +
			"command for is refused, naming the ones it has.",
		Example:      "  dotf lane exec --agent claude --prompt-file prompt.md",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			b, err := readPrompt(promptFile)
			if err != nil {
				return fmt.Errorf("lane exec: %w", err)
			}
			argv, err := lane.LaunchArgv(agent, strings.TrimSpace(string(b)))
			if err != nil {
				return err
			}
			return laneExec(argv)
		},
	}
	c.Flags().StringVar(&agent, "agent", "", "agent to start ("+strings.Join(lane.Agents(), ", ")+")")
	c.Flags().StringVar(&promptFile, "prompt-file", "", "file holding the prompt")
	_ = c.MarkFlagRequired("agent")
	_ = c.MarkFlagRequired("prompt-file")
	return c
}

// readPrompt reads the prompt file. The line `lane next` types into a pane
// names it relative to the pane's directory (lane.PromptFile), so a relative
// path that is not there is looked for at the worktree root too, in case the
// pane stands in a subdirectory.
func readPrompt(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err == nil || filepath.IsAbs(path) || !errors.Is(err, fs.ErrNotExist) {
		return b, err
	}
	top, gitErr := gitOut("rev-parse", "--show-toplevel")
	if gitErr != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(top, path))
}

func newLaneNextCmd() *cobra.Command {
	var agent string
	var dryRun bool
	c := &cobra.Command{
		Use:   "next",
		Short: "Start the next queued issue in a new worktree and session",
		Long: "next takes the first issue of the repo's queue (.dotf/lane.json) that is open\n" +
			"and has no assignee, creates its worktree off the base branch in the session\n" +
			"backend (herdr), and starts an agent on it with the repo's prompt template.\n" +
			"The agent is the one running in the calling pane unless --agent says\n" +
			"otherwise. It never removes a worktree, the caller's included, and a second\n" +
			"run for the same issue starts nothing. With no backend it prints the\n" +
			"commands to run instead, with the agent left for the person to pick when\n" +
			"--agent does not name one.",
		Example:      "  dotf lane next\n  dotf lane next --agent pi --dry-run",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return runLaneNext(c, agent, dryRun)
		},
	}
	c.Flags().StringVar(&agent, "agent", "", "agent to start (default: the calling pane's; "+strings.Join(lane.Agents(), ", ")+")")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "decide and print, change nothing")
	return c
}

func gitOut(args ...string) (string, error) {
	out, err := laneRun("git", args...)
	return strings.TrimSpace(out), err
}

func runLaneNext(c *cobra.Command, agent string, dryRun bool) error {
	top, err := gitOut("rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("lane next: not in a git checkout: %w", err)
	}
	common, err := gitOut("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("lane next: %w", err)
	}
	cfg, err := lane.LoadConfig(top)
	if err != nil {
		return err
	}
	tmpl, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(cfg.Template)))
	if err != nil {
		return fmt.Errorf("lane next: prompt template: %w", err)
	}
	parent := filepath.Dir(common)
	opts := lane.Options{
		Agent: agent, DryRun: dryRun, Config: cfg, Template: string(tmpl),
		Parent: parent, PromptDir: filepath.Join(common, "dotf-lane"), Dotf: selfBinary(c), GitDir: common,
		WorktreeRoot: env.ResolvePath("WORKTREE_ROOT"),
		Fetch: func() error {
			remote, branch := cfg.Remote()
			_, err := laneRun("git", "-C", parent, "fetch", "--quiet", remote, branch)
			return err
		},
	}
	var backend lane.Backend
	if h := (lane.Herdr{Run: laneRun}); h.Available() {
		backend = h
	}
	out, err := lane.Handoff(opts, backend, lane.GH{Run: laneRun})
	if err != nil {
		return err
	}
	printOutcome(c.OutOrStdout(), out)
	return nil
}

func printOutcome(w io.Writer, out lane.Outcome) {
	if out.Issue.Number != 0 {
		_, _ = fmt.Fprintf(w, "next: #%d %s -> %s (%s)\n", out.Issue.Number, out.Issue.Title, out.Branch, out.Agent)
	}
	if out.Note != "" {
		_, _ = fmt.Fprintln(w, out.Note)
	}
	for _, line := range out.Commands {
		_, _ = fmt.Fprintln(w, "  "+line)
	}
	if out.Started {
		_, _ = fmt.Fprintln(w, "started the successor")
	}
}

// selfBinary is the path of the running dotf, which the successor is told to
// run. An empty answer falls back to `dotf` on the pane's PATH, which may be a
// different release: say so rather than start it silently.
func selfBinary(c *cobra.Command) string {
	self, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(c.ErrOrStderr(), "lane next: cannot locate this binary (%v); the successor runs `dotf` from the pane's PATH\n", err)
		return ""
	}
	return self
}
