#!/bin/bash
# init-repo-agents.sh
# Purpose: Bootstrap AGENTS.md in a repo with the Spec-Driven Development snippet from the vault.
#          Idempotent — re-running is safe (skips if section already present).
#
# Usage:
#   ./init-repo-agents.sh [--repo <path>] [--force]
#
# See: $VAULT_PATH/00_meta/templates/agents-spec-section.md for the snippet source.

set -euo pipefail

usage() {
    cat <<EOF
Usage: init-repo-agents.sh [--repo <path>] [--force]

  --repo <path>    target repo root (default: current git repo)
  --force          overwrite existing SDD section in AGENTS.md
  -h, --help       show this help

VAULT_PATH (env)   vault root (default: \$HOME/Projects/knowledge)
EOF
}

# --- Parse args ---
TARGET_REPO=""
FORCE=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --repo) TARGET_REPO="$2"; shift 2 ;;
        --force) FORCE=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) printf '[ERROR] Unknown option: %s\n' "$1" >&2; usage >&2; exit 1 ;;
    esac
done

# --- Resolve target repo ---
if [[ -z "$TARGET_REPO" ]]; then
    if ! TARGET_REPO="$(git rev-parse --show-toplevel 2>/dev/null)"; then
        printf '[ERROR] Not in a git repo. Use --repo <path> or cd into a repo.\n' >&2
        exit 1
    fi
fi

if [[ ! -d "$TARGET_REPO" ]]; then
    printf '[ERROR] Repo not found: %s\n' "$TARGET_REPO" >&2
    exit 1
fi

# --- Resolve vault snippet ---
VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"
SNIPPET_SRC="$VAULT_PATH/00_meta/templates/agents-spec-section.md"

if [[ ! -f "$SNIPPET_SRC" ]]; then
    printf '[ERROR] Snippet template not found: %s\n' "$SNIPPET_SRC" >&2
    # shellcheck disable=SC2016
    printf '        Set %s env var to vault root, or clone the vault.\n' '$VAULT_PATH' >&2
    exit 2
fi

# --- Extract snippet content (between markers, exclusive) ---
SNIPPET="$(awk '
    /^## --- BEGIN SNIPPET ---$/ {flag=1; next}
    /^## --- END SNIPPET ---$/   {flag=0; next}
    flag {print}
' "$SNIPPET_SRC")"

if [[ -z "$SNIPPET" ]]; then
    printf '[ERROR] BEGIN/END SNIPPET markers not found in %s\n' "$SNIPPET_SRC" >&2
    exit 2
fi

# --- Target AGENTS.md ---
AGENTS_FILE="$TARGET_REPO/AGENTS.md"

# --- Idempotency / force ---
if [[ -f "$AGENTS_FILE" ]] && grep -q '^## Spec-Driven Development$' "$AGENTS_FILE"; then
    if [[ "$FORCE" -eq 0 ]]; then
        printf '[OK] Spec-Driven Development section already present in %s\n' "$AGENTS_FILE"
        printf '     Re-run with --force to overwrite.\n'
        exit 0
    fi
    # Replace existing section (from heading until next ^## or EOF)
    awk -v snippet="$SNIPPET" '
        /^## Spec-Driven Development$/ {skip=1; print snippet; next}
        skip && /^## / && !/^## Spec-Driven Development$/ {skip=0}
        !skip {print}
    ' "$AGENTS_FILE" > "${AGENTS_FILE}.tmp" && mv "${AGENTS_FILE}.tmp" "$AGENTS_FILE"
    printf '[OK] Replaced Spec-Driven Development section in %s\n' "$AGENTS_FILE"
    exit 0
fi

# --- Append or create ---
if [[ -f "$AGENTS_FILE" ]]; then
    # Append with leading blank line separation
    {
        printf '\n'
        printf '%s\n' "$SNIPPET"
    } >> "$AGENTS_FILE"
    printf '[OK] Appended Spec-Driven Development section to %s\n' "$AGENTS_FILE"
else
    # Create new with minimal header
    cat > "$AGENTS_FILE" <<EOF
# AGENTS.md

> Instructions for AI coding agents (Claude Code, Copilot, Cursor, Codex) operating in this repo.

$SNIPPET
EOF
    printf '[OK] Created %s with Spec-Driven Development section\n' "$AGENTS_FILE"
fi
