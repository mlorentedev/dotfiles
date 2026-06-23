#!/usr/bin/env bash
# session-handoff.sh — MEMORY-001 cross-agent session bridge (implements ADR-014).
#
# Wired to Claude Code's `SessionEnd` hook. Reads the hook JSON on stdin and
# ARCHIVES the `## Session Handoff` block that the /handoff skill wrote into the
# project's MEMORY.md into an append-only record:
#     $VAULT_PATH/10_projects/<project>/sessions/<date>-<project>-claude.md
# The agent authors the handoff (via /handoff, with reasoning); this hook only
# persists it as durable cross-session history.
#
# Resilience contract: a session-end hook must NEVER crash a session. Every
# trivial / missing / malformed input is a clean no-op (exit 0, no file).
#
# Usage: session-handoff.sh        (SessionEnd JSON payload on stdin)

VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"

# 1. Read the payload; empty stdin -> no-op.
payload="$(cat 2>/dev/null)"
[ -n "$payload" ] || exit 0
command -v jq >/dev/null 2>&1 || exit 0

# 2. Extract cwd + session_id (graceful on malformed JSON).
cwd="$(printf '%s' "$payload" | jq -r '.cwd // empty' 2>/dev/null)"
sid="$(printf '%s' "$payload" | jq -r '.session_id // empty' 2>/dev/null)"
[ -n "$cwd" ] || exit 0
project="$(basename "$cwd")"

# 3. Locate the project's MEMORY.md; absent -> no-op.
memory="$VAULT_PATH/10_projects/$project/memory/MEMORY.md"
[ -f "$memory" ] || exit 0

# 4. Extract the `## Session Handoff` block (from after the heading to the next
#    `## ` heading or EOF). Empty/whitespace block -> trivial session, no-op.
block="$(awk '
    /^## Session Handoff[[:space:]]*$/ { cap=1; next }
    cap && /^## / { exit }
    cap { print }
' "$memory")"
[ -n "$(printf '%s' "$block" | tr -d '[:space:]')" ] || exit 0

# 5. Append a durable, timestamped session record.
date="$(date -u +%Y-%m-%d)"
out_dir="$VAULT_PATH/10_projects/$project/sessions"
mkdir -p "$out_dir" 2>/dev/null || exit 0
out="$out_dir/$date-$project-claude.md"

{
    printf -- '---\n'
    printf 'id: "session-%s-%s-claude"\n' "$date" "$project"
    printf 'type: session\n'
    printf 'session_id: %s\n' "${sid:-unknown}"
    printf 'agent: claude\n'
    printf 'project: %s\n' "$project"
    printf 'date: "%s"\n' "$date"
    printf 'tags: [session, handoff, %s]\n' "$project"
    printf -- '---\n\n'
    printf '# Session %s — %s (claude)\n\n' "$date" "$project"
    printf '%s\n' "$block"
} > "$out" 2>/dev/null || exit 0

exit 0
