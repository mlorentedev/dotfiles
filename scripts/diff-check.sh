#!/bin/bash

# diff-check.sh: Detect drift between repo and deploy-dir (~/.dotfiles).
#
# The repo (~/Projects/dotfiles) is the source of truth. Setup-linux.sh
# COPIES files into ~/.dotfiles (the deploy-dir), then symlinks from $HOME
# into the deploy-dir. If you edit a file in the repo without re-running
# setup, the repo and deploy-dir diverge silently — every shell still reads
# the stale deploy-dir copy. This script reports those divergences.
#
# Usage: ./scripts/diff-check.sh [--verbose|-v] [--help|-h]
# Exit:  0 if no drift, 1 if drift detected, 2 on usage/setup error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    # shellcheck source=utils.sh disable=SC1091
    . "$SCRIPT_DIR/utils.sh"
else
    echo "Error: utils.sh not found in $SCRIPT_DIR" >&2
    exit 2
fi

verbose=false
for arg in "$@"; do
    case "$arg" in
        --verbose|-v) verbose=true ;;
        --help|-h)
            cat <<'EOF'
Usage: diff-check.sh [--verbose|-v]

Reports files that differ between the repo and the deploy-dir (~/.dotfiles).
Both must exist; only files tracked by git in the repo are checked.

Env:
  DOTFILES_REPO_DIR   Repo root (default: parent of script's dir)
  DOTFILES_DIR        Deploy-dir (default: ~/.dotfiles)

Exit codes:
  0  no drift
  1  drift detected
  2  setup error (missing dirs, not a git repo, etc.)
EOF
            exit 0
            ;;
        *)
            echo "Unknown argument: $arg" >&2
            echo "Run with --help for usage" >&2
            exit 2
            ;;
    esac
done

REPO_DIR="${DOTFILES_REPO_DIR:-$(dirname "$SCRIPT_DIR")}"
DEPLOY_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"

[ -d "$REPO_DIR" ]   || { echo "Error: repo not found: $REPO_DIR" >&2; exit 2; }
[ -d "$DEPLOY_DIR" ] || { echo "Error: deploy-dir not found: $DEPLOY_DIR" >&2; exit 2; }
[ -d "$REPO_DIR/.git" ] || { echo "Error: not a git repo: $REPO_DIR" >&2; exit 2; }

# Top-level files and dirs that setup-linux.sh copies into DEPLOY_DIR.
# MUST mirror the copy block in setup-linux.sh (around the DOTFILES_DIR section).
# Anything else in DEPLOY_DIR is legacy cruft from older sync mechanisms or
# runtime state — not "drift" against current setup, so we skip it.
_is_managed() {
    case "$1" in
        versions.conf|.zshrc|.bashrc|.profile|.gitconfig|tmux.conf) return 0 ;;
        .zsh/*|ssh/*|scripts/*|sensitive/*) return 0 ;;
        *) return 1 ;;
    esac
}

log_info "Checking drift: $REPO_DIR ↔ $DEPLOY_DIR"

drift_count=0
checked_count=0
unmanaged_count=0

while IFS= read -r rel_path; do
    [ -z "$rel_path" ] && continue

    if ! _is_managed "$rel_path"; then
        unmanaged_count=$((unmanaged_count + 1))
        continue
    fi

    repo_file="$REPO_DIR/$rel_path"
    deploy_file="$DEPLOY_DIR/$rel_path"

    [ -f "$deploy_file" ] || continue
    [ -f "$repo_file" ]   || continue

    checked_count=$((checked_count + 1))

    if ! cmp -s "$repo_file" "$deploy_file"; then
        drift_count=$((drift_count + 1))
        printf '  %bDRIFT%b: %s\n' "${RED:-}" "${NC:-}" "$rel_path"
        if $verbose; then
            diff -u "$deploy_file" "$repo_file" | sed 's/^/    /'
        fi
    fi
done < <(cd "$REPO_DIR" && git ls-files)

echo ""
log_info "Checked: $checked_count managed files (ignored $unmanaged_count tracked but not deployed)"

if [ $drift_count -eq 0 ]; then
    log_success "No drift detected — repo and deploy-dir agree"
    exit 0
fi

log_warning "Drift detected in $drift_count file(s)"
echo "  Run './setup-linux.sh' to refresh the deploy-dir from the repo"
echo "  (or use --verbose to see the diff)"
exit 1
