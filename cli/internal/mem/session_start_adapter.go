package mem

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// This file is the Claude session-start adapter (CLI-025 PR2b-2b): it composes the
// agnostic session-brief sb_* emitters (PR2a) and the Claude-only injectors (PR2b-2a)
// into the hook's additionalContext, in the exact CONTEXT_LINES order of
// claude-session-start.sh, then wraps it in the SessionStart JSON envelope. The
// output is contractually byte-equivalent to the shell hook (golden-fixture gated).

// sddReminder is the unconditional Discipline Gate reminder (SDD-001), always first
// in additionalContext regardless of CWD/vault state. Byte-identical to the shell.
const sddReminder = "[sdd] Before your first tool call, read `AGENTS.md` at the repo root (or `~/Projects/dotfiles/AGENTS.md` as fallback) and apply its \"Spec-Driven Development\" (including the Discipline Gate) and \"Standing Orders\" sections. SDD applies by default for PR-sized changes (~50-300 LOC, public contract, new dep, multi-PR sequence). Skip ONLY for: typos, comment-only edits, mechanical refactors, bug fixes <20 lines with obvious cause, documentation-only changes. When in doubt, ASK the user."

// lessonsStaleDays is the sb_lessons_staleness threshold (session-brief.sh default).
const lessonsStaleDays = 14

// ClaudeJSONMinBytes is the size below which ~/.claude/.claude.json counts as
// truncated by the upstream strip bug (anthropics/claude-code#59870): the
// session-start canary warns under it, and `dotf deploy`'s snapshot guard
// (CLI-063) restores only a snapshot at least this large. One constant, so the
// two cannot disagree about what a healthy file is.
const ClaudeJSONMinBytes = 10240

// ClaudeContextInput injects every path/clock the adapter touches, so the assembly
// is hermetically testable; the command wiring resolves them from the env-contract.
type ClaudeContextInput struct {
	Cwd          string          // hook stdin .cwd
	Vault        string          // KNOWLEDGE_VAULT (VAULT_PATH)
	VaultHealth  VaultHealthFunc // nil skips the vault health section
	Home         string          // $HOME — roots ~/.claude/projects/<encoded>/memory
	ContractPath string          // env-contract.json; doctor-drift is gated on its presence
	ClaudeJSON   string          // ~/.claude/.claude.json
	ConfigPath   string          // session-start-config.json (SDD-004 thresholds)
	Now          time.Time       // injected clock for staleness/temperature
	DoctorQuick  func() string   // returns `dotf doctor --quick` output; nil = skip
	// TriageQueue returns the pull requests awaiting a disposition, or an error
	// when the question could not be answered. nil skips the section.
	TriageQueue func() (string, error)
}

// ClaudeContext assembles the additionalContext string in claude-session-start.sh's
// exact order: [sdd] reminder, doctor-drift, (.git: hive + specs + lessons), prepend
// vault headline, vault-health, auto-memory link, knowledge-health, vault-baseline,
// memory-temperature, claude.json-size.
func ClaudeContext(in ClaudeContextInput) string {
	cfg := loadAdapterConfig(in.ConfigPath)
	ctx := sddReminder

	// doctor-drift is gated on a deployed env-contract (the hermetic test skips it).
	if in.DoctorQuick != nil && fileExists(in.ContractPath) {
		ctx += doctorDrift(in.DoctorQuick())
	}

	vaultRoot := findVaultRoot(in.Cwd)

	checkoutRoot, project := checkoutIdentity(in.Cwd)
	if checkoutRoot != "" {
		ctx += hiveProjectNamed(checkoutRoot, in.Vault, project)
		ctx += specs(checkoutRoot)
		ctx += lessonsStaleness(checkoutRoot, lessonsStaleDays, in.Now)
		if in.TriageQueue != nil {
			ctx += triageQueue(in.TriageQueue())
		}
	}

	vaultName := ""
	if vaultRoot != "" {
		vaultName = filepath.Base(vaultRoot)
		ctx = vaultDetect(vaultRoot) + "\n\n" + ctx // prepend the headline + blank line
	}

	ctx += vaultHealth(vaultRoot, vaultName, in.VaultHealth)
	ctx += memorySymlink(in.Cwd, in.Vault, in.Home)

	memoryDir := memlink.ClaudeMemoryTarget(in.Home, in.Cwd)
	ctx += knowledgeHealth(filepath.Join(memoryDir, "MEMORY.md"),
		cfg.threshold("memory_md_max_lines", 150), cfg.threshold("crystallize_max_days", 14), in.Now)
	ctx += vaultBaseline(vaultRoot)
	ctx += memoryTemperature(memoryDir,
		cfg.threshold("memory_temp_hot_days", 7), cfg.threshold("memory_temp_warm_days", 30),
		cfg.threshold("memory_temp_cold_days", 60), in.Now)
	ctx += claudeJSONSize(in.ClaudeJSON, cfg.threshold("claude_json_min_bytes", ClaudeJSONMinBytes))

	return ctx
}

// hookEnvelope is the SessionStart hook output shape. Struct field order pins the
// JSON key order (a map would randomize it).
type hookEnvelope struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// ClaudeEnvelope renders the additionalContext as the SessionStart hook JSON, matching
// `jq -n` byte-for-byte: 2-space indent, a trailing newline, and — critically — no
// HTML escaping of <, >, & (Go's json default escapes them; jq does not).
func ClaudeEnvelope(ctx string) (string, error) {
	var env hookEnvelope
	env.HookSpecificOutput.HookEventName = "SessionStart"
	env.HookSpecificOutput.AdditionalContext = ctx

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// fileExists reports whether path exists (file or dir).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
