#!/usr/bin/env bash
# obs-cli.sh: Thin wrapper around Obsidian CLI
# Handles --no-sandbox (Linux), default vault, and GUI connectivity check.
# Usage: obs-cli.sh <command> [args...]
# Exit: 0 = success, 1 = error, 2 = GUI not running

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    # shellcheck source=utils.sh disable=SC1091
    . "$SCRIPT_DIR/utils.sh"
else
    echo "Error: utils.sh not found in $SCRIPT_DIR" >&2
    exit 1
fi

OBS_VAULT="${OBS_VAULT:-knowledge}"

usage() {
    printf 'Usage: obs-cli.sh <command> [args...]\n\n'
    printf 'Wrapper for Obsidian CLI with --no-sandbox and default vault.\n\n'
    printf 'Environment:\n'
    printf '  OBS_VAULT   Vault name (default: knowledge)\n\n'
    printf 'Examples:\n'
    printf '  obs-cli.sh backlinks file="path/to/note.md"\n'
    printf '  obs-cli.sh orphans\n'
    printf '  obs-cli.sh tags\n'
    printf '  obs-cli.sh tags:rename old=wip new=in-progress\n'
    printf '  obs-cli.sh search query="pattern"\n'
    printf '  obs-cli.sh eval "app.vault.getFiles().length"\n'
}

if [ $# -eq 0 ] || [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    usage
    exit 0
fi

if ! command_exists obsidian; then
    log_error "Obsidian not found in PATH"
    exit 1
fi

# Connectivity check: verify GUI is running
obsidian --no-sandbox vault --vault "$OBS_VAULT" >/dev/null 2>&1 || fail_obsidian_gui

exec obsidian --no-sandbox "$@" --vault "$OBS_VAULT"
