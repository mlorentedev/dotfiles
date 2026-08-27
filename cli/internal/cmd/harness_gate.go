package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// gateExitBlock is the exit code every harness reads as "stop this tool call".
// Claude documents 2 for a blocking hook; the emitted pi and opencode wrappers
// translate it into their own primitive.
const gateExitBlock = 2

// newHarnessGateCmd is the binding primitive for HARNESS-045 (#561).
//
// THIS COMMAND IS THE AGNOSTIC SEAM. Every harness's hook is a thin wrapper
// around one line that calls it, so the persona logic exists once, in Go,
// testable with no harness installed. The alternative — a per-harness adapter
// carrying its own logic — is a second implementation that will drift, which is
// the failure this repository has catalogued repeatedly.
//
// IT EXITS 0 OR 2 AND NOTHING ELSE. The harnesses disagree about what a hook
// *error* means: claude's hooks must never crash a session (fail-open), while pi
// documents that `tool_call` errors block the tool (fail-safe, fail-closed). A
// command that could exit 1 would therefore mean opposite things on two
// harnesses. Every path here resolves to an explicit decision instead, so the
// harness never sees an error and its error semantics never apply.
func newHarnessGateCmd() *cobra.Command {
	var (
		harnessName string
		role        string
		repoRoot    string
		stateDir    string
	)

	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Decide whether a tool call may proceed under a persona's forced skills",
		Long: `gate reads a harness's tool-call event on stdin and answers with an exit code:
0 to allow, 2 to block. It is what a persona's "enforced by hook" claim compiles
to, and it is the same binary on every harness.

A skill declared 'enforce: block' that has not been consumed this session blocks
the call. One declared 'enforce: warn' is reported on stderr and allowed. A skill
carrying no declared severity is neither — it is a migration gap, surfaced by
` + "`dotf doctor`" + `, not noise on every tool call.

Invoking a skill is never blocked: forbidding it would deadlock the session.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, _ := io.ReadAll(cmd.InOrStdin())

			call, understood := normaliseToolCall(harnessName, payload)
			if !understood {
				// MEASURED 2026-08-26: this branch is why it exists. Without
				// it a malformed payload produced a zero ToolCall, Decide saw
				// a valid persona with nothing consumed, and the gate BLOCKED
				// — the opposite of the documented contract, and invisible to
				// unit tests that call Decide directly. A gate that blocks on
				// input it cannot read blocks on every harness upgrade.
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "[gate] allow: payload not recognised")
				return nil
			}
			if stateDir == "" {
				stateDir = defaultGateStateDir()
			}
			statePath := harness.StatePath(stateDir, call.SessionID)

			// A skill invocation is the act the gate exists to require: record
			// it and get out of the way. Recording failures are ignored on
			// purpose — losing the record costs a redundant skill run, while
			// failing here would block a session over a full disk.
			if call.Skill != "" {
				_ = harness.RecordConsumed(statePath, call.Skill)
				return nil
			}

			persona := loadGatePersona(repoRoot, role)
			result := harness.Decide(harness.GateInput{
				Persona:  persona,
				Call:     call,
				Consumed: harness.LoadConsumed(statePath),
			})

			for _, w := range result.Warned {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[gate] warn: %s not consumed\n", w)
			}
			if result.Decision == harness.Block {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[gate] blocked: %s\n", result.Reason)
				os.Exit(gateExitBlock)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "harness emitting the event (claude|pi|opencode)")
	cmd.Flags().StringVar(&role, "role", "", "persona in scope")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "root containing harness/agents (default: the deploy dir, else the checkout)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "where per-session consumption is recorded")
	return cmd
}

// loadGatePersona resolves the persona, returning nil rather than an error.
//
// Nil means Allow. A misconfigured or missing record must not block every tool
// call in a session — that is the "normal state is red" failure, and it is worse
// than the gate being absent, because an operator disables a noisy gate and then
// has nothing.
func loadGatePersona(repoRoot, role string) *harness.Persona {
	if strings.TrimSpace(role) == "" {
		return nil
	}
	if repoRoot == "" {
		repoRoot = os.Getenv("DOTFILES_DIR")
		if repoRoot == "" {
			repoRoot = filepath.Join(os.Getenv("HOME"), ".dotfiles")
		}
	}
	p, err := harness.LoadPersona(filepath.Join(repoRoot, "harness", "agents", role, "AGENT.md"))
	if err != nil {
		return nil
	}
	return p
}

func defaultGateStateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "dotfiles")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "dotfiles")
}

// claudeHookPayload is the slice of claude's PreToolUse JSON the gate consumes.
type claudeHookPayload struct {
	SessionID string         `json:"session_id"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// piHookPayload is pi's `tool_call` shape.
type piHookPayload struct {
	SessionID string         `json:"sessionId"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

// opencodeHookPayload is opencode's `tool.execute.before` shape.
type opencodeHookPayload struct {
	SessionID string         `json:"sessionID"`
	Tool      string         `json:"tool"`
	Args      map[string]any `json:"args"`
}

// normaliseToolCall turns a harness's payload into the neutral shape.
//
// Normalisation is the ONLY per-harness code in the gate, and it is deliberately
// dumb: field renaming, nothing else. Every judgement lives in harness.Decide.
//
// A payload that cannot be parsed yields a zero ToolCall, which Decide resolves
// to Allow. Refusing to guess is the point — a gate that blocks on input it does
// not understand blocks on every harness upgrade.
func normaliseToolCall(harnessName string, payload []byte) (harness.ToolCall, bool) {
	if len(payload) == 0 {
		return harness.ToolCall{}, false
	}
	switch harnessName {
	case "pi":
		var p piHookPayload
		if json.Unmarshal(payload, &p) != nil {
			return harness.ToolCall{}, false
		}
		return harness.ToolCall{SessionID: p.SessionID, Tool: p.Tool, Skill: skillArg(p.Tool, p.Arguments)}, p.Tool != ""
	case "opencode":
		var p opencodeHookPayload
		if json.Unmarshal(payload, &p) != nil {
			return harness.ToolCall{}, false
		}
		return harness.ToolCall{SessionID: p.SessionID, Tool: p.Tool, Skill: skillArg(p.Tool, p.Args)}, p.Tool != ""
	default:
		var p claudeHookPayload
		if json.Unmarshal(payload, &p) != nil {
			return harness.ToolCall{}, false
		}
		return harness.ToolCall{SessionID: p.SessionID, Tool: p.ToolName, Skill: skillArg(p.ToolName, p.ToolInput)}, p.ToolName != ""
	}
}

// skillArg extracts the skill name when the tool IS the harness's skill
// primitive. Matched case-insensitively on the tool name because the three
// harnesses spell it differently and none of them promises stability.
func skillArg(tool string, args map[string]any) string {
	if !strings.EqualFold(tool, "skill") {
		return ""
	}
	for _, k := range []string{"skill", "name", "command"} {
		if v, ok := args[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
