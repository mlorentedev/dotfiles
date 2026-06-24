package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
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
	cmd.AddCommand(newMemSessionStartCmd())
	return cmd
}

// newMemSessionStartCmd wires the agent-agnostic session-brief core (CLI-025 PR2a),
// ported from scripts/session-brief.sh. It renders the vault signals — detection,
// health, spec counts, lessons-staleness, baseline — via --format=stdout|markdown.
// File-based agents (opencode/agy/copilot) consume this out-of-band; the Claude
// SessionStart adapter (PR2b) wraps this same core in its additionalContext envelope.
func newMemSessionStartCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "session-start",
		Short: "Emit the agent-agnostic session-brief (vault signals) for the given --format",
		Long: "session-start renders the agnostic session-brief core — vault detection,\n" +
			"vault-health, active/archived spec counts, lessons-staleness, and vault-baseline\n" +
			"integrity — that file-based agents (opencode/agy/copilot) inject out-of-band.\n" +
			"The Claude SessionStart hook wraps this same core in its additionalContext\n" +
			"envelope. Ported from scripts/session-brief.sh (ADR-023, HARNESS-026).",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd := os.Getenv("SESSION_BRIEF_CWD")
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			brief := mem.Brief(mem.BriefOptions{
				Cwd:        cwd,
				ScriptsDir: memScriptsDir(),
				StaleDays:  14,
				Now:        time.Now(),
			})
			out, err := mem.RenderBrief(brief, format)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), out)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: stdout|markdown")
	return cmd
}

// memScriptsDir locates the scripts/ dir hosting the sibling vault-health.sh,
// resolved from the env-contract (DOTFILES_REPO_DIR) — the Go equivalent of
// session-brief.sh's $(dirname "$0") sibling lookup. "" when unresolved, which
// makes vaultHealth emit the same "not found" line the shell does.
func memScriptsDir() string {
	repo := env.ResolvePath("DOTFILES_REPO_DIR")
	if repo == "" {
		return ""
	}
	return filepath.Join(repo, "scripts")
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
