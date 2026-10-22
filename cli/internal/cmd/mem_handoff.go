package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// newMemHandoffWriteCmd replaces the "merge threads by hand" instruction with a
// mechanism (HARNESS-088, #1278).
//
// `harness/skills/handoff/SKILL.md` already tells a session to merge rather than
// overwrite. It was violated twice in one evening, once by each of two sessions,
// and neither noticed — last-writer-wins produces a well-formed file and a
// successful edit, so the failure is invisible until a later session follows a
// pointer into a block that no longer exists.
//
// One shared mutable slot with N concurrent writers is a data-structure problem.
// This command makes each session write only its own thread, so overwriting a
// peer stops being something to remember not to do.
func newMemHandoffWriteCmd() *cobra.Command {
	var w handoffWrite

	cmd := &cobra.Command{
		Use:   "handoff-write",
		Short: "Write this session's handoff thread, leaving other sessions' threads untouched",
		Long: `handoff-write replaces one thread's sub-block under "## Session Handoff",
reading the body from stdin. Every other thread is left byte-identical.

The body is stored in the thread's own file, threads/<thread>.md beside
MEMORY.md, and the sub-block becomes an entry derived from it: the Updated date,
the first line of the Next action, how many items await Manu, the Journal line and
a "[full thread](threads/<thread>.md)" pointer. A block an older release wrote
inline stays readable and is moved to this shape on its next write. A write that
would take MEMORY.md past 25,000 bytes, which is what a session start reads,
fails and names the oldest threads to archive. --dry-run prints MEMORY.md and
names the thread file on stderr, writing neither.

A thread is a LINE OF WORK, and git already names it: the branch. So the key
defaults to the current branch (e.g. feat-x), which means work resumed tomorrow on
another machine lands in the same thread. Ambient work on main/master is qualified
by host (main@msi), because two machines' main are not one line of work, and a
detached HEAD falls back to worktree@host rather than guessing.

Pass --thread to override — a session that switches branches mid-flight re-keys
otherwise, and the line of work may well be the one it started on.

Run from a repository that is not the project --memory belongs to (the vault
checkout writing a project's MEMORY.md), the current branch names no line of work
there, so handoff-write refuses without --thread and names the key it would have
used.

Pass --agent to name the writer. It is stamped into the heading as
"(writer: <agent>)", and a block another agent wrote under the same key is kept:
this write goes to <thread>+<agent> instead, and stderr names both agents and the
key. An unstamped block's writer is read from its Journal line; a block nothing
attributes is replaced as before. Without --agent nothing is stamped or forked.

Skills should call this instead of instructing an Edit: the merge is the part that
was being got wrong, and it belongs where it can be tested.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return w.run(cmd)
		},
	}

	cmd.Flags().StringVar(&w.memoryPath, "memory", "", "path to the project's MEMORY.md")
	cmd.Flags().StringVar(&w.thread, "thread", "", "thread key (default: this checkout's branch)")
	cmd.Flags().StringVar(&w.agent, "agent", "", "the agent writing, stamped into the heading; another agent's block is kept and the write goes to <thread>+<agent>")
	cmd.Flags().BoolVar(&w.dryRun, "dry-run", false, "print the result instead of writing it")
	return cmd
}

// handoffWrite holds the flags of one handoff-write run.
type handoffWrite struct {
	memoryPath string
	thread     string
	agent      string
	dryRun     bool
}

func (w *handoffWrite) run(cmd *cobra.Command) error {
	if w.memoryPath == "" {
		return fmt.Errorf("--memory is required (the project's MEMORY.md)")
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve the current directory: %w", err)
	}
	thread, err := mem.HandoffThread(w.thread, w.memoryPath, wd)
	if err != nil {
		return err
	}
	body, err := readHandoffBody(cmd)
	if err != nil {
		return err
	}

	// The lock spans read -> compute -> rename. Without it two writers at
	// the same moment each renamed their own update over the file and the
	// later one erased the other's thread, both exiting 0 (#1884). A dry
	// run writes nothing, so it does not wait on a writer.
	if !w.dryRun {
		unlock, err := mem.LockHandoffMemory(w.memoryPath)
		if err != nil {
			return err
		}
		defer unlock()
	}

	current, err := os.ReadFile(w.memoryPath) // #nosec G304 -- operator-supplied path
	if err != nil {
		return fmt.Errorf("read %s: %w", w.memoryPath, err)
	}
	res, err := mem.StoreThreadAs(string(current), w.memoryPath, thread, w.agent, body)
	if err != nil {
		return err
	}
	prev, err := mem.ReadThreadFile(res.File)
	if err != nil {
		return fmt.Errorf("read %s: %w", res.File, err)
	}
	if err := mem.CheckThreadFile(prev, res.Key); err != nil {
		return err
	}
	if err := mem.CheckMemoryBudget(string(current), res.Content); err != nil {
		return err
	}
	return w.publish(cmd, thread, string(current), string(prev), res)
}

// readHandoffBody reads the thread body from stdin and refuses an empty one.
func readHandoffBody(cmd *cobra.Command) (string, error) {
	body, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("read handoff body from stdin: %w", err)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty handoff body — refusing to blank a thread, which is the clobber this command exists to prevent")
	}
	for _, warning := range mem.ThreadWarnings(string(body)) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning    %s\n", warning)
	}
	return string(body), nil
}

// publish reports what the write did and, unless this is a dry run, writes the
// thread file and then MEMORY.md.
func (w *handoffWrite) publish(cmd *cobra.Command, thread, current, prev string, res mem.StoredThread) error {
	// The one outcome where the handoff is not where its writer asked,
	// so it is said every time, written or unchanged (#1690).
	if res.Kept != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "forked     thread %q is %s's, so this %s handoff went to %q\n",
			thread, res.Kept, w.agent, res.Key)
		if project, ok := mem.MemoryProject(w.memoryPath); ok {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "journal    sessions/%s\n",
				mem.JournalName(now().Format("2006-01-02"), project, w.agent, res.Key))
		}
	}
	if !res.Changed && prev == res.FileBody {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unchanged  thread %q already says this\n", res.Key)
		return nil
	}
	// A block nobody wrote this session moves, so say so (#1651). On stderr,
	// because under --dry-run stdout is the document itself.
	if legacy, ok := mem.LegacyThreadKey(current); ok {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "migrated   the un-threaded handoff block into thread %q\n", legacy)
	}
	if w.dryRun {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "thread     %s would hold %d bytes\n", res.File, len(res.FileBody))
		_, _ = fmt.Fprint(cmd.OutOrStdout(), res.Content)
		return nil
	}
	if err := w.write(res); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote      thread %q in %s\n", res.Key, res.File)
	return nil
}

// write replaces the thread file and then MEMORY.md. The file goes first: an
// entry pointing at a file not yet written is the one order that reads as lost,
// while a file ahead of its entry is the thread a reader follows anyway. A new
// thread file takes MEMORY.md's mode, which the store is a part of.
func (w *handoffWrite) write(res mem.StoredThread) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(w.memoryPath); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := mem.ReplaceFile(res.File, res.FileBody, mode); err != nil {
		return err
	}
	if !res.Changed {
		return nil
	}
	return mem.ReplaceFile(w.memoryPath, res.Content, mode)
}

// newMemThreadCmd prints this session's thread key and journal filename, so a
// skill can find its own record instead of guessing which `-2`/`-3` suffix was
// its.
func newMemThreadCmd() *cobra.Command {
	var (
		date    string
		project string
		agent   string
	)

	cmd := &cobra.Command{
		Use:   "thread",
		Short: "Print this session's handoff thread key and journal filename",
		Long: `thread answers "which session am I" from the working directory.

Journal files were named <date>-<project>-<agent>.md, so two concurrent sessions
on one day collided into -2 and -3 suffixes encoding nothing — six such files
exist across two days, and no session could derive its own. With the thread in
the key, it is derivable rather than remembered.

The key is the branch, so the same line of work resumed on another machine
resolves to the same journal. See handoff-write for the full rule.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			key, err := mem.ThreadKeyForCwd()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "thread   %s\n", key)
			if date != "" && project != "" && agent != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "journal  sessions/%s\n",
					mem.JournalName(date, project, agent, key))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "YYYY-MM-DD, to also print the journal filename")
	cmd.Flags().StringVar(&project, "project", "", "project slug")
	cmd.Flags().StringVar(&agent, "agent", "", "agent name")
	return cmd
}
