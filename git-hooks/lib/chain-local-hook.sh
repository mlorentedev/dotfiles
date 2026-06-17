#!/usr/bin/env bash
#
# chain-local-hook.sh <hook-type> [args...] — exec the repo-local hook of the
# given type, if present. A global core.hooksPath makes git ignore .git/hooks/
# for ALL hook types; this restores per-repo hooks by exec-ing the literal path.
# A clean no-op (exit 0) when no local hook exists.

set -u

hook_type="${1:?hook type required}"
shift

toplevel="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
local_hook="$toplevel/.git/hooks/$hook_type"
[ -x "$local_hook" ] && exec "$local_hook" "$@"
exit 0
