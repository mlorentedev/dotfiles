package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	var (
		memoryPath string
		thread     string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "handoff-write",
		Short: "Write this session's handoff thread, leaving other sessions' threads untouched",
		Long: `handoff-write replaces one thread's sub-block under "## Session Handoff",
reading the body from stdin. Every other thread is left byte-identical.

The thread key defaults to this working directory's worktree (e.g. wt-pi-harness),
because that is what distinguishes two concurrent sessions — not the agent and not
the date. A checkout that is not a worktree resolves to "main".

Skills should call this instead of instructing an Edit: the merge is the part that
was being got wrong, and it belongs where it can be tested.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if thread == "" {
				thread = mem.ThreadKeyForCwd()
			}
			if memoryPath == "" {
				return fmt.Errorf("--memory is required (the project's MEMORY.md)")
			}

			body, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read handoff body from stdin: %w", err)
			}
			if len(body) == 0 {
				return fmt.Errorf("empty handoff body — refusing to blank a thread, which is the clobber this command exists to prevent")
			}

			current, err := os.ReadFile(memoryPath) // #nosec G304 -- operator-supplied path
			if err != nil {
				return fmt.Errorf("read %s: %w", memoryPath, err)
			}

			updated, changed, err := mem.WriteThread(string(current), thread, string(body))
			if err != nil {
				return err
			}
			if !changed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unchanged  thread %q already says this\n", thread)
				return nil
			}
			if dryRun {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), updated)
				return nil
			}
			// Written through a temp file in the same directory: a half-written
			// MEMORY.md is the one outcome worse than a clobbered one, and the
			// file is read at the start of every session.
			tmp, err := os.CreateTemp(filepath.Dir(memoryPath), ".handoff-*")
			if err != nil {
				return fmt.Errorf("stage the write: %w", err)
			}
			tmpName := tmp.Name()
			if _, err := tmp.WriteString(updated); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmpName)
				return fmt.Errorf("stage the write: %w", err)
			}
			if err := tmp.Close(); err != nil {
				_ = os.Remove(tmpName)
				return fmt.Errorf("stage the write: %w", err)
			}
			if err := os.Rename(tmpName, memoryPath); err != nil {
				_ = os.Remove(tmpName)
				return fmt.Errorf("replace %s: %w", memoryPath, err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote      thread %q in %s\n", thread, memoryPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&memoryPath, "memory", "", "path to the project's MEMORY.md")
	cmd.Flags().StringVar(&thread, "thread", "", "thread key (default: this worktree)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the result instead of writing it")
	return cmd
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

Journal files were named <date>-<project>-<agent>.md, so two WORKTREES on one day
collided into -2 and -3 suffixes encoding nothing — six such files exist across
two days, and no session could derive its own. With the worktree in the key, it
is derivable rather than remembered.`,
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
