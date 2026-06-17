#!/bin/bash

# claude-session-start.sh: Claude Code SessionStart hook
# Runs automatically at the start of every Claude Code session.
# Detects if CWD is inside an Obsidian vault and provides vault
# health context to Claude.
#
# Hook input: JSON on stdin with { cwd, session_id, ... }
# Hook output: JSON on stdout with additionalContext
#
# Deployed via dotfiles to ~/.dotfiles/scripts/
# Registered in ~/.claude/settings.json under hooks.SessionStart

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

# --- SDD-004: session-start-config.json reader ---
# Single source of truth for thresholds + injector flags. Lives at repo root
# (one level above scripts/). Override via SESSION_START_CONFIG env var.
# If the file or jq is missing, the script falls back to historical hardcoded
# defaults so SessionStart never breaks on a partial dotfiles install.
CONFIG_PATH="${SESSION_START_CONFIG:-$SCRIPT_DIR/../session-start-config.json}"
if [ -f "$CONFIG_PATH" ] && command -v jq >/dev/null 2>&1; then
    CFG_AVAILABLE=1
else
    CFG_AVAILABLE=0
fi

# Read a numeric threshold from session-start-config.json; echoes the default
# if config absent or key missing. Usage: VAR=$(cfg_threshold key default)
cfg_threshold() {
    local key="$1" default="$2"
    if [ "$CFG_AVAILABLE" = "1" ]; then
        jq -r ".thresholds.$key // $default" "$CONFIG_PATH" 2>/dev/null
    else
        echo "$default"
    fi
}

# Check if an injector is enabled in session-start-config.json. Defaults to
# true if the config or key is absent (preserves historical "all on" behavior).
# Helper is provided for future incremental gating; current PR does NOT wrap
# any injector — adding gates is a low-risk per-injector follow-up.
# Usage: cfg_injector_enabled key && { ... }
cfg_injector_enabled() {
    local key="$1" val
    if [ "$CFG_AVAILABLE" = "1" ]; then
        val=$(jq -r ".injectors.$key.enabled // true" "$CONFIG_PATH" 2>/dev/null)
        [ "$val" = "true" ]
    else
        return 0
    fi
}

# Read hook input from stdin
INPUT=$(cat)
CWD=$(echo "$INPUT" | jq -r '.cwd // empty' 2>/dev/null || echo "")

if [ -z "$CWD" ]; then
    CWD="$(pwd)"
fi

# SDD discipline reminder -- unconditional, first in additionalContext (SDD-001).
# Surfaces the Discipline Gate to every Claude Code session regardless of CWD,
# repo, or vault state. Cross-OS parity with claude-session-start.ps1. All
# subsequent diagnostic blocks (claude-mem, doctor, hive, specs, vault, memory)
# APPEND to CONTEXT_LINES defensively so they cannot wipe this reminder.
CONTEXT_LINES='[sdd] Before your first tool call, read `AGENTS.md` at the repo root (or `~/Projects/dotfiles/AGENTS.md` as fallback) and apply its "Spec-Driven Development" (including the Discipline Gate) and "Standing Orders" sections. SDD applies by default for PR-sized changes (~50-300 LOC, public contract, new dep, multi-PR sequence). Skip ONLY for: typos, comment-only edits, mechanical refactors, bug fixes <20 lines with obvious cause, documentation-only changes. When in doubt, ASK the user.'

# --- session-brief core (ADR-023, HARNESS-026) ---
# The agent-independent vault signals (vault detection, vault-health, spec
# counts, baseline integrity) live in the agnostic session-brief core. This hook
# is the Claude adapter: it sources the core as a library and calls the sb_*
# emitters at their legacy positions below, so the emitted additionalContext
# stays byte-identical. No-op fallbacks first so a partial install (core script
# not yet deployed) still runs and degrades to "no vault signals".
# NOTE: sourced AFTER the [sdd] reminder above — the fallback find_vault_root()
# stub must stay below the [sdd] line (tests/hooks.bats anchors the SDD-first
# invariant on the first find_vault_root definition).
find_vault_root() { return 1; }
sb_vault_detect() { :; }
sb_vault_health() { :; }
sb_specs() { :; }
sb_vault_baseline() { :; }
SESSION_BRIEF_CORE="$SCRIPT_DIR/session-brief.sh"
if [ -f "$SESSION_BRIEF_CORE" ]; then
    # shellcheck source=session-brief.sh
    SESSION_BRIEF_LIB=1 . "$SESSION_BRIEF_CORE"
