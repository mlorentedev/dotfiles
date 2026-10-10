#!/usr/bin/env bats
# GUARD: no production caller references claude-mem (MEM-002, ADR-016 Q2).
# claude-mem (the @thedotmack conversation-memory plugin) is retired. This guard
# pins that no production path re-introduces a reference to it. It scans the
# wiring surfaces — scripts/, setup-linux.sh, setup-windows.ps1, cli/,
# session-start-config.json, and .github/workflows/ — for a case-insensitive
# `claude.?mem` (matches claude-mem, claude_mem, claudemem, CLAUDE_MEM...).
#
# EXCLUSIONS (deliberate, not production wiring):
#   - docs/ and specs/ (historical records, ADRs, troubleshooting archive).
#   - *.md anywhere (prose).
#   - this guard file itself.
#   - a block marked `# MEM-002:` up to the next blank line, which may name the
#     plugin to remove it (setup-windows.ps1's retired-script removal list). It
#     is stripped before scanning, so an UNRELATED claude-mem line is still
#     caught. The marketplace cleanup itself left the setups for `dotf deploy`
#     (ai/claude/plugins.json `retired_marketplaces`, #1431).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

# Print $1 with the contiguous MEM-002 cleanup block(s) removed. A block starts
# at a line matching `# MEM-002:` and ends at the first following blank line.
_strip_mem002_block() {
    awk '
        /^[[:space:]]*#+[[:space:]]*MEM-002:/ { inblock = 1 }
        inblock && /^[[:space:]]*$/           { inblock = 0; next }
        inblock                               { next }
        { print }
    ' "$1"
}

# Emit every offending "path:line:text" hit across the production surfaces,
# with the allowed MEM-002 cleanup blocks already stripped. Empty == clean.
collect_hits() {
    local files=()
    # Recursive file lists from the scanned directories (skip .md + this guard).
    local d
    for d in "$REPO/scripts" "$REPO/cli" "$REPO/.github/workflows"; do
        [ -d "$d" ] || continue
        while IFS= read -r f; do files+=("$f"); done < <(
            find "$d" -type f ! -name '*.md' ! -name 'guard-no-claude-mem.bats'
        )
    done
    # Individual files.
    for f in "$REPO/setup-linux.sh" "$REPO/setup-windows.ps1" "$REPO/session-start-config.json"; do
        [ -f "$f" ] && files+=("$f")
    done

    local file rel
    for file in "${files[@]}"; do
        rel="${file#"$REPO"/}"
        # Strip the MEM-002 block, keep line numbers via grep -n on the stripped
        # stream, then re-prefix the path so the report is "path:line:text".
        _strip_mem002_block "$file" \
            | grep -nIE 'claude.?mem' 2>/dev/null \
            | sed "s#^#${rel}:#"
    done
}

@test "no production path references claude-mem (MEM-002 guard)" {
    run collect_hits
    if [ -n "$output" ]; then
        echo "Forbidden claude-mem reference(s) in production paths:"
        echo "$output"
        echo "---"
        echo "claude-mem is retired (MEM-002). Remove the reference, or — if this is"
        echo "the one-cycle uninstall/cleanup — keep it inside the MEM-002 block."
        return 1
    fi
    [ -z "$output" ]
}

@test "the retirement is still declared, so this guard does not pass because it was deleted" {
    # The setup cleanup blocks are gone (#1431): `dotf deploy` removes the
    # marketplace from ai/claude/plugins.json `retired_marketplaces` and checks
    # the CLI's own registry afterwards. Dropping the entry would let the
    # marketplace re-clone with autoUpdate on any box that still registers it.
    jq -e '.retired_marketplaces | index("thedotmack")' "$REPO/ai/claude/plugins.json"
}
