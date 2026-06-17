#!/usr/bin/env bash
#
# memory-sink-guard.sh (GUARD-001) — block agent-memory artifacts from being
# committed to any repo that is NOT the knowledge vault. The keystone of the
# single-sink convention: memory lives only in the vault.
#
# Invoked by the global pre-commit dispatcher. Exits 0 (allow) when the repo is
# the vault or no memory artifact is staged; exits 1 (block) with a message
# otherwise. `git commit --no-verify` is the documented escape hatch.
#
# Env: VAULT_PATH (default $HOME/Projects/knowledge).

set -u

toplevel="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[ -n "$toplevel" ] || exit 0

VAULT="${VAULT_PATH:-$HOME/Projects/knowledge}"

# Vault detection: the sentinel .obsidian/ at the repo root (cross-machine) or
# the repo root being the configured vault path. The vault is the allowed sink.
if [ -d "$toplevel/.obsidian" ] || [ "$toplevel" = "$VAULT" ]; then
    exit 0
fi

# Scan staged additions/modifications for memory artifacts: a MEMORY.md file in
# any directory, or any path under a memory/ directory. Narrow by design — a
# legitimate need is unblocked with --no-verify (user approval).
staged="$(git diff --cached --name-only --diff-filter=AM 2>/dev/null)"
[ -n "$staged" ] || exit 0

offenders="$(printf '%s\n' "$staged" | grep -E '(^|/)MEMORY\.md$|(^|/)memory/' || true)"
[ -n "$offenders" ] || exit 0

{
    echo "✖ GUARD-001: agent memory belongs only in the knowledge vault (single-sink convention)."
    echo "  This repo is not the vault, so these staged paths are blocked:"
    printf '%s\n' "$offenders" | sed 's/^/      /'
    echo "  Move them into the vault. To bypass with approval: git commit --no-verify"
} >&2
exit 1
