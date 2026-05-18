#!/bin/bash
# scripts/skills-to-opencode.sh
#
# Port Claude Code skills (ai/skills/<name>/SKILL.md) to OpenCode commands
# (ai/opencode/commands/<name>.md). Source of truth stays the SKILL.md files;
# OpenCode commands are a deterministic derivative.
#
# Differences between the two formats:
#   - SKILL.md frontmatter has  `name:` + `description:` -- the filename plus
#     `name:` both encode the skill identifier (redundant on purpose).
#   - OpenCode commands take only `description:` -- the *filename* alone is
#     the command identifier, so `audit.md` becomes `/audit` in the TUI.
#   - Body markdown is copied verbatim.
#
# Skip-list: skills that depend on Claude Code-only primitives (TaskCreate,
# AskUserQuestion, subagent_type) or on Anthropic-only MCPs (claude-mem)
# would not work in OpenCode and are excluded.
#
# Usage:
#   scripts/skills-to-opencode.sh              # generate / sync
#   scripts/skills-to-opencode.sh --check      # CI gate: exit 1 if drift
#
# Idempotent: re-running with identical inputs is a no-op.

set -euo pipefail

SKIP_SKILLS=(
    "creating-skills"               # meta-skill for Claude skill anatomy
    "dispatching-parallel-agents"   # uses Task subagent_type (Claude-only)
    "crystallize"                   # depends on claude-mem MCP (Anthropic-only)
    "insights"                      # depends on claude-mem MCP
    "executing-plans"               # depends on TaskCreate primitive
)

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SKILLS_DIR="$REPO_DIR/ai/skills"
COMMANDS_DIR="$REPO_DIR/ai/opencode/commands"

CHECK_ONLY=0
case "${1:-}" in
    --check) CHECK_ONLY=1 ;;
    -h|--help)
        sed -n '/^# Usage:/,/^# Idempotent:/p' "$0" | sed 's/^# \{0,1\}//'
        exit 0
        ;;
    "") ;;
    *)
        echo "Unknown argument: $1 (use -h)" >&2
        exit 2
        ;;
esac

is_skipped() {
    local name=$1
    for skip in "${SKIP_SKILLS[@]}"; do
        [ "$skip" = "$name" ] && return 0
    done
    return 1
}

# transform_skill SRC DST
# Reads SKILL.md frontmatter, drops `name:` lines, keeps `description:` and
# any other yaml fields, then appends the body unchanged.
transform_skill() {
    local src=$1
    local dst=$2
    awk '
        BEGIN { state = "before"; }
        # First --- opens frontmatter.
        state == "before" && /^---$/ {
            state = "front"
            print
            next
        }
        # Second --- closes frontmatter.
        state == "front" && /^---$/ {
            state = "body"
            print
            next
        }
        # Inside frontmatter: skip `name:` lines (OpenCode does not consume it).
        state == "front" {
            if ($0 ~ /^name:[[:space:]]/) next
            print
            next
        }
        # Body: copy verbatim.
        state == "body" { print }
    ' "$src" > "$dst"
}

mkdir -p "$COMMANDS_DIR"

EXIT_CODE=0
GENERATED=0
SKIPPED=0
DRIFT=0
ORPHAN=0

# Phase 1: process each SKILL.md.
for skill_dir in "$SKILLS_DIR"/*/; do
    name=$(basename "$skill_dir")
    src="$skill_dir/SKILL.md"
    [ -f "$src" ] || continue

    if is_skipped "$name"; then
        SKIPPED=$((SKIPPED + 1))
        # If a stale command file exists for a now-skipped skill, flag it.
        if [ -f "$COMMANDS_DIR/$name.md" ]; then
            if [ "$CHECK_ONLY" = "1" ]; then
                echo "DRIFT: $COMMANDS_DIR/$name.md exists but $name is on the skip-list"
                DRIFT=$((DRIFT + 1))
                EXIT_CODE=1
            else
                rm -f "$COMMANDS_DIR/$name.md"
                echo "Removed stale $name.md (skill is on skip-list)"
            fi
        fi
        continue
    fi

    dst="$COMMANDS_DIR/$name.md"

    if [ "$CHECK_ONLY" = "1" ]; then
        tmp=$(mktemp)
        transform_skill "$src" "$tmp"
        if [ ! -f "$dst" ] || ! cmp -s "$tmp" "$dst"; then
            echo "DRIFT: $dst out of sync with $src"
            DRIFT=$((DRIFT + 1))
            EXIT_CODE=1
        fi
        rm -f "$tmp"
    else
        transform_skill "$src" "$dst"
        GENERATED=$((GENERATED + 1))
    fi
done

# Phase 2: detect orphan command files (no matching SKILL.md, not on skip-list).
if [ -d "$COMMANDS_DIR" ]; then
    for cmd_file in "$COMMANDS_DIR"/*.md; do
        [ -f "$cmd_file" ] || continue
        cmd_name=$(basename "$cmd_file" .md)
        if [ ! -d "$SKILLS_DIR/$cmd_name" ]; then
            if [ "$CHECK_ONLY" = "1" ]; then
                echo "DRIFT: $cmd_file has no matching skill at $SKILLS_DIR/$cmd_name/"
                ORPHAN=$((ORPHAN + 1))
                EXIT_CODE=1
            else
                rm -f "$cmd_file"
                echo "Removed orphan $cmd_file"
            fi
        fi
    done
fi

if [ "$CHECK_ONLY" = "1" ]; then
    if [ "$DRIFT" -gt 0 ] || [ "$ORPHAN" -gt 0 ]; then
        echo "FAIL: $DRIFT drift + $ORPHAN orphan(s). Run: scripts/skills-to-opencode.sh"
    else
        echo "OK: ai/opencode/commands/ in sync with ai/skills/ (skipped $SKIPPED Claude-only)"
    fi
else
    echo "Generated $GENERATED command(s) in $COMMANDS_DIR (skipped $SKIPPED Claude-only)"
fi

exit $EXIT_CODE
