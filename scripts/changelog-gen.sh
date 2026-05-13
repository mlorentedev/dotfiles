#!/bin/bash

# changelog-gen.sh: regenerate CHANGELOG.md from conventional-commit history.
#
# Reads `git log` (no merges), groups commits by Conventional Commit type
# (feat, fix, refactor, …), and writes a Markdown changelog grouped by type
# in reverse-chronological order. Deterministic: running it twice on the
# same history produces the same file (idempotent).
#
# Usage: ./scripts/changelog-gen.sh [--output FILE] [--since REV] [--check]
# Exit:  0 success, 1 if --check and CHANGELOG would change, 2 on error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
# Resolve repo: env override > git toplevel from cwd > script's parent dir.
if [ -n "${DOTFILES_REPO_DIR:-}" ]; then
    REPO_DIR="$DOTFILES_REPO_DIR"
elif REPO_DIR="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    :
else
    REPO_DIR="$(dirname "$SCRIPT_DIR")"
fi
OUTPUT="$REPO_DIR/CHANGELOG.md"
SINCE=""
CHECK_MODE=false

while [ $# -gt 0 ]; do
    case "$1" in
        --output) OUTPUT="$2"; shift 2 ;;
        --since)  SINCE="$2"; shift 2 ;;
        --check)  CHECK_MODE=true; shift ;;
        --help|-h)
            cat <<'EOF'
Usage: changelog-gen.sh [--output FILE] [--since REV] [--check]

Regenerate CHANGELOG.md from conventional commits.

Options:
  --output FILE   Write to FILE instead of CHANGELOG.md at repo root
  --since REV     Only include commits after REV (e.g. --since v1.0.0)
  --check         Don't write; exit 1 if the file would change (CI use)

The script groups commits by Conventional Commit type:
  feat, fix, perf, refactor, docs, test, ci, build, chore.
Anything else lands under "Other".
EOF
            exit 0
            ;;
        *) echo "Unknown argument: $1" >&2; exit 2 ;;
    esac
done

cd "$REPO_DIR"
[ -d .git ] || { echo "Error: not a git repo: $REPO_DIR" >&2; exit 2; }

range="HEAD"
[ -n "$SINCE" ] && range="${SINCE}..HEAD"

# Type → heading (order matters; this is the order in the output).
type_headings=(
    "feat:Features"
    "fix:Bug Fixes"
    "perf:Performance"
    "refactor:Refactoring"
    "docs:Documentation"
    "test:Tests"
    "ci:CI"
    "build:Build"
    "chore:Chores"
)

declare -A buckets
for entry in "${type_headings[@]}"; do
    buckets["${entry%%:*}"]=""
done
buckets[other]=""

# Read git log: <ISO-date>|<short-hash>|<subject>
while IFS='|' read -r date hash subject; do
    [ -z "$subject" ] && continue
    line="- ${date}: ${subject} (${hash})"$'\n'

    if [[ "$subject" =~ ^([a-z]+)(\([^\)]+\))?(!)?:\ .+$ ]]; then
        type="${BASH_REMATCH[1]}"
        if [ -n "${buckets[$type]+set}" ]; then
            buckets[$type]+="$line"
        else
            buckets[other]+="$line"
        fi
    else
        buckets[other]+="$line"
    fi
done < <(git log "$range" --no-merges --pretty=tformat:'%cs|%h|%s')

# Render to a temp buffer first so --check can compare without writing.
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

{
    echo "# Changelog"
    echo ""
    echo "Generated from conventional commits via \`scripts/changelog-gen.sh\`. Do not edit by hand."
    echo ""

    for entry in "${type_headings[@]}"; do
        type="${entry%%:*}"
        heading="${entry#*:}"
        if [ -n "${buckets[$type]}" ]; then
            echo "## $heading"
            echo ""
            printf '%s' "${buckets[$type]}"
            echo ""
        fi
    done

    if [ -n "${buckets[other]}" ]; then
        echo "## Other"
        echo ""
        printf '%s' "${buckets[other]}"
        echo ""
    fi
} > "$tmp"

if $CHECK_MODE; then
    if [ ! -f "$OUTPUT" ] || ! cmp -s "$tmp" "$OUTPUT"; then
        echo "CHANGELOG out of date — run scripts/changelog-gen.sh" >&2
        exit 1
    fi
    echo "CHANGELOG up to date"
    exit 0
fi

mv "$tmp" "$OUTPUT"
trap - EXIT
echo "CHANGELOG written to $OUTPUT"
