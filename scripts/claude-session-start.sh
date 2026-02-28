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

if [ -z "$VAULT_ROOT" ]; then
    # Not inside a vault — exit cleanly, no context to add
    exit 0
fi

VAULT_NAME=$(basename "$VAULT_ROOT")
CONTEXT_LINES="Obsidian vault detected: $VAULT_NAME ($VAULT_ROOT)"

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

# Return context to Claude via hook output format
jq -n --arg ctx "$CONTEXT_LINES" '{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": $ctx
  }
}'

exit 0
