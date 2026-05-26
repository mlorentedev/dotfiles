#!/usr/bin/env bash
# cleanup-legacy-ai.sh: one-shot script to remove legacy AI tooling artifacts
# from the user's machine. NOT auto-run by setup-linux.sh — user invokes when ready.
#
# Removes (after per-category confirmation):
#   - legacy `@google/gemini-cli` npm install (replaced by `agy`)
#   - `aider-chat` Python install via pipx/pip (sunset by SDD-007)
#   - ~/.config/aider/ config directory
#   - ~/.aider* cache files in the user's home
#
# Does NOT touch:
#   - ~/.gemini/      — that's agy's home dir (kept the path from gemini-cli)
#   - ~/.dotfiles/    — managed by setup-linux.sh
#   - ai/agy/         — agy stays (replacement for legacy gemini-cli)
#
# Usage:
#   scripts/cleanup-legacy-ai.sh              # dry-run + prompts per category
#   scripts/cleanup-legacy-ai.sh --yes        # auto-confirm everything
#   scripts/cleanup-legacy-ai.sh --dry-run    # only list, never touch

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    . "$SCRIPT_DIR/utils.sh"
else
    echo "Error: scripts/utils.sh not found" >&2
    exit 1
fi

AUTO_YES=0
DRY_RUN=0
for arg in "$@"; do
    case "$arg" in
        --yes|-y) AUTO_YES=1 ;;
        --dry-run|-n) DRY_RUN=1 ;;
        --help|-h)
            sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) log_warning "Unknown arg: $arg (try --help)" ;;
    esac
done

confirm() {
    [ "$AUTO_YES" -eq 1 ] && return 0
    [ "$DRY_RUN" -eq 1 ] && return 1
    printf '  Proceed? [y/N] '
    read -r reply
    case "$reply" in
        [yY]|[yY][eE][sS]) return 0 ;;
        *) return 1 ;;
    esac
}

run_or_dry() {
    if [ "$DRY_RUN" -eq 1 ]; then
        log_info "  [dry-run] would run: $*"
        return 0
    fi
    log_info "  running: $*"
    eval "$@" || log_warning "  command exited non-zero (continuing)"
}

# ---------------------------------------------------------------------------
# Category 1: legacy gemini-cli npm binary
# ---------------------------------------------------------------------------
log_info "[1/3] Legacy @google/gemini-cli (replaced by agy)"
if command -v npm >/dev/null 2>&1; then
    gemini_pkg="$(npm ls -g --depth=0 2>/dev/null | grep -E '@google/gemini-cli' || true)"
    if [ -n "$gemini_pkg" ]; then
        log_warning "  Found global npm install: $gemini_pkg"
        if confirm; then
            run_or_dry "npm uninstall -g @google/gemini-cli"
        else
            log_info "  Skipped."
        fi
    else
        log_success "  No global @google/gemini-cli install found."
    fi
else
    log_info "  npm not on PATH; skipping."
fi

# ---------------------------------------------------------------------------
# Category 2: aider Python package
# ---------------------------------------------------------------------------
log_info "[2/3] Aider (sunset by SDD-007, OpenCode covers same use case)"
found_aider=0
if command -v pipx >/dev/null 2>&1 && pipx list 2>/dev/null | grep -qE 'aider-chat|^   package aider'; then
    found_aider=1
    log_warning "  Found pipx install: aider-chat"
    if confirm; then
        run_or_dry "pipx uninstall aider-chat"
    else
        log_info "  Skipped pipx uninstall."
    fi
fi
if command -v pip >/dev/null 2>&1 && pip show aider-chat >/dev/null 2>&1; then
    found_aider=1
    log_warning "  Found pip install: aider-chat (user-level)"
    if confirm; then
        run_or_dry "pip uninstall -y aider-chat"
    else
        log_info "  Skipped pip uninstall."
    fi
fi
[ "$found_aider" -eq 0 ] && log_success "  No aider install found."

# ---------------------------------------------------------------------------
# Category 3: stale config + cache directories
# ---------------------------------------------------------------------------
log_info "[3/3] Stale config / cache directories"
to_remove=()
[ -d "$HOME/.config/aider" ] && to_remove+=("$HOME/.config/aider")
[ -d "$HOME/.aider" ] && to_remove+=("$HOME/.aider")
for f in "$HOME"/.aider.tags.cache.v* "$HOME"/.aider.input.history "$HOME"/.aider.chat.history.md; do
    [ -e "$f" ] && to_remove+=("$f")
done

if [ "${#to_remove[@]}" -eq 0 ]; then
    log_success "  No stale aider directories or cache files found."
else
    for path in "${to_remove[@]}"; do
        log_warning "  Found: $path"
    done
    if confirm; then
        for path in "${to_remove[@]}"; do
            run_or_dry "rm -rf '$path'"
        done
    else
        log_info "  Skipped."
    fi
fi

log_success "Cleanup pass complete."
[ "$DRY_RUN" -eq 1 ] && log_info "Dry-run only — nothing was changed. Re-run without --dry-run to apply."
