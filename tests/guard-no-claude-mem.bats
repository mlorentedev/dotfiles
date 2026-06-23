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
#   - the MEM-002 one-cycle uninstall/cleanup block in setup-{linux,windows}:
#     it MUST name the plugin to remove it. The block is delimited by a
#     `# MEM-002:` marker and the next blank line; we strip it before scanning,
#     so an UNRELATED claude-mem line in a setup script is still caught.
#     Prune the block after rollout; this guard then tightens automatically.

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

@test "the MEM-002 cleanup block IS present in both setup scripts (anti-regression)" {
    # The guard must not pass merely because the cleanup was also deleted.
    grep -qF 'claude plugin uninstall claude-mem@thedotmack' "$REPO/setup-linux.sh"
    grep -qF 'claude plugin uninstall claude-mem@thedotmack' "$REPO/setup-windows.ps1"
}
