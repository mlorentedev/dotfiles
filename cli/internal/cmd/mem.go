package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/doctor"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
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
		Long: "mem hosts the agent session hooks and the MEMORY.md handoff commands.\n" +
			"The SessionEnd hook is a thin `dotf mem session-end` call.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newMemSessionEndCmd())
	cmd.AddCommand(newMemSessionStartCmd())
	cmd.AddCommand(newMemProjectKeyCmd())
	cmd.AddCommand(newMemHandoffWriteCmd())
	cmd.AddCommand(newMemThreadCmd())
	return cmd
}

// newMemProjectKeyCmd exposes memlink.ClaudeProjectKey as a CLI so a script
// obtains the Claude auto-memory key from the one Go implementation instead of
// re-deriving it — the datum-duplication that drifted and mis-encoded the
// junction on Windows (BUG-031/#689; #551 fixed only the Go side). Prints the key for <path> and a
// trailing newline.
func newMemProjectKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project-key <path>",
		Short: "Print Claude Code's per-project auto-memory key for a working directory",
		Long: "project-key encodes a working directory into Claude Code's per-project key\n" +
			"(the directory name under ~/.claude/projects) — every '/', '\\' and drive ':'\n" +
			"maps to '-'. Setup and crystallize call it on every OS, so the directory\n" +
			"they link can never drift from the key Claude Code uses.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), memlink.ClaudeProjectKey(args[0]))
			return err
		},
	}
}

// newMemSessionEndCmd wires the SessionEnd hook. Per the resilience contract a
// session-end hook must NEVER crash a session, so it reads the payload, persists
// the handoff record best-effort, and ALWAYS exits 0 — an error is reported on
// stderr, never returned.
func newMemSessionEndCmd() *cobra.Command {
	agent := mem.DefaultSessionEndAgent
	cmd := &cobra.Command{
		Use:   "session-end",
		Short: "Archive the /handoff block into a durable session record (SessionEnd hook)",
		Long: "session-end reads the SessionEnd hook JSON on stdin and archives this\n" +
			"agent's thread of the `## Session Handoff` block from the project's MEMORY.md\n" +
			"into <vault>/10_projects/<project>/sessions/<date>-<project>-<agent>-<thread>.md,\n" +
			"unless a journal is already there. --agent names the harness (default claude,\n" +
			"whose payload names none). Every trivial / missing / malformed input is a\n" +
			"clean no-op, and it always exits 0.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, _ := io.ReadAll(cmd.InOrStdin())
			// Best-effort by contract: a SessionEnd hook must never crash a
			// session, so no error becomes the exit status — exit 0.
			//
			// Local time, not UTC: `now` is formatted down to a calendar date
			// that becomes the record's filename and its human-facing heading,
			// so it must be the date the operator actually worked (CLI-043).
			// UTC filed every post-18:00 session in America/Denver under the
			// next day and collided with the following morning's record.
			// session-start below is already local; this keeps the pair consistent.
			//
			// Every error SessionEndAs returns is printed: a refused --agent (the
			// hook's own configuration) and a failed write alike. Swallowing
			// either would leave the session without a record and nothing saying
			// why. Trivial, missing and malformed input are no-ops, not errors.
			if _, err := mem.SessionEndAs(payload, vault.ResolveVault(), agent, time.Now()); err != nil {
				cmd.PrintErrf("dotf mem session-end: %v\n", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agent, "agent", agent, "the harness ending the session; names the fallback journal and selects its thread block")
	return cmd
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
	brief := mem.Brief(mem.BriefOptions{
		Cwd: cwd, VaultHealth: memVaultHealth, StaleDays: 14, Now: time.Now(),
		// The file-based agents get the same probe as the Claude hook. Wiring it
		// on one path only would rebuild the asymmetry this CLI exists to remove:
		// opencode, agy and copilot read this brief, and a triage loop that only
		// closes in one harness is not closed.
		TriageQueue: memTriageQueue(cwd),
	})
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
		VaultHealth:  memVaultHealth,
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
		TriageQueue: memTriageQueue(cwd),
	})
	out, err := mem.ClaudeEnvelope(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

// memTriageQueue builds the session-start triage probe for cwd, or nil when this
// repository does not run the review-triage loop at all.
//
// The nil case is load-bearing and is NOT the same as an empty queue. Most repos
// carry no reviewer registry; reporting "could not be computed" in every one of
// them would train the reader to skim past the line, and the line only has value
// while it is rare. A missing registry means "not applicable" and stays silent;
// a registry that is present and unanswerable is reported loudly.
//
// The gh call is bounded at five seconds. Session start is a latency budget
// nobody volunteered for, and an unreachable API must degrade to one visible
// message rather than a hung shell.
func memTriageQueue(cwd string) func() (string, error) {
	registry := filepath.Join(cwd, "harness", "review-attestation.json")
	if _, err := os.Stat(registry); err != nil {
		return nil
	}
	return func() (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pending, err := prtriage.Fetch(ctx, "", registry)
		if err != nil {
			return "", err
		}
		if len(pending) == 0 {
			return "", nil
		}
		refs := make([]string, 0, len(pending))
		for _, st := range pending {
			refs = append(refs, fmt.Sprintf("#%d", st.PR.Number))
		}
		return strings.Join(refs, ", "), nil
	}
}

// cwdFromPayload extracts the hook JSON's .cwd, or "" when absent/malformed.
func cwdFromPayload(payload []byte) string {
	var p struct {
		Cwd string `json:"cwd"`
	}
	_ = json.Unmarshal(payload, &p)
	return p.Cwd
}

// memRepoDir resolves the dotfiles checkout for mem's sibling-script and config
// lookups: the ADR-025 cascade value when it names a real directory, else the
// .git walk-up (env.RepoDir). "" when neither resolves, so callers emit the same
// "not found" line the shell twin does — instead of probing the phantom contract
// default ~/Projects/dotfiles, which reads as "run setup" even after setup ran
// (#696).
func memRepoDir() string {
	if r := env.ResolvePath("DOTFILES_REPO_DIR"); r != "" && dirExists(r) {
		return r
	}
	return env.RepoDir()
}

// memVaultHealth is the session brief's vault health section: vault.RunHealth
// in-process, resolved as `dotf vault health` resolves it (healthOptions), with
// the vault detected from the session's cwd taking precedence. It replaced a
// bash run of scripts/vault-health.sh (CLI-023), so the banner and the
// command now report from the same code.
func memVaultHealth(w io.Writer, vaultDir, vaultName string) (int, error) {
	opts := healthOptions(vaultName, false)
	if vaultDir != "" {
		opts.VaultDir = vaultDir
	}
	return vault.RunHealth(w, opts)
}

// memConfigPath resolves session-start-config.json: the SESSION_START_CONFIG
// override, else <checkout>/session-start-config.json (the shell's
// $SCRIPT_DIR/../session-start-config.json). "" falls back to historical defaults.
func memConfigPath() string {
	if c := os.Getenv("SESSION_START_CONFIG"); c != "" {
		return c
	}
	repo := memRepoDir()
	if repo == "" {
		return ""
	}
	return filepath.Join(repo, "session-start-config.json")
}
