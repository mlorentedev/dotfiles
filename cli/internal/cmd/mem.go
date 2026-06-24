package cmd

import (
	"io"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/vault"
	"github.com/spf13/cobra"
)

// newMemCmd is the `dotf mem` noun: the session-start/end hook cluster, ported
// from the twin shell scripts so the SessionStart/SessionEnd hooks shrink to thin
// shims (CLI-025). session-end lands first; session-start follows once HARNESS-026
// is pinned.
func newMemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mem",
		Short: "Cross-agent memory session hooks (session-end / session-start)",
		Long: "mem hosts the Claude session hooks as one Go noun (ADR-014, MEMORY-001),\n" +
			"converging the drifting session-handoff.{sh,ps1} twins. The SessionEnd hook\n" +
			"becomes a thin `dotf mem session-end` shim.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newMemSessionEndCmd())
	return cmd
}

// newMemSessionEndCmd wires the SessionEnd hook. Per the resilience contract a
// session-end hook must NEVER crash a session, so it reads the payload, persists
// the handoff record best-effort, and ALWAYS exits 0 — every error is swallowed.
func newMemSessionEndCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-end",
		Short: "Archive the /handoff block into a durable session record (SessionEnd hook)",
		Long: "session-end reads the SessionEnd hook JSON on stdin and archives the\n" +
			"`## Session Handoff` block from the project's MEMORY.md into\n" +
			"<vault>/10_projects/<project>/sessions/<date>-<project>-claude.md. Every\n" +
			"trivial / missing / malformed input is a clean no-op; it never errors.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, _ := io.ReadAll(cmd.InOrStdin())
			// Best-effort by contract: a SessionEnd hook must never crash a
			// session, so the write result is intentionally discarded — exit 0.
			_, _ = mem.SessionEnd(payload, vault.ResolveVault(), time.Now().UTC())
			return nil
		},
	}
}
