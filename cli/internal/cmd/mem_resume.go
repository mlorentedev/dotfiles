package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// newMemResumeCmd renders this session's handoff thread, so the next session
// reads a render of the durable record instead of a prompt typed by hand
// (MEMORY-008).
func newMemResumeCmd() *cobra.Command {
	var memoryPath, thread, agent, format string
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Render this checkout's handoff thread for the next session",
		Long: `resume renders one handoff thread: the thread file a MEMORY.md entry points at,
or the inline block an older release wrote. The output depends only on that
text, so two runs print the same bytes.

--format prompt (the default) is the paste-ready instruction a session used to
type by hand; --format context is the block a session-start hook injects. Both
put the Next action, the Verify commands and what awaits Manu first. A Verify
command outside a read-only allowlist, or one that chains, pipes or redirects,
is marked unverified: the file was written by another session.

The thread is this checkout's (see handoff-write), or this agent's fork of it.
When the working directory names no single thread for this agent, resume picks
none and lists every thread, each with the --thread flag that selects it.

--memory defaults to the vault memory of the checkout's project.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMemResume(cmd, memoryPath, thread, agent, format)
		},
	}
	cmd.Flags().StringVar(&memoryPath, "memory", "", "path to the project's MEMORY.md (default: the checkout's vault memory)")
	cmd.Flags().StringVar(&thread, "thread", "", "thread key (default: this checkout's branch)")
	cmd.Flags().StringVar(&agent, "agent", "", "the agent resuming; its fork of the thread is preferred, and another agent's thread is not chosen")
	cmd.Flags().StringVar(&format, "format", "prompt", "prompt|context")
	return cmd
}

func runMemResume(cmd *cobra.Command, memoryPath, thread, agent, format string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve the current directory: %w", err)
	}
	if memoryPath == "" {
		if memoryPath, err = defaultMemoryPath(wd); err != nil {
			return err
		}
	}
	key, err := mem.HandoffThread(thread, memoryPath, wd)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(memoryPath) // #nosec G304 -- operator-supplied or vault-derived path
	if err != nil {
		return fmt.Errorf("read %s: %w", memoryPath, err)
	}
	chosen, candidates := mem.SelectThread(string(content), key, agent)
	if chosen == "" {
		var carry string
		if cmd.Flags().Changed("memory") {
			carry += " --memory " + shellQuote(memoryPath)
		}
		if agent != "" {
			carry += " --agent " + agent
		}
		return listCandidates(cmd, key, agent, carry, candidates)
	}
	text, source := mem.ThreadText(string(content), chosen, memoryPath)
	out, err := mem.RenderResume(text, source, format)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

// defaultMemoryPath is the vault MEMORY.md of the project the checkout belongs
// to, resolved the way session-end resolves it.
func defaultMemoryPath(wd string) (string, error) {
	v := vault.ResolveVault()
	if v == "" {
		return "", fmt.Errorf("no vault resolved; pass --memory")
	}
	p, _ := mem.ProjectMemory(v, wd)
	return p, nil
}

// listCandidates says no thread was chosen and lists the ones there are. Not an
// error: a list is the answer when the working directory names no single thread.
func listCandidates(cmd *cobra.Command, key, agent, carry string, candidates []string) error {
	who := "this session"
	if agent != "" {
		who = agent
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "No single thread for %s here: %q is absent or another agent's. Threads:\n", who, key)
	if len(candidates) == 0 {
		_, _ = fmt.Fprintln(out, "  (none)")
	}
	for _, c := range candidates {
		_, _ = fmt.Fprintf(out, "  dotf mem resume --thread %s%s\n", c, carry)
	}
	return nil
}

// shellQuote quotes a path for a command the user copies: a single-quoted
// word, with any single quote inside it closed, escaped and reopened.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
