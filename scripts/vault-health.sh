#!/bin/bash

# vault-health.sh: Knowledge vault health report using Obsidian CLI
# Usage: ./scripts/vault-health.sh [-v|--verbose] [--vault NAME]
# Exit: 0 = all pass, 1 = check failures, 2 = GUI not running
#
# Requires Obsidian GUI running (CLI communicates via IPC).
# All checks degrade gracefully when GUI is down.

set -euo pipefail

# Load utility functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    # shellcheck source=utils.sh disable=SC1091
    . "$SCRIPT_DIR/utils.sh"
else
    echo "Error: utils.sh not found in $SCRIPT_DIR"
    exit 1
fi

# Defaults
VAULT_NAME="${VAULT_NAME:-knowledge}"
# VAULT_DIR is the internal handoff from the session-start caller (already resolved).
# When run standalone, fall back to the ADR-025 contract seam VAULT_PATH, then the
# contract's POSIX default. (VAULT_DIR wins so the caller's handoff is intact.)
VAULT_DIR="${VAULT_DIR:-${VAULT_PATH:-$HOME/Projects/knowledge}}"
VERBOSE=false
OBSIDIAN_CMD="obsidian --no-sandbox"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose) VERBOSE=true; shift ;;
        --vault) VAULT_NAME="$2"; shift 2 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# Test counters
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_SKIPPED=0

