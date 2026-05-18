#!/bin/bash
# init-repo-github-defaults.sh
# Purpose: Apply opinionated GitHub repo-level defaults to a target repo
#          via `gh api`. Idempotent (detect-current-state, patch only when
#          a setting diverges).
#
# Why: avoids the slow drift of stale merged-PR head branches on origin.
#      Empirically captured 2026-05-18 after a marathon session left 8
#      orphan branches because `gh pr merge` was used without
#      `--delete-branch` and the repo did not have
#      `delete_branch_on_merge` enabled.
#
# Documented as a cross-project pattern in vault:
#   00_meta/patterns/github-branch-hygiene.md
#
# Usage:
#   ./init-repo-github-defaults.sh [--repo <owner/name>] [--dry-run]
#
# If --repo is not given, derives owner/name from the current repo's
# `origin` remote URL.

set -euo pipefail

usage() {
    cat <<'EOF'
Usage: init-repo-github-defaults.sh [--repo <owner/name>] [--dry-run]

  --repo <owner/name>   target repo (default: derived from `origin` remote)
  --dry-run             show diffs without applying
  -h, --help            show this help

Settings applied:
  delete_branch_on_merge = true   GitHub auto-deletes the head branch when
                                  a PR is merged (squash, rebase, or merge
                                  commit). Eliminates the orphan branch
                                  drift this script was written to prevent.

Requires: gh CLI authenticated (`gh auth status`).
EOF
}

TARGET_REPO=""
DRY_RUN=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --repo) TARGET_REPO="$2"; shift 2 ;;
        --dry-run) DRY_RUN=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) printf '[ERROR] Unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

if ! command -v gh >/dev/null 2>&1; then
    printf '[ERROR] gh CLI not found. Install: https://cli.github.com/\n' >&2
    exit 3
fi

# --- Resolve repo ---
if [[ -z "$TARGET_REPO" ]]; then
    if ! ORIGIN_URL="$(git remote get-url origin 2>/dev/null)"; then
        printf '[ERROR] Not in a git repo or no `origin` remote. Use --repo <owner/name>.\n' >&2
        exit 4
    fi
    # Handle both SSH (git@github.com:owner/name.git) and HTTPS forms
    TARGET_REPO=$(printf '%s\n' "$ORIGIN_URL" \
        | sed -E 's#^git@github\.com:##; s#^https?://github\.com/##; s#\.git$##')
    if ! [[ "$TARGET_REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
        printf '[ERROR] Could not derive owner/name from origin (%s). Use --repo.\n' "$ORIGIN_URL" >&2
        exit 5
    fi
fi

printf '[INFO] Target repo: %s\n' "$TARGET_REPO"

# --- Current state ---
CURRENT_STATE=$(gh api "/repos/$TARGET_REPO" --jq '{delete_branch_on_merge}' 2>/dev/null || echo '{}')
CURRENT_DELETE_ON_MERGE=$(printf '%s\n' "$CURRENT_STATE" | grep -oE 'true|false' | head -1 || echo 'unknown')

printf '[INFO] Current delete_branch_on_merge: %s\n' "$CURRENT_DELETE_ON_MERGE"

if [[ "$CURRENT_DELETE_ON_MERGE" = "true" ]]; then
    printf '[OK] Already enabled, nothing to do.\n'
    exit 0
fi

# --- Apply ---
if [[ "$DRY_RUN" -eq 1 ]]; then
    printf '[DRY-RUN] Would PATCH /repos/%s with delete_branch_on_merge=true\n' "$TARGET_REPO"
    exit 0
fi

if gh api -X PATCH "/repos/$TARGET_REPO" -f delete_branch_on_merge=true --jq '{delete_branch_on_merge}' 2>&1; then
    printf '[OK] delete_branch_on_merge enabled on %s\n' "$TARGET_REPO"
else
    printf '[ERROR] PATCH failed (insufficient permissions, or repo archived?)\n' >&2
    exit 6
fi
