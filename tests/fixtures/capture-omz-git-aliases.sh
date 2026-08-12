#!/usr/bin/env bash
# Regenerate tests/fixtures/omz-git-plugin-aliases.txt from the oh-my-zsh git
# plugin actually installed on this machine (BUG-045).
#
# Run this ONLY to deliberately refresh the vendored snapshot, and say why in
# the commit -- e.g. "oh-my-zsh git plugin added alias X upstream". Never run
# it to make a failing collision test go green without reading why it failed
# first: a new offender in the fixture is signal, not noise.
#
# Usage:
#   tests/fixtures/capture-omz-git-aliases.sh

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PLUGIN="${ZSH:-$HOME/.oh-my-zsh}/plugins/git/git.plugin.zsh"
OUT="$HERE/omz-git-plugin-aliases.txt"

[ -f "$PLUGIN" ] || {
    printf 'oh-my-zsh git plugin not found at %s\n' "$PLUGIN" >&2
    exit 1
}

commit_note="unknown"
if command -v git >/dev/null 2>&1 && [ -d "${ZSH:-$HOME/.oh-my-zsh}/.git" ]; then
    commit_note="$(cd "${ZSH:-$HOME/.oh-my-zsh}" && git rev-parse --short HEAD 2>/dev/null || echo unknown)"
fi

{
    printf "# Vendored snapshot of alias names owned by oh-my-zsh's git plugin.\n"
    printf '# Regenerate with: tests/fixtures/capture-omz-git-aliases.sh\n'
    printf '#\n'
    printf '# Names only (not the plugin file itself) -- this is a comparison target for\n'
    printf '# the g* namespace collision guard (BUG-045), not a copy of upstream logic.\n'
    printf '#\n'
    printf '# Upstream: https://github.com/ohmyzsh/ohmyzsh, plugins/git/git.plugin.zsh\n'
    printf '# Captured: %s\n' "$(date -u +%Y-%m-%d)"
    printf '# Upstream commit (informational): %s\n' "$commit_note"
    grep -hoE "^[[:space:]]*alias[[:space:]]+(-g[[:space:]]+)?[a-zA-Z_][a-zA-Z0-9_-]*=" "$PLUGIN" \
        | sed -E 's/^[[:space:]]*alias[[:space:]]+(-g[[:space:]]+)?//; s/=$//' \
        | LC_ALL=C sort -u
} > "$OUT"

printf 'captured %s aliases to %s\n' "$(grep -vc '^#' "$OUT")" "$OUT"