fi

# --- Self-heal claude-mem plugin if marketplace shipped broken artifacts ---
# Patches .mcp.json (${_R%/} regression, upstream #2385) and installs the
# missing zod runtime dep. Silent on healthy installs.
CLAUDE_MEM_HEAL="$SCRIPT_DIR/claude-mem-heal.sh"
if [ -x "$CLAUDE_MEM_HEAL" ]; then
    HEAL_OUTPUT=$(bash "$CLAUDE_MEM_HEAL" 2>&1) || true
    if [ -n "$HEAL_OUTPUT" ]; then
        CONTEXT_LINES="$CONTEXT_LINES

[claude-mem] self-healed plugin install:
$HEAL_OUTPUT"
    fi
fi

# --- Silent doctor: surface env-contract drift to Claude only when detected ---
# Uses `dotf doctor --quick` (CLI-013): the env-contract sweep only, no
# compile-harness gate, so it stays sub-100ms — unlike the full `dotf doctor`
# sweep (~2.8s) which is too heavy to fork per session. Forwards only [WARN]/
# [FAIL] lines. Gated on dotf being on PATH AND a deployed env-contract.json so
# the hermetic test (isolated HOME, no deployed contract) skips it.
DOTF_CONTRACT="${DOTFILES_DIR:-$HOME/.dotfiles}/env-contract.json"
if command -v dotf >/dev/null 2>&1 && [ -f "$DOTF_CONTRACT" ]; then
    DOCTOR_DRIFT=$(dotf doctor --quick 2>&1 | grep -E '^  \[(WARN|FAIL)\]' || true)
    if [ -n "$DOCTOR_DRIFT" ]; then
        if [ -n "$CONTEXT_LINES" ]; then
            CONTEXT_LINES="$CONTEXT_LINES

[doctor] env-contract drift detected (run 'dotf doctor --fix'):
$DOCTOR_DRIFT"
        else
            CONTEXT_LINES="[doctor] env-contract drift detected (run 'dotf doctor --fix'):
$DOCTOR_DRIFT"
        fi
    fi
fi

# find_vault_root is provided by the sourced session-brief core.
VAULT_ROOT=$(find_vault_root "$CWD") || true
KNOWLEDGE_VAULT="$HOME/Projects/knowledge"
VAULT_NAME=""

