package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/doctor"
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

// newMemSessionStartCmd wires the session-start noun (CLI-025), ported from
// session-brief.sh + claude-session-start.sh. It has two modes:
//
//   - `--format=stdout|markdown` renders the agent-agnostic session-brief core
//     (PR2a) for file-based agents (opencode/agy/copilot), out-of-band.
//   - no flag = the Claude SessionStart hook: read the hook JSON on stdin and emit
//     the additionalContext envelope (the agnostic core + the Claude-only injectors).
func newMemSessionStartCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "session-start",
		Short: "Session-start brief: --format for file-based agents, or the Claude hook envelope",
		Long: "session-start has two modes. With --format=stdout|markdown it renders the\n" +
			"agent-agnostic session-brief core (vault detection, health, specs, lessons,\n" +
			"baseline) that file-based agents (opencode/agy/copilot) inject out-of-band.\n" +
			"With no --format it is the Claude SessionStart hook: it reads the hook JSON on\n" +
			"stdin and emits the additionalContext envelope — the same core plus the\n" +
			"Claude-only injectors. Ported from session-brief.sh + claude-session-start.sh.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "" {
				return runSessionBrief(cmd, format)
			}
			return runClaudeHook(cmd)
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "stdout|markdown for the agnostic brief; omit for the Claude hook envelope")
	return cmd
}

// runSessionBrief renders the agnostic session-brief for a file-based agent.
func runSessionBrief(cmd *cobra.Command, format string) error {
	cwd := os.Getenv("SESSION_BRIEF_CWD")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	brief := mem.Brief(mem.BriefOptions{Cwd: cwd, ScriptsDir: memScriptsDir(), StaleDays: 14, Now: time.Now()})
	out, err := mem.RenderBrief(brief, format)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

// runClaudeHook reads the SessionStart hook JSON on stdin and emits the
// additionalContext envelope, resolving every path from the env-contract.
func runClaudeHook(cmd *cobra.Command) error {
	payload, _ := io.ReadAll(cmd.InOrStdin())
	cwd := cwdFromPayload(payload)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	home := env.Home()
	ctx := mem.ClaudeContext(mem.ClaudeContextInput{
		Cwd:          cwd,
		Vault:        vault.ResolveVault(),
		ScriptsDir:   memScriptsDir(),
		Home:         home,
		ContractPath: filepath.Join(env.DotfilesDir(home), "env-contract.json"),
		ClaudeJSON:   filepath.Join(home, ".claude", ".claude.json"),
		ConfigPath:   memConfigPath(),
		Now:          time.Now(),
		DoctorQuick: func() string {
			var buf bytes.Buffer
			_, _ = doctor.Run(doctor.Options{Quick: true, Out: &buf})
			return buf.String()
		},
	})
	out, err := mem.ClaudeEnvelope(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

// cwdFromPayload extracts the hook JSON's .cwd, or "" when absent/malformed.
func cwdFromPayload(payload []byte) string {
	var p struct {
		Cwd string `json:"cwd"`
	}
	_ = json.Unmarshal(payload, &p)
	return p.Cwd
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

// memConfigPath resolves session-start-config.json: the SESSION_START_CONFIG
// override, else <DOTFILES_REPO_DIR>/session-start-config.json (the shell's
// $SCRIPT_DIR/../session-start-config.json). "" falls back to historical defaults.
func memConfigPath() string {
	if c := os.Getenv("SESSION_START_CONFIG"); c != "" {
		return c
	}
	repo := env.ResolvePath("DOTFILES_REPO_DIR")
	if repo == "" {
		return ""
	}
	return filepath.Join(repo, "session-start-config.json")
}