pass() {
    printf '  %bPASS%b: %s\n' "$GREEN" "$NC" "$1"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

fail() {
    printf '  %bFAIL%b: %s\n' "$RED" "$NC" "$1"
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
}

warn() {
    printf '  %bWARN%b: %s\n' "$YELLOW" "$NC" "$1"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

skip() {
    printf '  %bSKIP%b: %s - %s\n' "$YELLOW" "$NC" "$1" "$2"
    CHECKS_SKIPPED=$((CHECKS_SKIPPED + 1))
}

info() {
    printf '  %bINFO%b: %s\n' "$CYAN" "$NC" "$1"
}

section() {
    echo ""
    printf '%b[%s]%b %s\n' "$BLUE" "$1" "$NC" "$2"
}

# Run an obsidian CLI command, capturing output
# Returns 1 if CLI fails (GUI not running)
obsidian_cmd() {
    $OBSIDIAN_CMD "$@" --vault "$VAULT_NAME" 2>/dev/null
}

# ============================================================
echo "========================================"
echo "   VAULT HEALTH REPORT"
echo "========================================"
echo "Vault: $VAULT_NAME ($VAULT_DIR)"

# ==================================================
section "1/7" "Working Tree Integrity"

# GUI-independent check: detect files deleted from working tree but still in HEAD.
# Triggered by 2026-05-13 incident — SKILL.md vanished from disk, git status showed `D`.
# Pattern `^.D ` matches Y-column=D (unstaged delete), NOT X-column=D (intentional `git rm`).
if [ ! -d "$VAULT_DIR" ]; then
    skip "Working tree integrity" "vault dir not found"
elif [ ! -d "$VAULT_DIR/.git" ]; then
    skip "Working tree integrity" "vault is not a git repo"
else
    DELETED_LINES=$(git -C "$VAULT_DIR" status --short 2>/dev/null | grep '^.D ' || true)
    if [ -z "$DELETED_LINES" ]; then
        pass "Working tree clean — no files deleted from disk"
    else
        DELETED_COUNT=$(printf '%s\n' "$DELETED_LINES" | wc -l | tr -d ' ')
        fail "$DELETED_COUNT file(s) deleted from working tree but still in HEAD — likely sync/watcher race"
        printf '%s\n' "$DELETED_LINES" | sed 's|^|        |'
        printf '        Recovery: cd %s && git checkout -- <file>\n' "$VAULT_DIR"
    fi
fi

# ==================================================
section "2/7" "Vault Connectivity"

if ! command_exists obsidian; then
    log_error "Obsidian CLI not found in PATH"
    exit 1
fi

# Test CLI connectivity by running a simple command
obsidian_cmd vault 2>/dev/null | grep -q . || fail_obsidian_gui
pass "Obsidian CLI connected to vault '$VAULT_NAME'"

if [ -d "$VAULT_DIR" ]; then
    TOTAL_FILES=$(find "$VAULT_DIR" -name '*.md' -not -path '*/.obsidian/*' | wc -l)
    info "Total markdown files: $TOTAL_FILES"
else
    fail "Vault directory not found: $VAULT_DIR"
    TOTAL_FILES=0
fi

# ==================================================
section "3/7" "Orphans & Dead-Ends"

ORPHAN_OUTPUT=$(obsidian_cmd orphans 2>/dev/null || true)
ORPHAN_COUNT=$(echo "$ORPHAN_OUTPUT" | grep -c '[^[:space:]]') || ORPHAN_COUNT=0

DEADEND_OUTPUT=$(obsidian_cmd dead-ends 2>/dev/null || true)
DEADEND_COUNT=$(echo "$DEADEND_OUTPUT" | grep -c '[^[:space:]]') || DEADEND_COUNT=0

if [ "$TOTAL_FILES" -gt 0 ]; then
    ORPHAN_PCT=$((ORPHAN_COUNT * 100 / TOTAL_FILES))
    DEADEND_PCT=$((DEADEND_COUNT * 100 / TOTAL_FILES))
else
    ORPHAN_PCT=0
    DEADEND_PCT=0
fi

# Orphans threshold: <30% pass, 30-50% warn, >50% fail
if [ "$ORPHAN_PCT" -le 30 ]; then
    pass "Orphans: $ORPHAN_COUNT/$TOTAL_FILES ($ORPHAN_PCT%)"
elif [ "$ORPHAN_PCT" -le 50 ]; then
    warn "Orphans: $ORPHAN_COUNT/$TOTAL_FILES ($ORPHAN_PCT%) — consider adding backlinks"
else
    fail "Orphans: $ORPHAN_COUNT/$TOTAL_FILES ($ORPHAN_PCT%) — too many isolated files"
fi

# Dead-ends threshold: same as orphans
if [ "$DEADEND_PCT" -le 30 ]; then
    pass "Dead-ends: $DEADEND_COUNT/$TOTAL_FILES ($DEADEND_PCT%)"
elif [ "$DEADEND_PCT" -le 50 ]; then
    warn "Dead-ends: $DEADEND_COUNT/$TOTAL_FILES ($DEADEND_PCT%) — consider adding outgoing links"
else
    fail "Dead-ends: $DEADEND_COUNT/$TOTAL_FILES ($DEADEND_PCT%) — too many files without outgoing links"
fi

if [ "$VERBOSE" = true ]; then
    if [ "$ORPHAN_COUNT" -gt 0 ]; then
        echo "  --- Orphan files ---"
        echo "$ORPHAN_OUTPUT" | head -20
        [ "$ORPHAN_COUNT" -gt 20 ] && echo "  ... and $((ORPHAN_COUNT - 20)) more"
    fi
fi

# ==================================================
section "4/7" "Unresolved Links"

UNRESOLVED_OUTPUT=$(obsidian_cmd unresolved 2>/dev/null || true)
UNRESOLVED_COUNT=$(echo "$UNRESOLVED_OUTPUT" | grep -c '[^[:space:]]') || UNRESOLVED_COUNT=0

if [ "$UNRESOLVED_COUNT" -eq 0 ]; then
    pass "No unresolved links"
elif [ "$UNRESOLVED_COUNT" -le 10 ]; then
    warn "Unresolved links: $UNRESOLVED_COUNT"
else
    fail "Unresolved links: $UNRESOLVED_COUNT"
fi

if [ "$VERBOSE" = true ] && [ "$UNRESOLVED_COUNT" -gt 0 ]; then
    echo "  --- Unresolved links ---"
    echo "$UNRESOLVED_OUTPUT" | head -20
    [ "$UNRESOLVED_COUNT" -gt 20 ] && echo "  ... and $((UNRESOLVED_COUNT - 20)) more"
fi

# ==================================================
section "5/7" "Frontmatter Coverage"

check_frontmatter_field() {
    local field="$1"
    if [ "$TOTAL_FILES" -eq 0 ]; then
        skip "$field" "no markdown files"
        return
    fi
    local count
    count=$(grep -rl "^${field}:" "$VAULT_DIR" --include='*.md' 2>/dev/null | grep -vc '.obsidian') || count=0
    local pct=$((count * 100 / TOTAL_FILES))

    if [ "$pct" -ge 80 ]; then
        pass "$field: $count/$TOTAL_FILES ($pct%)"
    elif [ "$pct" -ge 50 ]; then
        warn "$field: $count/$TOTAL_FILES ($pct%)"
    else
        fail "$field: $count/$TOTAL_FILES ($pct%)"
    fi
}

if [ "$TOTAL_FILES" -gt 0 ]; then
    for field in id type status tags created owner; do
        check_frontmatter_field "$field"
    done
else
    skip "Frontmatter" "no markdown files found"
fi

# ==================================================
section "6/7" "Tag Hygiene"

TAG_OUTPUT=$(obsidian_cmd tags 2>/dev/null || true)
TAG_COUNT=$(echo "$TAG_OUTPUT" | grep -c '[^[:space:]]') || TAG_COUNT=0
info "Total unique tags: $TAG_COUNT"

if [ "$VERBOSE" = true ] && [ "$TAG_COUNT" -gt 0 ]; then
    echo "  --- Tags ---"
    echo "$TAG_OUTPUT" | head -30
    [ "$TAG_COUNT" -gt 30 ] && echo "  ... and $((TAG_COUNT - 30)) more"
fi

# ==================================================
section "7/7" "Backlog Integrity"

# GUI-independent: detect backlog drift in project task files — duplicate ticket
# IDs and status contradictions — via check-backlog-integrity.sh (SDD-012).
# Incident->guard sibling of the working-tree (sec 1) and \n-escape checks.
if [ ! -d "$VAULT_DIR" ]; then
    skip "Backlog integrity" "vault dir not found"
else
    BACKLOG_FILES=0
    BACKLOG_BAD=0
    for tasks in "$VAULT_DIR"/10_projects/*/11-tasks.md; do
        [ -f "$tasks" ] || continue
        BACKLOG_FILES=$((BACKLOG_FILES + 1))
        if ! "$SCRIPT_DIR/check-backlog-integrity.sh" "$tasks" >/dev/null 2>&1; then
            BACKLOG_BAD=$((BACKLOG_BAD + 1))
            fail "Backlog drift in $(basename "$(dirname "$tasks")")/11-tasks.md (duplicate IDs / status contradictions)"
            "$SCRIPT_DIR/check-backlog-integrity.sh" "$tasks" 2>/dev/null | sed 's|^|        |'
        fi
    done
    if [ "$BACKLOG_FILES" -eq 0 ]; then
        skip "Backlog integrity" "no 10_projects/*/11-tasks.md found"
    elif [ "$BACKLOG_BAD" -eq 0 ]; then
        pass "Backlog integrity: $BACKLOG_FILES task file(s) clean (one ticket = one entry)"
    fi

    # Semantic drift (advisory): open [ ] entries whose work already shipped
    # (full id has an archived spec) via check-backlog-merged.sh (SDD-012b).
    # WARN, not fail — the signal is strong but the tick is a human/agent action.
    for tasks in "$VAULT_DIR"/10_projects/*/11-tasks.md; do
        [ -f "$tasks" ] || continue
        if ! merged_out="$("$SCRIPT_DIR/check-backlog-merged.sh" "$tasks" 2>/dev/null)"; then
            warn "Stale-merged ticks in $(basename "$(dirname "$tasks")")/11-tasks.md — work shipped, tick still [ ]:"
            printf '%s\n' "$merged_out" | sed 's|^|        |'
        fi
    done
fi

# ==================================================
echo ""
echo "========================================"
printf 'Results: %b%d passed%b, %b%d failed%b, %b%d skipped%b\n' \
    "$GREEN" "$CHECKS_PASSED" "$NC" \
    "$RED" "$CHECKS_FAILED" "$NC" \
    "$YELLOW" "$CHECKS_SKIPPED" "$NC"
echo "========================================"

if [ "$CHECKS_FAILED" -gt 0 ]; then
    exit 1
fi
exit 0
