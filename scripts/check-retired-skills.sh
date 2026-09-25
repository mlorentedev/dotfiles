#!/usr/bin/env bash
# check-retired-skills.sh SKILL... — exit 1 when a harness surface in $HOME still
# carries one of the named skills; exit 0 when none does.
#
# Two kinds of surface, both read from harness/manifest.json rather than listed
# here, so a harness added to the manifest is covered without editing this file:
#
#   - skills.deploy: a copy this pipeline rendered. It is ours only when it
#     carries the vault source it came from (generated_from: in frontmatter, or
#     the from: comment of a flat prompt). Other tools install skills into the
#     same directories, and a foreign skill that happens to share a name is not
#     ours to report.
#   - agents.presence, skills.catalog and doctrine.deploy: an instruction file
#     that names the skill as a catalog entry or in a forced roster.
#
# SKILL-001 retired eight skills and its independent review found the first two
# forms of this check blind: one read names only, and one hand-listed three of
# the six instruction files, missing opencode's and pi's.
set -euo pipefail

[ "$#" -gt 0 ] || { echo "usage: $0 SKILL..." >&2; exit 2; }
repo="$(cd "$(dirname "$0")/.." && pwd)"
manifest="$repo/harness/manifest.json"
found=0

while IFS=$'\t' read -r render dir; do
    for s in "$@"; do
        case "$render" in
            skill) f="$HOME/$dir/$s/SKILL.md" ;;
            command | prompt) f="$HOME/$dir/$s.md" ;;
            *) echo "unknown render type '$render' for $dir in $manifest" >&2; exit 2 ;;
        esac
        if grep -qsE "(^generated_from: |; from: )00_meta/skills/$s/SKILL\.md" "$f"; then
            echo "rendered copy: $f"
            found=1
        fi
    done
done < <(jq -r '.skills.deploy[] | [.render, .dir] | @tsv' "$manifest")

names="$(IFS='|'; echo "$*")"
while IFS= read -r rel; do
    f="$HOME/$rel"
    [ -f "$f" ] || continue
    if grep -nE "^- \*\*($names)\*\*|MUST consume: \[.*\b($names)\b" "$f" | sed "s|^|named in $f:|"; then
        found=1
    fi
done < <(jq -r '[.agents.presence[].file, .skills.catalog.file, .doctrine.deploy[].file] | unique[]' "$manifest")

exit "$found"
