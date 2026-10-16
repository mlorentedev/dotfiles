package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// laneSources builds what status reads; a variable so a test answers from a
// fixture instead of git, gh and herdr.
var laneSources = realLaneSources

func newLaneStatusCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "status",
		Short: "Show every lane and the phase it is in",
		Long: "status joins git worktrees, their leases, the forge (REST, one request per\n" +
			"branch), the sessions that are live and herdr into one row per lane, and\n" +
			"derives each lane's phase: Pending, Running, Review, Landing, Landed, Reaped,\n" +
			"Stalled, Orphan or Unadopted. Phases are computed on every call and never\n" +
			"stored. It changes nothing. herdr not running is a supported state: the rows\n" +
			"say herdr: absent and keep their phase.",
		Example:       "  dotf lane status\n  dotf lane status --json",
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
			if asJSON {
				return writeLaneJSON(c.OutOrStdout(), rows)
			}
			return writeLaneTable(c.OutOrStdout(), rows)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "output in JSON format")
	return c
}

// orDash fills a cell a lane has no value for, such as the agent of an
// unadopted worktree or the liveness of a lane with no worktree.
func orDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func writeLaneJSON(w io.Writer, rows []lane.Row) error {
	if rows == nil {
		rows = []lane.Row{} // [] and not null, for a consumer that indexes it
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeLaneTable(w io.Writer, rows []lane.Row) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no lanes")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "PHASE\tTICKET\tBRANCH\tPR\tAGENT\tLIVE\tHERDR")
	for _, r := range rows {
		pr, ticket := "-", "-"
		if r.PR != 0 {
			pr = fmt.Sprintf("#%d %s", r.PR, r.PRState)
		}
		if r.Ticket != 0 {
			ticket = fmt.Sprintf("#%d", r.Ticket)
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.Phase, ticket, r.Branch, pr, orDash(r.Agent), orDash(string(r.Live)), r.Herdr)
	}
	return tw.Flush()
}

// realLaneSources wires status to the repository the command runs in.
func realLaneSources() (lane.Sources, error) {
	common, err := gitOut("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return lane.Sources{}, fmt.Errorf("lane status: not in a git checkout: %w", err)
	}
	parent := filepath.Dir(common)
	src := lane.Sources{Now: time.Now(), Repo: parent, BranchPrefix: "fix"}
	cfg, err := lane.LoadConfig(parent)
	switch {
	case err == nil:
		src.Queue, src.BranchPrefix = cfg.Queue, cfg.BranchPrefix
		src.Issues = lane.GH{Run: laneRun}
	case !errors.Is(err, fs.ErrNotExist):
		return lane.Sources{}, err
	}
	forge := lane.CachedForge(lane.REST{Run: laneRun})
	src.Forge = forge
	src.List = func() ([]worktree.Info, error) {
		return worktree.ListWithRunner(parent, lane.ForgeGit{RealGitRunner: &worktree.RealGitRunner{}, Forge: forge}, src.Now)
	}
	src.Live = func(path string) lane.Liveness {
		return lane.Combine(processLiveness(path), sessionLiveness(path, src.Now))
	}
	src.Herdr = lane.Herdr{Run: laneRun}
	return src, nil
}

// processLiveness is Gate f's reading: a process with its cwd in the worktree.
// A platform with no scan answers unknown, never idle.
func processLiveness(path string) lane.Liveness {
	r, supported := worktree.ProcessInside(path)
	switch {
	case !supported:
		return lane.LiveUnknown
	case r.Inside:
		return lane.LiveYes
	}
	return lane.LiveIdle
}

// sessionLiveness is the transcript half. It reads Claude Code's store only
// until RecentHarnessSession (#2575, BUG-148) lands, which adds pi and Copilot;
// swap it here, behind this one function. An unreadable store is unknown.
func sessionLiveness(path string, now time.Time) lane.Liveness {
	_, live, err := worktree.RecentSession(claudeProjectsDir(), path, now, worktree.SessionWindow)
	switch {
	case err != nil:
		return lane.LiveUnknown
	case live:
		return lane.LiveYes
	}
	return lane.LiveIdle
}
