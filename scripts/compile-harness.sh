#!/usr/bin/env bash
# compile-harness.sh — agent-artifact deploy engine (ENGINE-001 / HARNESS-001 #162).
#
# Compiles enforced vault patterns into a marker-delimited "Overrides of Harness
# Defaults" block inside the deployed agent instruction files. The SSOT is the
# vault (00_meta/patterns); the generated output is COMMITTED so CI and machines
# without the vault can still verify it (ADR-013, builds on ADR-012).
#
# Modes:
#   --refresh   vault section  -> harness/enforced/<id>.md (source-of-record)
#                              -> injected, marker-delimited, into each target.
#               Requires the vault. Asserts per-file line caps. Run by setup.
#   --check     offline: render from harness/enforced/<id>.md and diff against
#               each target's region. NO vault access. Used by CI / healthcheck.
#   --help
#
# Spec: specs/ENGINE-001-deploy-engine-core/proposal.md
set -euo pipefail

BEGIN_PREFIX='<!-- BEGIN HARNESS GENERATED'
END_MARKER='<!-- END HARNESS GENERATED -->'

usage() {
    cat <<EOF
Usage: compile-harness.sh (--refresh | --check | --help)

  --refresh   Extract enforced sections from the vault, write source-of-record
              files under harness/enforced/, and inject them into each target's
              managed region. Requires the vault. Asserts line caps.
  --check     Offline drift check: render from harness/enforced/ and diff each
              target's managed region. No vault needed (CI / healthcheck).
  --help      This help.
EOF
}

# --- locate repo + inputs ---
if ! REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    printf '[ERROR] not in a git repo\n' >&2
    exit 1
fi
MANIFEST="$REPO_ROOT/harness/manifest.json"
RECORD_DIR="$REPO_ROOT/harness/enforced"
VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"

require_tools() {
    command -v jq >/dev/null 2>&1 || { printf '[ERROR] jq is required\n' >&2; exit 2; }
}

# --- helpers ---

# Extract a markdown section body by GitHub-style heading slug.
# Args: <pattern_file> <slug>. Prints the section body (heading excluded).
extract_section() {
    local file="$1" want="$2" out
    out="$(awk -v want="$want" '
        function slug(s){ s=tolower(s); gsub(/[^a-z0-9 -]/,"",s); gsub(/ +/,"-",s); gsub(/-+/,"-",s); return s }
        /^#{1,6} / {
            if (cap) { exit }
            hdr=$0; sub(/^#{1,6} +/,"",hdr)
            if (slug(hdr)==want) { cap=1; next }
        }
        cap { print }
    ' "$file")"
    if [[ -z "$out" ]]; then
        printf '[ERROR] section "%s" not found (or empty) in %s\n' "$want" "$file" >&2
        return 1
    fi
    printf '%s\n' "$out"
}

# Validate a target has exactly one well-ordered BEGIN/END marker pair.
validate_markers() {
    local file="$1" b e bl el
    b="$(grep -c "^$BEGIN_PREFIX" "$file" 2>/dev/null || true)"
    e="$(grep -cF "$END_MARKER" "$file" 2>/dev/null || true)"
    if [[ "$b" != "1" || "$e" != "1" ]]; then
        printf '[ERROR] %s: need exactly 1 BEGIN + 1 END HARNESS marker (found %s/%s)\n' "$file" "$b" "$e" >&2
        return 1
    fi
    bl="$(grep -n "^$BEGIN_PREFIX" "$file" | head -1 | cut -d: -f1)"
    el="$(grep -nF "$END_MARKER" "$file" | head -1 | cut -d: -f1)"
    if [[ "$bl" -ge "$el" ]]; then
        printf '[ERROR] %s: END marker precedes BEGIN marker\n' "$file" >&2
        return 1
    fi
}

# Print the content strictly between the markers of a target file.
region_content() {
    awk -v b="$BEGIN_PREFIX" -v e="$END_MARKER" '
        index($0,b)==1 {inreg=1; next}
        $0==e {inreg=0; next}
        inreg {print}
    ' "$1"
}

# Render the expected region for a target: concat of record files, in order.
# Args: id... . Prints to stdout. Fails if a record is missing.
render_region() {
    local id
    for id in "$@"; do
        if [[ ! -f "$RECORD_DIR/$id.md" ]]; then
            printf '[ERROR] missing source-of-record: harness/enforced/%s.md (run --refresh)\n' "$id" >&2
            return 1
        fi
        cat "$RECORD_DIR/$id.md"
    done
}

# Per-file deployed line cap (0 = no cap).
cap_for() {
    case "$1" in
        ai/claude/CLAUDE.md) printf '100' ;;
        ai/agy/AGY.md)       printf '50'  ;;
        *)                   printf '0'   ;;
    esac
}

