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

# Read hook input from stdin
INPUT=$(cat)
CWD=$(echo "$INPUT" | jq -r '.cwd // empty' 2>/dev/null || echo "")

if [ -z "$CWD" ]; then
    CWD="$(pwd)"
fi

# Walk up from CWD to find an Obsidian vault (.obsidian/ directory)
find_vault_root() {
    local dir="$1"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.obsidian" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

VAULT_ROOT=$(find_vault_root "$CWD") || true
KNOWLEDGE_VAULT="$HOME/Projects/knowledge"
VAULT_NAME=""
CONTEXT_LINES=""

# --- Hive: detect project and suggest vault queries ---
# If CWD is a git repo, check if there's a matching vault project entry.
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
    fi
}

if [ -d "$CWD/.git" ]; then
    detect_hive_project
fi

if [ -z "$VAULT_ROOT" ] && [ -z "$CONTEXT_LINES" ]; then
    # Not inside a vault and no project detected — exit cleanly
    exit 0
fi

if [ -n "$VAULT_ROOT" ]; then
    VAULT_NAME=$(basename "$VAULT_ROOT")
    CONTEXT_LINES="Obsidian vault detected: $VAULT_NAME ($VAULT_ROOT)
$CONTEXT_LINES"
fi

# Try running vault-health.sh if available
VAULT_HEALTH="$SCRIPT_DIR/vault-health.sh"
if [ -x "$VAULT_HEALTH" ]; then
    # Run with vault env vars, capture output, tolerate failures
    HEALTH_OUTPUT=$(
        VAULT_DIR="$VAULT_ROOT" VAULT_NAME="$VAULT_NAME" \
        bash "$VAULT_HEALTH" 2>&1
    ) || HEALTH_EXIT=$?
    HEALTH_EXIT=${HEALTH_EXIT:-0}

    if [ "$HEALTH_EXIT" -eq 2 ]; then
        CONTEXT_LINES="$CONTEXT_LINES
Obsidian GUI not running — vault health skipped. Run 'vault-health.sh' manually when GUI is up."
    elif [ "$HEALTH_EXIT" -eq 0 ]; then
        CONTEXT_LINES="$CONTEXT_LINES
Vault health: ALL CHECKS PASSED"
    else
        # Extract summary line (Results: X passed, Y failed, Z skipped)
        SUMMARY=$(echo "$HEALTH_OUTPUT" | sed 's/\x1b\[[0-9;]*m//g' | grep '^Results:' || echo "")
        # Extract FAIL lines
        FAILURES=$(echo "$HEALTH_OUTPUT" | sed 's/\x1b\[[0-9;]*m//g' | grep 'FAIL' || echo "")
        CONTEXT_LINES="$CONTEXT_LINES
Vault health: $SUMMARY
Issues found:
$FAILURES"
    fi
else
    CONTEXT_LINES="$CONTEXT_LINES
vault-health.sh not found at $VAULT_HEALTH — run dotfiles setup to install."
fi

# --- Knowledge maintenance health check ---
# Derives MEMORY.md path from CWD using Claude Code's path encoding convention.
# Warns if MEMORY.md is stale or too large.

encode_project_path() {
    printf '%s' "$1" | tr '/' '-'
}

check_knowledge_health() {
    local encoded memory_file today line_count last_date days_since today_epoch last_epoch
    encoded=$(encode_project_path "$CWD")
    memory_file="$HOME/.claude/projects/$encoded/memory/MEMORY.md"

    [ -f "$memory_file" ] || return 0

    today=$(date +%Y-%m-%d)
    line_count=$(wc -l < "$memory_file")
    last_date=$(grep '^## Last Crystallized:' "$memory_file" | tail -1 | sed 's/## Last Crystallized: //' || true)

    if [ "$line_count" -gt 150 ]; then
        CONTEXT_LINES="$CONTEXT_LINES
MEMORY.md has $line_count lines (limit: 150) — run /crystallize to trim"
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
            if [ "$days_since" -gt 14 ]; then
                CONTEXT_LINES="$CONTEXT_LINES
CRYSTALLIZE NEEDED (${days_since} days stale)"
            fi
        fi
    fi
}

check_knowledge_health

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

    for f in "$memory_dir"/*.md; do
        [ -f "$f" ] || continue
        [ "$(basename "$f")" = "MEMORY.md" ] && continue

        # Get file modification time (Linux/macOS compatible)
        file_epoch=$(stat -c %Y "$f" 2>/dev/null || stat -f %m "$f" 2>/dev/null || echo 0)
        [ "$file_epoch" -eq 0 ] && continue

        days_ago=$(( (now - file_epoch) / 86400 ))
        fname=$(basename "$f")

        if [ "$days_ago" -le 7 ]; then
            label="HOT"
        elif [ "$days_ago" -le 30 ]; then
            label="WARM"
        elif [ "$days_ago" -le 60 ]; then
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

# Return context to Claude via hook output format
jq -n --arg ctx "$CONTEXT_LINES" '{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": $ctx
  }
}'

exit 0
