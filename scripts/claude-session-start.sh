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

# --- Self-heal claude-mem plugin if marketplace shipped broken artifacts ---
# Patches .mcp.json (${_R%/} regression, upstream #2385) and installs the
# missing zod runtime dep. Silent on healthy installs.
CLAUDE_MEM_HEAL="$SCRIPT_DIR/claude-mem-heal.sh"
CONTEXT_LINES=""
if [ -x "$CLAUDE_MEM_HEAL" ]; then
    HEAL_OUTPUT=$(bash "$CLAUDE_MEM_HEAL" 2>&1) || true
    if [ -n "$HEAL_OUTPUT" ]; then
        CONTEXT_LINES="[claude-mem] self-healed plugin install:
$HEAL_OUTPUT"
    fi
fi

# --- Silent doctor: surface env-contract drift to Claude only when detected ---
# Runs check-only; suppresses [ok]/[info] lines and only forwards [warn]/[fail].
DOCTOR_SCRIPT="$SCRIPT_DIR/doctor.sh"
if [ -x "$DOCTOR_SCRIPT" ] && command -v jq >/dev/null 2>&1; then
    DOCTOR_DRIFT=$(bash "$DOCTOR_SCRIPT" 2>&1 | grep -E '^  \[(warn|fail)\]' || true)
    if [ -n "$DOCTOR_DRIFT" ]; then
        if [ -n "$CONTEXT_LINES" ]; then
            CONTEXT_LINES="$CONTEXT_LINES

[doctor] env-contract drift detected (run scripts/doctor.sh --fix):
$DOCTOR_DRIFT"
        else
            CONTEXT_LINES="[doctor] env-contract drift detected (run scripts/doctor.sh --fix):
$DOCTOR_DRIFT"
        fi
    fi
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

# --- Spec-Driven Development: detect repo specs/ and surface in-flight work ---
# Counts active vs archived specs under $CWD/specs/ and flags any spec
# containing unresolved [AGENT-DRAFT] or [AGENT-SUGGESTION] tags.
# Discovery without proactive nag — silent if no specs/.
detect_repo_specs() {
    local specs_dir d name active_count archive_count drafts_specs draft_count msg
    specs_dir="$CWD/specs"
    [ -d "$specs_dir" ] || return 0

    active_count=0
    drafts_specs=""
    for d in "$specs_dir"/*/; do
        [ -d "$d" ] || continue
        name=$(basename "$d")
        [ "$name" = "archive" ] && continue
        active_count=$((active_count + 1))
        if grep -rlE '\[AGENT-DRAFT\]|\[AGENT-SUGGESTION\]' "$d" >/dev/null 2>&1; then
            drafts_specs="$drafts_specs $name"
        fi
    done

    archive_count=0
    if [ -d "$specs_dir/archive" ]; then
        for d in "$specs_dir/archive"/*/; do
            [ -d "$d" ] || continue
            archive_count=$((archive_count + 1))
        done
    fi

    # Silent if entirely empty
    if [ "$active_count" -eq 0 ] && [ "$archive_count" -eq 0 ]; then
        return 0
    fi

    msg="[specs] $active_count active, $archive_count archived"
    if [ -n "$drafts_specs" ]; then
        drafts_specs="${drafts_specs# }"
        draft_count=$(printf '%s' "$drafts_specs" | wc -w | tr -d ' ')
        msg="$msg — $draft_count with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags:"
        for name in $drafts_specs; do
            msg="$msg
  - $name"
        done
    fi

    CONTEXT_LINES="$CONTEXT_LINES
$msg"
}

if [ -d "$CWD/.git" ]; then
    detect_hive_project
    detect_repo_specs
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
        # GUI down: GUI-dependent checks skipped, but integrity check (git-based) ran anyway.
        # Scan HEALTH_OUTPUT for any FAIL lines so integrity alerts still surface.
        INTEGRITY_FAILS=$(echo "$HEALTH_OUTPUT" | sed 's/\x1b\[[0-9;]*m//g' | grep 'FAIL' || echo "")
        if [ -n "$INTEGRITY_FAILS" ]; then
            CONTEXT_LINES="$CONTEXT_LINES
Obsidian GUI not running — GUI-dependent checks skipped. Integrity issues found:
$INTEGRITY_FAILS"
        else
            CONTEXT_LINES="$CONTEXT_LINES
Obsidian GUI not running — vault health skipped. Run 'vault-health.sh' manually when GUI is up."
        fi
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

# --- Auto-create memory symlink if vault has memory/ for this project ---
# Runs before health check so the symlink exists when health check reads it.
ensure_memory_symlink() {
    local encoded target_dir project_name vault_memory parent_dir
    encoded=$(encode_project_path "$CWD")
    target_dir="$HOME/.claude/projects/$encoded/memory"

    # Already linked? Skip.
    if [ -L "$target_dir" ]; then return 0; fi
    if [ -d "$target_dir" ] && [ "$(ls -A "$target_dir" 2>/dev/null)" ]; then return 0; fi

    # Try 10_projects/<name>/memory/ (personal projects convention)
    project_name=$(basename "$CWD")
    vault_memory="$KNOWLEDGE_VAULT/10_projects/$project_name/memory"

    if [ ! -d "$vault_memory" ]; then
        # Try CWD/memory/ (knowledge sessions where CWD is inside the vault)
        case "$CWD" in
            "$KNOWLEDGE_VAULT"*) vault_memory="$CWD/memory" ;;
            *) vault_memory="" ;;
        esac
    fi

    # Try 50_work/45-development/<family>/<component>/memory/ for nested work SDK repos
    if [ -z "$vault_memory" ] || [ ! -d "$vault_memory" ]; then
        local dev_dir="$KNOWLEDGE_VAULT/50_work/45-development"
        if [ -d "$dev_dir" ]; then
            local cwd_path_slug sdk_family_dir sdk_family_slug sdk_comp_dir sdk_comp_slug
            cwd_path_slug=$(printf '%s' "$CWD" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9/')
            for sdk_family_dir in "$dev_dir"/*/; do
                [ -d "$sdk_family_dir" ] || continue
                sdk_family_slug=$(basename "$sdk_family_dir" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')
                if printf '%s' "$cwd_path_slug" | grep -q "$sdk_family_slug"; then
                    for sdk_comp_dir in "$sdk_family_dir"*/; do
                        [ -d "$sdk_comp_dir" ] || continue
                        sdk_comp_slug=$(basename "$sdk_comp_dir" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')
                        if printf '%s' "$cwd_path_slug" | grep -q "$sdk_comp_slug"; then
                            vault_memory="$sdk_comp_dir/memory"
                            break 2
                        fi
                    done
                fi
            done
        fi
    fi

    [ -d "$vault_memory" ] || return 0

    # Create parent dir and symlink
    parent_dir="$HOME/.claude/projects/$encoded"
    mkdir -p "$parent_dir"

    # Remove empty target dir if it exists (no files, not a symlink)
    if [ -d "$target_dir" ] && [ -z "$(ls -A "$target_dir" 2>/dev/null)" ]; then
        rmdir "$target_dir" 2>/dev/null || true
    fi

    if ln -s "$vault_memory" "$target_dir" 2>/dev/null; then
        CONTEXT_LINES="$CONTEXT_LINES
[auto-memory] Created symlink for $project_name"
    fi
}

ensure_memory_symlink

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