# Replace a target's managed region with new content + a fresh BEGIN marker.
# Args: <file> <begin_marker> <content_file>
replace_region() {
    local file="$1" begin_marker="$2" content_file="$3" tmp rc=0
    tmp="$(mktemp)"
    if awk -v beginm="$begin_marker" -v endm="$END_MARKER" -v bp="$BEGIN_PREFIX" -v cf="$content_file" '
        index($0,bp)==1 {
            print beginm
            while ((getline l < cf) > 0) print l
            close(cf)
            skip=1; next
        }
        $0==endm { if (skip){ print; skip=0; next } }
        skip { next }
        { print }
    ' "$file" > "$tmp"; then rc=0; else rc=$?; fi
    if [[ $rc -ne 0 ]]; then rm -f "$tmp"; return $rc; fi
    mv "$tmp" "$file"
}

sha_of() { sha256sum "$1" | cut -c1-16; }

target_inject() { jq -r --arg f "$1" '.targets[] | select(.file==$f) | .inject[]' "$MANIFEST"; }

# --- modes ---

do_refresh() {
    require_tools
    local vsub pat_dir
    vsub="$(jq -r '.vault_subpath' "$MANIFEST")"
    pat_dir="$VAULT_PATH/$vsub"
    if [[ ! -d "$pat_dir" ]]; then
        cat >&2 <<EOF
[FATAL] --refresh needs the vault. Patterns dir not found: $pat_dir
  -> clone the vault to \$VAULT_PATH (default ~/Projects/knowledge), or set VAULT_PATH.
  ( --check works offline and needs no vault. )
EOF
        exit 2
    fi

    mkdir -p "$RECORD_DIR"

    # 1. vault section -> source-of-record per enforced rule
    local id source pat slug body
    while IFS=$'\t' read -r id source; do
        pat="${source%%#*}"; slug="${source#*#}"
        body="$(extract_section "$pat_dir/$pat" "$slug")"
        printf '%s\n' "$body" > "$RECORD_DIR/$id.md"
        printf '[refresh] record: harness/enforced/%s.md  <- %s#%s\n' "$id" "$pat" "$slug"
    done < <(jq -r '.enforced[] | "\(.id)\t\(.source)"' "$MANIFEST")

    # 2. inject into each target
    local file ids tmpc sha begin cap lines
    while IFS= read -r file; do
        mapfile -t ids < <(target_inject "$file")
        validate_markers "$REPO_ROOT/$file"
        tmpc="$(mktemp)"
        render_region "${ids[@]}" > "$tmpc"
        sha="$(sha_of "$tmpc")"
        begin="$BEGIN_PREFIX (sha256:$sha) — SSOT: vault $vsub; edit there + re-run setup, do NOT edit between markers -->"
        replace_region "$REPO_ROOT/$file" "$begin" "$tmpc"
        rm -f "$tmpc"
        cap="$(cap_for "$file")"
        if [[ "$cap" != "0" ]]; then
            lines="$(wc -l < "$REPO_ROOT/$file")"
            if [[ "$lines" -gt "$cap" ]]; then
                printf '[ERROR] %s is %s lines after injection (cap %s)\n' "$file" "$lines" "$cap" >&2
                exit 1
            fi
        fi
        printf '[refresh] injected -> %s (%s)\n' "$file" "$(wc -l < "$REPO_ROOT/$file" | tr -d ' ')L"
    done < <(jq -r '.targets[].file' "$MANIFEST")

    printf '[refresh] OK\n'
}

do_check() {
    require_tools
    local file ids drift=0 expected actual
    while IFS= read -r file; do
        if ! validate_markers "$REPO_ROOT/$file"; then drift=1; continue; fi
        mapfile -t ids < <(target_inject "$file")
        expected="$(mktemp)"; actual="$(mktemp)"
        if ! render_region "${ids[@]}" > "$expected"; then drift=1; rm -f "$expected" "$actual"; continue; fi
        region_content "$REPO_ROOT/$file" > "$actual"
        if ! diff -u "$expected" "$actual" >/dev/null 2>&1; then
            printf '[DRIFT] %s: managed region differs from harness/enforced/ (run --refresh)\n' "$file" >&2
            diff -u "$actual" "$expected" | sed 's/^/    /' >&2 || true
            drift=1
        else
            printf '[check] OK -> %s\n' "$file"
        fi
        rm -f "$expected" "$actual"
    done < <(jq -r '.targets[].file' "$MANIFEST")

    if [[ "$drift" -ne 0 ]]; then
        printf '[check] FAIL: harness drift detected\n' >&2
        exit 1
    fi
    printf '[check] OK: no harness drift\n'
}

# --- dispatch ---
case "${1:-}" in
    --refresh) do_refresh ;;
    --check)   do_check ;;
    -h|--help) usage ;;
    *)         usage >&2; exit 2 ;;
esac
