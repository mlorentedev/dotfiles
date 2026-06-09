#!/usr/bin/env bash
# dotfiles-selfupdate.sh -- OPS-001
#
# Opt-in self-deploy entrypoint, run by the systemd --user timer (Linux) /
# Scheduled Task (Windows). Pulls the dotfiles repo from its git remote and
# re-runs the idempotent setup -- but ONLY via a clean fast-forward, and ONLY
# when HEAD actually moved. Anything else (dirty worktree, diverged history,
# unreachable remote, no upstream) is logged and skipped with exit 0. A real
# setup failure is the only non-zero exit, so `systemctl --user status
# dotfiles-selfupdate` (and the journal) surface it.
#
# This is the OPPOSITE direction to dotfiles-sync.sh (which pushes the repo and
# rsyncs repo -> ~/.dotfiles). Kept as a separate script on purpose (SRP).
#
# Usage: dotfiles-selfupdate.sh
# Env:
#   DOTFILES_REPO_DIR              repo to update   (default: $HOME/Projects/dotfiles)
#   DOTFILES_SELFUPDATE_SETUP_CMD  setup to re-run  (default: <repo>/setup-linux.sh)

set -euo pipefail

REPO="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"
SETUP_CMD="${DOTFILES_SELFUPDATE_SETUP_CMD:-$REPO/setup-linux.sh}"

# Colors only on a real terminal (a timer run has none).
if [ -t 1 ]; then
    BLUE='\033[0;34m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; RED='\033[0;31m'; NC='\033[0m'
else
    BLUE='' GREEN='' YELLOW='' RED='' NC=''
fi
log_info()    { printf '%b->%b %s\n'    "$BLUE"   "$NC" "$1"; }
log_success() { printf '%b[ok]%b %s\n'  "$GREEN"  "$NC" "$1"; }
log_skip()    { printf '%b[skip]%b %s\n' "$YELLOW" "$NC" "$1"; }
log_error()   { printf '%b[err]%b %s\n' "$RED"    "$NC" "$1" >&2; }

# --- guard: must be a git repo ---
if ! git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1; then
    log_skip "Not a git repo: $REPO -- nothing to self-update"
    exit 0
fi

# --- guard: never touch a dirty worktree (the primary failure mode) ---
if [ -n "$(git -C "$REPO" status --porcelain 2>/dev/null)" ]; then
    log_skip "Dirty worktree in $REPO -- skipping (commit or stash your changes first)"
    exit 0
fi

# --- fetch; a network failure is transient -> skip and retry next slot ---
if ! git -C "$REPO" fetch --quiet 2>/dev/null; then
    log_skip "git fetch failed (network?) -- skipping self-update this run"
    exit 0
fi

# --- guard: an upstream must be configured for the current branch ---
if ! upstream="$(git -C "$REPO" rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null)"; then
    log_skip "No upstream configured for the current branch -- skipping"
    exit 0
fi

local_rev="$(git -C "$REPO" rev-parse HEAD)"
remote_rev="$(git -C "$REPO" rev-parse '@{u}')"
base_rev="$(git -C "$REPO" merge-base HEAD '@{u}')"

# --- already current: no deploy needed ---
if [ "$local_rev" = "$remote_rev" ]; then
    log_info "Already current ($upstream) -- no deploy needed"
    exit 0
fi

# --- diverged / non-fast-forward: never merge, rebase, or reset unattended ---
if [ "$base_rev" != "$local_rev" ]; then
    log_skip "Local branch has diverged from $upstream (non fast-forward) -- skipping"
    exit 0
fi

# --- clean fast-forward, then re-run the idempotent setup ---
log_info "Fast-forwarding to $upstream ..."
if ! git -C "$REPO" merge --ff-only '@{u}' >/dev/null 2>&1; then
    log_skip "Fast-forward failed unexpectedly -- skipping (worktree left untouched)"
    exit 0
fi
log_success "Updated $(git -C "$REPO" rev-parse --short "$local_rev") -> $(git -C "$REPO" rev-parse --short HEAD)"

log_info "Re-running setup: $SETUP_CMD"
if "$SETUP_CMD"; then
    log_success "Self-update complete"
else
    rc=$?
    log_error "Setup failed (exit $rc) -- see output above"
    exit "$rc"
fi
