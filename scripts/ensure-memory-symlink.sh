#!/usr/bin/env bash
#
# ensure-memory-symlink.sh — agent-agnostic vault->memory symlink (MEMORY-002).
#
# Links an agent's per-project memory directory to its source in the knowledge
# vault, so agent memory lives in the single sink (the vault) and is surfaced
# wherever the agent expects it. Extracted from claude-session-start.sh's
# `ensure_memory_symlink` so every agent — and `dotf init` — can reuse the same
# vault-source resolution instead of it living inside the Claude-only hook.
#
# The resolver (three conventions below) is agnostic; the per-agent TARGET path
# is an argument. Claude's path encoding stays in the Claude hook (the caller),
# not here.
#
# Usage:
#   ensure-memory-symlink.sh --cwd <project-dir> --target <agent-memory-path>
#                            [--project <name>]
#
#   --cwd      Project working directory (used to resolve the vault source).
#   --target   Where the symlink is created (the agent's memory dir path).
#   --project  Override the project name (default: basename of --cwd).
#
# Env:
#   VAULT_PATH   Vault root (default: $HOME/Projects/knowledge).
#
# Behaviour: best-effort and idempotent. Creates the symlink only when a vault
# source exists and the target is safe to replace; otherwise a clean no-op.
# Exits 0 on every resolvable outcome so it can never fail a session start.
# Cross-OS: POSIX/Linux/macOS symlink path; the Windows memory layout differs
# and is not wired here (MEMORY-002 R4 — follow-up).

set -u

CWD=""
TARGET=""
PROJECT=""

while [ $# -gt 0 ]; do
    case "$1" in
        --cwd)     CWD="${2:-}"; shift 2 ;;
        --target)  TARGET="${2:-}"; shift 2 ;;
        --project) PROJECT="${2:-}"; shift 2 ;;
        -h|--help)
            grep '^#' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) echo "ensure-memory-symlink: unknown argument: $1" >&2; exit 2 ;;
    esac
done

if [ -z "$CWD" ] || [ -z "$TARGET" ]; then
    echo "ensure-memory-symlink: --cwd and --target are required" >&2
    exit 2
fi

VAULT="${VAULT_PATH:-$HOME/Projects/knowledge}"
[ -n "$PROJECT" ] || PROJECT="$(basename "$CWD")"

# Already linked, or a non-empty real dir (the agent's own data)? Leave it.
if [ -L "$TARGET" ]; then exit 0; fi
if [ -d "$TARGET" ] && [ "$(ls -A "$TARGET" 2>/dev/null)" ]; then exit 0; fi

# --- Resolve the vault memory source (three conventions, in precedence order) ---

# 1) 10_projects/<name>/memory/ (personal projects convention)
vault_memory="$VAULT/10_projects/$PROJECT/memory"

# 2) CWD/memory/ (sessions whose CWD is inside the vault itself)
if [ ! -d "$vault_memory" ]; then
    case "$CWD" in
        "$VAULT"*) vault_memory="$CWD/memory" ;;
        *) vault_memory="" ;;
    esac
fi

# 3) 50_work/45-development/<family>/<component>/memory/ (nested work SDK repos)
if [ -z "$vault_memory" ] || [ ! -d "$vault_memory" ]; then
    dev_dir="$VAULT/50_work/45-development"
    if [ -d "$dev_dir" ]; then
        cwd_path_slug=$(printf '%s' "$CWD" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9/')
        for sdk_family_dir in "$dev_dir"/*/; do
            [ -d "$sdk_family_dir" ] || continue
            sdk_family_slug=$(basename "$sdk_family_dir" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')
            if printf '%s' "$cwd_path_slug" | grep -q "$sdk_family_slug"; then
                for sdk_comp_dir in "$sdk_family_dir"*/; do
                    [ -d "$sdk_comp_dir" ] || continue
                    sdk_comp_slug=$(basename "$sdk_comp_dir" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9')
                    if printf '%s' "$cwd_path_slug" | grep -q "$sdk_comp_slug"; then
                        vault_memory="${sdk_comp_dir%/}/memory"
                        break 2
                    fi
                done
            fi
        done
    fi
fi

# No source found anywhere -> clean no-op.
[ -n "$vault_memory" ] && [ -d "$vault_memory" ] || exit 0

# --- Create the symlink ---

parent_dir="$(dirname "$TARGET")"
mkdir -p "$parent_dir"

# Remove an empty placeholder target dir (no files, not a symlink) so ln succeeds.
if [ -d "$TARGET" ] && [ -z "$(ls -A "$TARGET" 2>/dev/null)" ]; then
    rmdir "$TARGET" 2>/dev/null || true
fi

if ln -s "$vault_memory" "$TARGET" 2>/dev/null; then
    echo "[auto-memory] Created symlink for $PROJECT"
fi

exit 0
