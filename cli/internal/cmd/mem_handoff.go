package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
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
		unlock, err := lockMemoryFile(w.memoryPath)
		if err != nil {
			return err
		}
		defer unlock()
	}

	current, err := os.ReadFile(w.memoryPath) // #nosec G304 -- operator-supplied path
	if err != nil {
		return fmt.Errorf("read %s: %w", w.memoryPath, err)
	}
	res, err := mem.WriteThreadAs(string(current), thread, w.agent, body)
	if err != nil {
		return err
	}
	return w.publish(cmd, thread, string(current), res)
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

// publish reports what the write did and, unless this is a dry run, replaces
// the file.
func (w *handoffWrite) publish(cmd *cobra.Command, thread, current string, res mem.ThreadWrite) error {
	// The one outcome where the handoff is not where its writer asked,
	// so it is said every time, written or unchanged (#1690).
	if res.Kept != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "forked     thread %q is %s's, so this %s handoff went to %q\n",
			thread, res.Kept, w.agent, res.Key)
	}
	if !res.Changed {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unchanged  thread %q already says this\n", res.Key)
		return nil
	}
	// A block nobody wrote this session moves, so say so (#1651). On stderr,
	// because under --dry-run stdout is the document itself.
	if legacy, ok := mem.LegacyThreadKey(current); ok {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "migrated   the un-threaded handoff block into thread %q\n", legacy)
	}
	if w.dryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), res.Content)
		return nil
	}
	if err := replaceMemoryFile(w.memoryPath, res.Content); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote      thread %q in %s\n", res.Key, w.memoryPath)
	return nil
}

// replaceMemoryFile writes content over path through a temp file in the same
// directory: a half-written MEMORY.md is the one outcome worse than a clobbered
// one, and the file is read at the start of every session.
//
// The file keeps the mode it had. os.CreateTemp makes the temp file 0600, and
// renaming it over MEMORY.md used to narrow every file this command wrote; a
// file shared with other tools is not ours to re-permission, the rule harness
// bind follows for settings files.
func replaceMemoryFile(path, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".handoff-*")
	if err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // a no-op once the rename succeeds
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("stage the write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	if fi, err := os.Stat(path); err == nil {
		if err := os.Chmod(tmpName, fi.Mode().Perm()); err != nil {
			return fmt.Errorf("stage the write: %w", err)
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
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
			key := mem.ThreadKeyForCwd()
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

// memoryLockWait bounds how long a writer waits for another. A write holds the
// lock for milliseconds, so ten seconds means the holder is stuck, and saying so
// beats waiting forever.
const memoryLockWait = 10 * time.Second

// lockMemoryFile takes the cross-process lock for one MEMORY.md.
//
// The lock file lives outside the vault, in the runtime or cache directory: a
// file beside MEMORY.md would be committed by the vault's auto-commit. Its name
// is a hash of the file's canonical path, because one MEMORY.md is reached by
// two paths (the vault one, and the symlink or junction under
// ~/.claude/projects/<key>/memory), and two writers holding different paths to
// the same file must still meet at one lock. EvalSymlinks resolves the Unix
// symlink; whether it resolves a Windows junction is not yet measured.
func lockMemoryFile(memoryPath string) (func(), error) {
	return lockMemoryFileWithin(memoryPath, memoryLockWait)
}

func lockMemoryFileWithin(memoryPath string, wait time.Duration) (func(), error) {
	canon, err := filepath.Abs(memoryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", memoryPath, err)
	}
	if real, err := filepath.EvalSymlinks(canon); err == nil {
		canon = real
	}
	if runtime.GOOS == "windows" {
		canon = strings.ToLower(canon) // one file, whatever case each writer typed
	}
	dir, err := memoryLockDir()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(canon))
	unlock, err := filelock.Lock(filepath.Join(dir, "handoff-"+hex.EncodeToString(sum[:8])+".lock"), wait)
	if err != nil {
		return nil, fmt.Errorf("another handoff-write is still writing %s: %w", memoryPath, err)
	}
	return unlock, nil
}

// memoryLockDir is where handoff locks live: the runtime directory on Linux,
// tmpfs and cleared at boot, else the user cache directory. The kernel releases
// the lock either way, so the choice is about keeping files out of the vault,
// not about correctness.
func memoryLockDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("no runtime or cache directory for the handoff lock: %w", err)
		}
		base = cache
	}
	dir := filepath.Join(base, "dotf", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create the handoff lock directory %s: %w", dir, err)
	}
	return dir, nil
}
