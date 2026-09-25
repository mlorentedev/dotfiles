#!/usr/bin/env bash
# check-retired-skills.sh SKILL... — exit 1 when a harness surface in $HOME still
# carries one of the named skills, 0 when none does, and 2 when it could not
# look.
#
# The surfaces are read from harness/manifest.json rather than listed here, so a
# harness added to the manifest is covered without editing this file:
#
#   - skills.deploy: a copy this pipeline rendered. It is ours only when it
#     carries the vault source it came from (generated_from: in frontmatter, or
#     the from: comment of a flat prompt). Other tools install skills into the
#     same directories, and a foreign skill that happens to share a name is not
#     ours to report.
#   - agents.presence, skills.catalog and doctrine.deploy: an instruction file
#     that names the skill as a catalog entry or in a forced roster.
#   - agents.deploy (render agent-md): a rendered agent definition, which carries
#     its roster too.
#
# It fails closed. A check that looked at nothing must not read as clean, so a
# missing jq, an unreadable manifest, or a manifest that declares no target to
# look at is exit 2, never 0.
#
# SKILL-001 retired eight skills, and its independent review found three earlier
# forms of this check blind: one read names only, one hand-listed three of the
# six instruction files, and one passed when it had nothing to read.
set -euo pipefail

[ "$#" -gt 0 ] || { echo "usage: $0 SKILL..." >&2; exit 2; }
repo="$(cd "$(dirname "$0")/.." && pwd)"
manifest="$repo/harness/manifest.json"

fail() { echo "check-retired-skills: $*" >&2; exit 2; }
type -P jq >/dev/null 2>&1 || fail "jq is required"
[ -r "$manifest" ] || fail "cannot read $manifest"

# list prints one jq query's answer with any CR removed: the winget build of jq
# on Windows ends every line with CRLF, which would leave a \r in each path (the
# same shadow compile-harness.sh keeps).
list() {
    local out
    out="$(jq -r "$1" "$manifest")" || fail "cannot read $manifest as JSON"
    printf '%s\n' "$out" | tr -d '\r'
}

deploy="$(list '.skills.deploy[]? | [.render, .dir] | @tsv')"
files="$(list '[.agents.presence[]?.file, .skills.catalog.file?, .doctrine.deploy[]?.file] | map(select(. != null)) | unique[]')"
agent_dirs="$(list '.agents.deploy[]? | select(.render == "agent-md") | .dir')"
[ -n "$deploy" ] || fail "the manifest declares no skills.deploy target, so nothing would be checked"
[ -n "$files" ] || fail "the manifest declares no instruction file, so nothing would be checked"
[ -n "$agent_dirs" ] || fail "the manifest renders no agent definition (agents.deploy, agent-md), so none would be checked"

found=0
while IFS=$'\t' read -r render dir; do
    for s in "$@"; do
        case "$render" in
            skill) f="$HOME/$dir/$s/SKILL.md" ;;
            command | prompt) f="$HOME/$dir/$s.md" ;;
            *) fail "unknown render type '$render' for $dir in $manifest" ;;
        esac
        if grep -qsE "(^generated_from: |; from: )00_meta/skills/$s/SKILL\.md" "$f"; then
            echo "rendered copy: $f"
            found=1
        fi
    done
done <<<"$deploy"

names="$(IFS='|'; echo "$*")"
check_file() {
    [ -f "$1" ] || return 0
    if grep -nE "^- \*\*($names)\*\*|MUST consume: \[.*\b($names)\b" "$1" | sed "s|^|named in $1:|"; then
        found=1
    fi
}
while IFS= read -r rel; do
    check_file "$HOME/$rel"
done <<<"$files"
while IFS= read -r dir; do
    [ -n "$dir" ] || continue
    for f in "$HOME/$dir"/*.md; do
        check_file "$f"
    done
done <<<"$agent_dirs"

exit "$found"
