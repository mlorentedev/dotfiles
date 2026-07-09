#!/usr/bin/env bash
# install-git-hooks.sh — deploy the GUARD-001 global memory-sink dispatcher and
# wire it machine-wide via core.hooksPath.
#
# This is the "place the dispatcher" half of GUARD-001 (#398). #415 shipped the
# dispatcher (git-hooks/) and the `dotf doctor` verifier, but nothing deployed
# the dispatcher to the ~/.dotfiles mirror or wired the global hook — so the
# guard was inert (#418). Bootstrap stays shell (ADR-020 C7): the deployed `dotf`
# release cannot self-deploy git-hooks/ because it carries no source tree, so the
# setup bootstrap places them and `dotf doctor` verifies the wiring thereafter.
#
# Sourced by setup-linux.sh; also runnable standalone to (re)install:
#     ./scripts/install-git-hooks.sh [src_dir] [dotfiles_dir]
#
# Idempotent (clean mirror + converge), safety-first (an unrelated pre-existing
# core.hooksPath is preserved, never clobbered — a global hooksPath has
# machine-wide blast radius). Cross-shell bash+zsh; the hooks themselves are
# POSIX (they run under Git-for-Windows `sh` too — a Windows setup twin is a
# tracked follow-up). Functions take src/dest as args so bats can drive them
# against fixtures under an isolated HOME with no real ~/.gitconfig mutation.

_GH_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

# Load logging helpers if the caller (setup) has not already sourced utils.sh.
if ! command -v log_info >/dev/null 2>&1; then
    # shellcheck source=/dev/null
    . "$_GH_SCRIPT_DIR/utils.sh"
fi

# deploy_git_hooks <src_dir> <dest_dir>: clean-mirror the dispatcher tree into
# the deploy mirror and make the entrypoints executable. A clean mirror (not a
# bare `cp`) so a hook removed upstream never lingers in the mirror and keeps
# firing — a stale security hook is worse than none.
deploy_git_hooks() {
    local src="$1" dest="$2"

    if [ ! -d "$src" ]; then
        log_error "install-git-hooks: source dispatcher dir not found: $src"
        return 1
    fi
    if [ ! -f "$src/pre-commit" ]; then
        log_error "install-git-hooks: $src has no pre-commit dispatcher — refusing to deploy"
        return 1
    fi

    # Guard the destructive mirror: the dest MUST be a non-root */git-hooks path.
    # Defends against an empty/misconfigured DOTFILES_DIR turning rm -rf loose.
    case "$dest" in
        */git-hooks) : ;;
        *) log_error "install-git-hooks: refusing to mirror to unexpected path: '$dest'"; return 1 ;;
    esac
    if [ "$dest" = "/git-hooks" ] || [ "$dest" = "$HOME/git-hooks" ]; then
        log_error "install-git-hooks: refusing unsafe dest: '$dest'"
        return 1
    fi

    # Defense in depth (#695): if src and dest are the same directory, the
    # clean-mirror below (rm -rf dest THEN cp src/. dest) would delete the
    # dispatcher and copy nothing back — emptying it while still reporting
    # success. The setup preflight already refuses the in-place layout that
    # causes this, but the function must be safe on its own. Nothing to mirror
    # onto itself; just refresh the executable bits.
    if [ "$src" -ef "$dest" ]; then
        log_info "install-git-hooks: src and dest are the same directory ($dest) — skipping mirror (already in place)"
        chmod +x "$dest/pre-commit" "$dest/commit-msg" "$dest/prepare-commit-msg" "$dest/pre-push" "$dest/post-checkout" 2>/dev/null || true
        [ -d "$dest/lib" ] && chmod +x "$dest"/lib/*.sh 2>/dev/null || true
        log_success "GUARD dispatcher already in place at $dest"
        return 0
    fi

    rm -rf "$dest"
    mkdir -p "$dest"
    cp -rf "$src/." "$dest/"

    # git execs these directly; the lib helpers are sourced/exec'd by them.
    chmod +x "$dest/pre-commit" "$dest/commit-msg" "$dest/prepare-commit-msg" "$dest/pre-push" "$dest/post-checkout" 2>/dev/null || true
    if [ -d "$dest/lib" ]; then
        chmod +x "$dest"/lib/*.sh 2>/dev/null || true
    fi
    log_success "GUARD dispatcher deployed to $dest"
    return 0
}

# wire_global_hooks_path <target_dir>: point git's global core.hooksPath at the
# deployed dispatcher — but ONLY when unset. An unrelated value is preserved
# (machine-wide blast radius) and surfaced as a warning; an already-correct value
# is a no-op. Mirrors the `dotf doctor` checkGuardHooks contract.
wire_global_hooks_path() {
    local target="$1" current
    current="$(git config --global --get core.hooksPath 2>/dev/null || true)"

    if [ -z "$current" ]; then
        if git config --global core.hooksPath "$target"; then
            log_success "core.hooksPath wired to the GUARD dispatcher ($target)"
        else
            log_error "install-git-hooks: failed to set core.hooksPath"
            return 1
        fi
    elif [ "$current" = "$target" ]; then
        log_info "core.hooksPath already wired to the GUARD dispatcher"
    else
        log_warning "core.hooksPath is '$current' (not the GUARD dispatcher) — preserving it; the memory-sink guard is INACTIVE. Point it at $target, or chain the dispatcher from your hooks, manually."
    fi
    return 0
}

# install_git_hooks [src_dir] [dotfiles_dir]: deploy + wire. Defaults resolve the
# repo's git-hooks/ (this script's parent) and the ~/.dotfiles deploy mirror.
install_git_hooks() {
    local src="${1:-$(cd "$_GH_SCRIPT_DIR/.." && pwd)/git-hooks}"
    local dotfiles_dir="${2:-${DOTFILES_DIR:-$HOME/.dotfiles}}"

    if [ -z "$dotfiles_dir" ] || [ "$dotfiles_dir" = "/" ] || [ "$dotfiles_dir" = "$HOME" ]; then
        log_error "install-git-hooks: unsafe DOTFILES_DIR: '$dotfiles_dir'"
        return 1
    fi

    local dest="$dotfiles_dir/git-hooks"
    deploy_git_hooks "$src" "$dest" || return 1
    wire_global_hooks_path "$dest" || return 1
    return 0
}

# Run standalone only when executed directly (not when sourced by setup).
if [ "${BASH_SOURCE[0]:-$0}" = "$0" ]; then
    install_git_hooks "$@"
fi
