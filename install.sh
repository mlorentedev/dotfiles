#!/usr/bin/env bash
# install.sh — One-liner bootstrap for a factory-fresh machine.
#
# Usage (fresh machine):
#   curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash
#
# Usage (already cloned):
#   ./install.sh
#
# Environment variables:
#   DOTFILES_DIR   — clone destination (default: ~/Projects/dotfiles)
#   DOTFILES_REPO  — upstream URL     (default: https://github.com/mlorentedev/dotfiles.git)
#   DOTFILES_SKIP_SETUP — set to 1 to skip setup-linux.sh (for testing the clone path)
#
# Security note: always inspect before piping.
#   curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | less

set -euo pipefail

DOTFILES_DIR="${DOTFILES_DIR:-$HOME/Projects/dotfiles}"
DOTFILES_REPO="${DOTFILES_REPO:-https://github.com/mlorentedev/dotfiles.git}"

# 1. Pre-flight: git must exist
if ! command -v git >/dev/null 2>&1; then
    echo "ERROR: git is not installed. Install git first, then re-run." >&2
    exit 1
fi

# 2. Clone or update
if [ -d "$DOTFILES_DIR" ]; then
    if [ -d "$DOTFILES_DIR/.git" ]; then
        echo "Updating existing $DOTFILES_DIR..."
        git -C "$DOTFILES_DIR" pull --ff-only
    else
        echo "ERROR: $DOTFILES_DIR exists but is not a git repository." >&2
        echo "       Remove it or set DOTFILES_DIR to a different path." >&2
        exit 1
    fi
else
    echo "Cloning $DOTFILES_REPO → $DOTFILES_DIR..."
    mkdir -p "$(dirname "$DOTFILES_DIR")"
    git clone "$DOTFILES_REPO" "$DOTFILES_DIR"
fi

# 3. Delegate to setup (unless skipped for testing)
if [ "${DOTFILES_SKIP_SETUP:-0}" = "1" ]; then
    echo "DOTFILES_SKIP_SETUP=1 — skipping setup-linux.sh"
    exit 0
fi

cd "$DOTFILES_DIR"
exec ./setup-linux.sh "$@"