# --- Hive: detect project and suggest vault queries ---
# If CWD is a git repo, check if there's a matching vault project entry.
# Checks: 1) personal projects (10_projects/), 2) work SDK projects (50_work/45-development/).
detect_hive_project() {
    local repo_name vault_project_dir
    repo_name=$(basename "$CWD")
    vault_project_dir="$KNOWLEDGE_VAULT/10_projects/$repo_name"

    if [ -d "$vault_project_dir" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
[hive] Project '$repo_name' found in vault. Use hive-vault MCP tools for on-demand context:
  - vault_query(project=\"$repo_name\", section=\"context\") — project overview
  - vault_query(project=\"$repo_name\", section=\"tasks\") — active backlog
  - vault_search(query=\"...\") — search across vault
  - vault_query(project=\"_meta\", path=\"patterns/...\") — cross-project patterns"
        return 0
    fi

    # Fallback: check work SDK projects under 50_work/45-development/
    find_work_sdk_project
}

# Detect work SDK nested repos by matching path segments against vault family/component dirs.
# Heuristic: slugify and compare CWD path segments against 45-development/<family>/<component>.
find_work_sdk_project() {
    local dev_dir="$KNOWLEDGE_VAULT/50_work/45-development"
    [ -d "$dev_dir" ] || return 0

    local family_dir family_name family_slug cwd_path_slug comp_dir comp_name comp_slug
    # Slugify CWD: lowercase, remove non-alphanumeric except /
    cwd_path_slug=$(printf '%s' "$CWD" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9/')

    for family_dir in "$dev_dir"/*/; do
        [ -d "$family_dir" ] || continue
        family_name=$(basename "$family_dir")
        family_slug=$(printf '%s' "$family_name" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')

        if printf '%s' "$cwd_path_slug" | grep -q "$family_slug"; then
            # Family matches — look for component
            for comp_dir in "$family_dir"*/; do
                [ -d "$comp_dir" ] || continue
                comp_name=$(basename "$comp_dir")
                comp_slug=$(printf '%s' "$comp_name" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')

                if printf '%s' "$cwd_path_slug" | grep -q "$comp_slug"; then
                    CONTEXT_LINES="$CONTEXT_LINES
[hive] Work SDK '$comp_name' (family: $family_name) found in vault. Use Hive MCP:
  - vault_query(project=\"_meta\", path=\"50_work/45-development/$family_name/$comp_name/00-context.md\") — project context
  - vault_query(project=\"_meta\", path=\"50_work/45-development/$family_name/$comp_name/memory/MEMORY.md\") — session memory
  - vault_query(project=\"_meta\", path=\"patterns/workflow-protocol\") — session protocol"
                    return 0
                fi
            done
            # Family matched but no component — suggest family context
            CONTEXT_LINES="$CONTEXT_LINES
[hive] Work SDK family '$family_name' found in vault. Use Hive MCP:
  - vault_query(project=\"_meta\", path=\"50_work/45-development/$family_name/00-context.md\") — all repos in family"
            return 0
        fi
    done
}

# Spec counts (sb_specs) come from the session-brief core; detect_hive_project
# stays in the adapter (deferred to a later slice).
if [ -d "$CWD/.git" ]; then
    detect_hive_project
    CONTEXT_LINES="$CONTEXT_LINES$(sb_specs "$CWD")"
fi

# Note: previous versions exited silently here when both VAULT_ROOT and
# CONTEXT_LINES were empty. After SDD-001 the [sdd] reminder is always present,
# so CONTEXT_LINES is never empty -- the early-exit branch is now dead code and
# was removed. The hook always emits the additionalContext envelope.

if [ -n "$VAULT_ROOT" ]; then
    VAULT_NAME=$(basename "$VAULT_ROOT")
    CONTEXT_LINES="$(sb_vault_detect "$VAULT_ROOT")

$CONTEXT_LINES"
fi

# Vault-health block (agent-independent) comes from the session-brief core.
CONTEXT_LINES="$CONTEXT_LINES$(sb_vault_health "$VAULT_ROOT" "$VAULT_NAME" "$SCRIPT_DIR")"

# --- Knowledge maintenance health check ---
# Derives MEMORY.md path from CWD using Claude Code's path encoding convention.
# Warns if MEMORY.md is stale or too large.

encode_project_path() {
    printf '%s' "$1" | tr '/' '-'
}

# --- Auto-create memory symlink if vault has memory/ for this project ---
# Runs before health check so the symlink exists when health check reads it.
# The vault-source resolution lives in the agent-agnostic ensure-memory-symlink.sh
# (MEMORY-002); this hook only computes Claude's encoded target and delegates.
ensure_memory_symlink() {
    local encoded target_dir msg helper
    helper="$SCRIPT_DIR/ensure-memory-symlink.sh"
    [ -x "$helper" ] || return 0   # degrade cleanly if the sibling isn't deployed
    encoded=$(encode_project_path "$CWD")
    target_dir="$HOME/.claude/projects/$encoded/memory"

    msg=$(VAULT_PATH="$KNOWLEDGE_VAULT" "$helper" \
        --cwd "$CWD" --target "$target_dir" 2>/dev/null) || true
    if [ -n "$msg" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
$msg"
    fi
    return 0   # never let an empty-msg short-circuit fail the hook under set -e
}

ensure_memory_symlink

check_knowledge_health() {
    local encoded memory_file today line_count last_date days_since today_epoch last_epoch
    encoded=$(encode_project_path "$CWD")
    memory_file="$HOME/.claude/projects/$encoded/memory/MEMORY.md"

    [ -f "$memory_file" ] || return 0

    today=$(date +%Y-%m-%d)
    line_count=$(wc -l < "$memory_file")
    # Marker may be indented (e.g. when MEMORY.md is embedded in a YAML `content: |`
    # block, as in the knowledge vault) — anchor allows leading whitespace (BUG-022).
    last_date=$(grep -E '^[[:space:]]*## Last Crystallized:' "$memory_file" | tail -1 | sed -E 's/^[[:space:]]*## Last Crystallized: //' || true)

    local mem_max_lines
    mem_max_lines=$(cfg_threshold memory_md_max_lines 150)
    if [ "$line_count" -gt "$mem_max_lines" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
MEMORY.md has $line_count lines (limit: $mem_max_lines) — run /crystallize to trim"
    fi

    if [ -z "$last_date" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
Knowledge crystallization never run — run: ./scripts/knowledge-crystallize.sh"
    else
        # POSIX-compatible date diff (GNU date on Linux, BSD date on macOS)
        today_epoch=$(date -d "$today" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$today" +%s 2>/dev/null || echo 0)
        last_epoch=$(date -d "$last_date" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$last_date" +%s 2>/dev/null || echo 0)
        if [ "$today_epoch" -gt 0 ] && [ "$last_epoch" -gt 0 ]; then
            days_since=$(( (today_epoch - last_epoch) / 86400 ))
            local stale_max
            stale_max=$(cfg_threshold crystallize_max_days 14)
            if [ "$days_since" -gt "$stale_max" ]; then
                CONTEXT_LINES="$CONTEXT_LINES
CRYSTALLIZE NEEDED (${days_since} days stale)"
            fi
        fi
    fi
}

check_knowledge_health

# Vault baseline integrity (SDD-017b) comes from the session-brief core.
CONTEXT_LINES="$CONTEXT_LINES$(sb_vault_baseline "$VAULT_ROOT")"

# --- Memory temperature scan ---
# Reads file modification times to classify memory files as HOT/WARM/COLD.
# Read-only — never modifies files.

check_memory_temperature() {
    local encoded memory_dir now file_epoch days_ago fname label
    local temp_report="" has_cold_archive=false

    encoded=$(encode_project_path "$CWD")
    memory_dir="$HOME/.claude/projects/$encoded/memory"

    [ -d "$memory_dir" ] || return 0

    now=$(date +%s)
    local hot_days warm_days cold_days
    hot_days=$(cfg_threshold memory_temp_hot_days 7)
    warm_days=$(cfg_threshold memory_temp_warm_days 30)
    cold_days=$(cfg_threshold memory_temp_cold_days 60)

    for f in "$memory_dir"/*.md; do
        [ -f "$f" ] || continue
        [ "$(basename "$f")" = "MEMORY.md" ] && continue

        # Get file modification time (Linux/macOS compatible)
        file_epoch=$(stat -c %Y "$f" 2>/dev/null || stat -f %m "$f" 2>/dev/null || echo 0)
        [ "$file_epoch" -eq 0 ] && continue

        days_ago=$(( (now - file_epoch) / 86400 ))
        fname=$(basename "$f")

        if [ "$days_ago" -le "$hot_days" ]; then
            label="HOT"
        elif [ "$days_ago" -le "$warm_days" ]; then
            label="WARM"
        elif [ "$days_ago" -le "$cold_days" ]; then
            label="COLD"
        else
            label="ARCHIVE"
            has_cold_archive=true
        fi

        temp_report="${temp_report}
  ${label}: ${fname} (${days_ago}d ago)"
    done

    if [ -n "$temp_report" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
Memory temperature:${temp_report}"
        if $has_cold_archive; then
            CONTEXT_LINES="$CONTEXT_LINES
ARCHIVE NEEDED: Move memory files >60d old to memory/archive/ and update MEMORY.md index"
        fi
    fi
}

check_memory_temperature

# --- .claude.json size monitor (SDD-021) ---
# Defensive layer for the `claude plugin install` truncation bug (dotfiles#33 trigger
# fix, anthropics/claude-code#59870 upstream). If `~/.claude/.claude.json` drops below
# 10 KB, flag it — healthy size is ~75 KB, post-truncation is ~1.5 KB. Catches
# recurrence even if a different `claude` CLI subcommand introduces the same strip
# behavior (e.g. mcp add/remove, plugin uninstall).
check_claude_json_size() {
    local claude_json="$HOME/.claude/.claude.json"
    local threshold size
    threshold=$(cfg_threshold claude_json_min_bytes 10240)

    [ -f "$claude_json" ] || return 0

    # stat -c works on Linux, stat -f on macOS/BSD
    size=$(stat -c %s "$claude_json" 2>/dev/null || stat -f %z "$claude_json" 2>/dev/null || echo 0)

    if [ "$size" -gt 0 ] && [ "$size" -lt "$threshold" ]; then
        CONTEXT_LINES="$CONTEXT_LINES
[claude.json] WARNING: ~/.claude/.claude.json is ${size} bytes (threshold ${threshold}). Healthy state is ~75 KB; truncation bug (anthropics/claude-code#59870) reduces it to ~1.5 KB and silently drops subscription state. Recovery: ls -t ~/.claude/backups/.claude.json.backup.* | head -1 && cp <newest-backup> ~/.claude/.claude.json"
    fi
}

check_claude_json_size

# Return context to Claude via hook output format
jq -n --arg ctx "$CONTEXT_LINES" '{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": $ctx
  }
}'

exit 0
