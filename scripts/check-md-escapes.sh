#!/bin/bash

# check-md-escapes.sh: detect Hive vault_patch corruption in markdown files.
#
# Hive's MCP `vault_patch` has been observed writing the literal 2-character
# sequence `\n` (backslash followed by 'n') into markdown content when the
# replacement string was meant to include a real newline. The corruption is
# invisible in rendered markdown but breaks line-based parsers (dotf spec
# frontmatter gate, vault-health.sh frontmatter scans, obs-cli graph queries).
#
# This script scans for the canonical signature: a literal `\n` followed by a
# recognised line-start glyph (`-`, `#`, `>`, `*`). Single-pass grep, no
# Obsidian dependency, safe to run from CI.
#
# Usage:
#   check-md-escapes.sh <path>...
#     <path>  file or directory; directories scanned recursively for *.md.
#
# Exit:
#   0  clean
#   1  one or more files contain the corruption pattern
#   2  usage error / missing path

set -euo pipefail

if [ "$#" -eq 0 ]; then
    cat >&2 <<'EOF'
Usage: check-md-escapes.sh <path>...
  Scans markdown files for literal '\n' sequences immediately followed by a
  line-start glyph (- # > *). This signature catches the Hive vault_patch
  corruption class: see specs/archive/SDD-006-vault-integrity-check/.

  <path> may be a file or directory. Directories are scanned recursively for
  *.md, excluding common vendor/build paths (.obsidian, node_modules, .git).
EOF
    exit 2
fi

# The pattern that catches corruption: \n followed by one of -#>* (the
# common markdown line-start glyphs). Two backslashes in the grep -E ERE
# match a single literal backslash, then literal 'n', then one of the glyphs.
PATTERN='\\n[-#>*]'

corrupted_count=0
for target in "$@"; do
    if [ -f "$target" ]; then
        if grep -q -E "$PATTERN" "$target"; then
            printf '  CORRUPTED: %s\n' "$target"
            corrupted_count=$((corrupted_count + 1))
        fi
    elif [ -d "$target" ]; then
        while IFS= read -r -d '' f; do
            if grep -q -E "$PATTERN" "$f"; then
                printf '  CORRUPTED: %s\n' "$f"
                corrupted_count=$((corrupted_count + 1))
            fi
        done < <(find "$target" \
            -name '*.md' \
            -not -path '*/.obsidian/*' \
            -not -path '*/node_modules/*' \
            -not -path '*/.git/*' \
            -print0)
    else
        printf '  ERROR: not found or unreadable: %s\n' "$target" >&2
        exit 2
    fi
done

if [ "$corrupted_count" -gt 0 ]; then
    cat >&2 <<EOF

$corrupted_count file(s) contain literal '\\n' sequences in markdown content.
This is the Hive vault_patch corruption class — see AGENTS.md Standing Order #4
and the SDD-006 spec for the incident-to-guard pattern. Recovery: open the
file in an editor and replace literal '\\n' sequences with real newlines.
EOF
    exit 1
fi
exit 0
