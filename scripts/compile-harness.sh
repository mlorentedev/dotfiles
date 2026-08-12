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
#               Skills (SDD-008): vault 00_meta/skills/<n> -> harness/skills/<n>
#               committed records (frontmatter validated). NO render here.
#               Requires the vault. Asserts per-file line caps. Run by setup.
#   --deploy    offline: render committed skill records to their per-agent $HOME
#               paths (de-symlinking first) and inject the copilot catalog into
#               the $HOME instructions file. NO vault access. Run by setup.
#   --check     offline: enforced regions diffed against harness/enforced/; skill
#               records validated to render cleanly. NO vault. CI / healthcheck.
#   --help
#
# Skills use option A (SDD-008): the committed record under harness/skills/ is the
# SSOT; agent-native outputs are NOT committed — they are rendered per-machine at
# deploy time. This sidesteps the symlink fragility of BUG-100 (deploy is a copy,
# never a symlink) while keeping CI able to verify drift offline (--check).
#
# Spec: specs/ENGINE-001-deploy-engine-core/proposal.md,
#       specs/SDD-008-skill-pipeline/proposal.md
set -euo pipefail

BEGIN_PREFIX='<!-- BEGIN HARNESS GENERATED'
END_MARKER='<!-- END HARNESS GENERATED -->'

# Distinct marker namespace for agent-presence injection (ADR-027). Kept separate
# from BEGIN_PREFIX/END_MARKER so an agent-presence region coexists with the
# patterns / skill-catalog region in the same always-loaded instructions file
# without either disturbing the other (validate_markers expects exactly one
# GENERATED pair; presence uses its own pair).
AGENT_BEGIN_PREFIX='<!-- BEGIN HARNESS AGENT-PRESENCE'
AGENT_END_MARKER='<!-- END HARNESS AGENT-PRESENCE -->'

usage() {
    cat <<EOF
Usage: compile-harness.sh (--refresh | --deploy | --check | --help)

  --refresh   Extract enforced sections from the vault into harness/enforced/
              and inject them into each target's managed region; copy vault
              skills into committed records under harness/skills/ (frontmatter
              validated). Requires the vault. Asserts line caps. No \$HOME write.
  --deploy    Offline: render committed skill records to their per-agent \$HOME
              paths (replacing any pre-existing symlink with a regular copy) and
              inject the copilot catalog into the \$HOME instructions file.
  --check     Offline drift check: enforced regions diffed against
              harness/enforced/; skill records validated to render cleanly.
              No vault needed (CI / healthcheck).
  --help      This help.
EOF
}

# --- locate repo + inputs ---
# Resolve the repo/deploy root from the script's own location: CWD-independent
# and deploy-model-independent. Correct in a git checkout, a linked worktree, and
# the non-git deploy copy (~/.dotfiles, ADR-012 copy-deploy) alike — and the copy
# is exactly where setup + healthcheck invoke --check / --deploy. (The previous
# `git rev-parse --show-toplevel` was CWD-dependent and errored on the non-git
# copy, which was the root cause of the section-12 drift false-fail.)
# HARNESS_REPO_ROOT overrides this — the test harness points the real script at a
# throwaway fixture tree, the same explicit-override idiom as VAULT_PATH below.
REPO_ROOT="${HARNESS_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)}"
MANIFEST="$REPO_ROOT/harness/manifest.json"
RECORD_DIR="$REPO_ROOT/harness/enforced"
VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"

require_tools() {
    type -P jq >/dev/null 2>&1 || { printf '[ERROR] jq is required\n' >&2; exit 2; }
    [ -f "$MANIFEST" ] || { printf '[ERROR] manifest not found: %s (run from the repo or a complete deploy)\n' "$MANIFEST" >&2; exit 2; }
}

# The winget Windows build of jq emits CRLF line terminators (even for a single
# value). MSYS command-substitution `$(jq …)` silently strips the trailing CRLF,
# but `read`/`mapfile`/`for` fed via `< <(jq …)` keep the bare `\r` in the last
# field — which broke slug/path matching on Windows (the "section not found"
# refresh failure). Shadow `jq` with a CR-stripping wrapper so every call yields
# LF output regardless of platform, while preserving jq's own exit status
# (PIPESTATUS[0]) so `jq -e` truthiness checks keep working. `require_tools` uses
# `type -P` above to verify the real binary, not this function. (Superseded once
# CLI-026 ports the engine to Go.)
jq() { command jq "$@" | tr -d '\r'; return "${PIPESTATUS[0]}"; }

# --- helpers ---

# Extract a markdown section body by GitHub-style heading slug.
# Args: <pattern_file> <slug>. Prints the section body (heading excluded).
# The section ends at the next heading of the SAME-OR-HIGHER level, so deeper
# sub-headings (e.g. a ### under a ## rule) stay inside the body instead of
# silently truncating it (the #156 regression class — ENGINE-002).
extract_section() {
    local file="$1" want="$2" out
    out="$(awk -v want="$want" '
        function slug(s){ s=tolower(s); gsub(/[^a-z0-9 -]/,"",s); gsub(/ +/,"-",s); gsub(/-+/,"-",s); return s }
        /^#{1,6} / {
            match($0, /^#+/); lvl=RLENGTH
            if (cap) {
                if (lvl <= caplvl) { exit }   # same/higher level == section boundary
                # deeper sub-heading: fall through, it is part of the body
            } else {
                hdr=$0; sub(/^#{1,6} +/,"",hdr)
                if (slug(hdr)==want) { cap=1; caplvl=lvl; next }
            }
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

# Inject `generated_*` provenance into a COMMITTED record in place (HARNESS-069).
# Option A's committed harness/{skills,agents}/<name>/{SKILL,AGENT}.md is a
# rendered artifact of the vault source, but until now carried nothing saying
# so — opened directly in the repo it read as hand-authored. Mirrors
# render_skill/render_agent's frontmatter injection, but rewrites in place
# instead of rendering to stdout, and the sha is of the VAULT SOURCE at refresh
# time (the field says "this is what I was refreshed from"), not of the record.
inject_record_provenance() {
    local file="$1" srcpath="$2" sha="$3" tmp
    tmp="$(mktemp)"
    awk -v gf="$srcpath" -v gs="$sha" '
        /^---[[:space:]]*$/ {
            fm++
            if (fm==1) { print; print "generated: true"; print "generated_from: " gf; print "generated_sha: " gs; next }
        }
        { print }
    ' "$file" > "$tmp"
    mv "$tmp" "$file"
}

# --- skills (kind: render) ---
# Skills are whole-file transforms: one vault 00_meta/skills/<name>/SKILL.md ->
# N agent-native outputs (committed in-repo; setup copies them to $HOME). Drift
# is checked offline against the committed source-of-record under harness/skills/.
# A leading HTML provenance comment would break the YAML frontmatter agents
# parse, so provenance is injected as `generated_*` frontmatter fields instead.

has_skills() { jq -e '.skills' "$MANIFEST" >/dev/null 2>&1; }
has_agents() { jq -e '.agents' "$MANIFEST" >/dev/null 2>&1; }

# Does this skill target this agent? Reads `targets:` from SKILL.md frontmatter.
# Absent `targets:` => all agents (default).
skill_targets_agent() {
    local skill_md="$1" agent="$2" line
    line="$(awk '/^---[[:space:]]*$/{n++; next} n==1 && /^targets:/{print; exit}' "$skill_md")"
    [[ -z "$line" ]] && return 0
    case "$line" in *"$agent"*) return 0 ;; *) return 1 ;; esac
}

# Render one skill record to an agent-native string on stdout.
# Args: <render_kind> <record_SKILL.md> <vault-relative-source-path>
#   skill   -> full SKILL.md + provenance frontmatter fields (claude, agy native)
#   command -> drop `name:` (opencode commands key off filename) + provenance
#   prompt  -> strip YAML frontmatter entirely, prepend provenance comment (agy
#              flat prompts in ~/.gemini/prompts/, mirrors setup's sed strip)
#
# The injected `generated_*` fields are deliberately dual-referent, not a
# single "where did this come from" answer (HARNESS-069): `generated_from` is
# always the vault path in `srcpath` — where a human edits, the SSOT — while
# `generated_sha` hashes `record` (the committed harness/skills/... file this
# call renders FROM), not the vault source. One field pair, two questions:
# "where do I fix this" and "is this deploy still fresh against the record it
# was built from". The committed record's OWN provenance (written by
# inject_record_provenance at --refresh) answers a related but different
# question the same way — generated_from = vault, generated_sha = the vault
# source's hash — which is why it must be stripped here rather than passed
# through: two blocks with the same field names but different sha referents,
# stacked in one file, would be actively misleading.
render_skill() {
    local kind="$1" record="$2" srcpath="$3" sha
    sha="$(sha_of "$record")"
    if [[ "$kind" == "prompt" ]]; then
        printf '<!-- generated: true; from: %s; sha256:%s; edit the vault source + re-run setup -->\n\n' "$srcpath" "$sha"
        awk '/^---[[:space:]]*$/{fm++; next} fm>=2{print} fm==0{print}' "$record"
        return
    fi
    awk -v kind="$kind" -v gf="$srcpath" -v gs="$sha" '
        /^---[[:space:]]*$/ {
            fm++
            if (fm==1) { print; print "generated: true"; print "generated_from: " gf; print "generated_sha: " gs; next }
        }
        fm==1 && kind=="command" && /^name:/ { next }   # opencode commands key off filename
        # The record (HARNESS-069) already carries its own generated_* fields,
        # describing its relationship to the vault. Strip them here so deploy
        # injects one fresh set, describing the deploy targets relationship to
        # the record, instead of stacking a second set on top.
        fm==1 && /^generated(_from|_sha)?:/ { next }
        { print }
    ' "$record"
}

# --- agents (kind: render, ADR-027) ---
# Render one neutral AGENT.md record to a harness-native agent file on stdout.
# agent-md (claude/opencode): keep only name/description in the frontmatter (the
# native required subset), inject `generated_*` provenance, body verbatim. The
# neutral-only / deferred keys (kind, model, capabilities, skills, targets) are
# dropped here — model/capability mapping is H-044; consumption is enforced by
# the emitted hook (deploy_agent_hooks), not by frontmatter.
render_agent() {
    local record="$1" srcpath="$2" sha
    sha="$(sha_of "$record")"
    awk -v gf="$srcpath" -v gs="$sha" '
        /^---[[:space:]]*$/ {
            fm++
            if (fm==1) { print; print "generated: true"; print "generated_from: " gf; print "generated_sha: " gs; next }
            if (fm==2) { print; next }
        }
        fm==1 { if ($0 ~ /^(name|description):/) print; next }
        { print }
    ' "$record"
}

# Output path (base-relative) for a rendered skill. <base> is repo- or
# $HOME-relative depending on caller. command/prompt -> a single file;
# skill -> a directory holding SKILL.md (+ verbatim auxiliary files).
skill_out_path() {
    local out_dir="$1" render="$2" name="$3"
    case "$render" in
        command|prompt) printf '%s/%s.md' "$out_dir" "$name" ;;
        *)              printf '%s/%s/SKILL.md' "$out_dir" "$name" ;;
    esac
}

# True if a deployed output carries our provenance marker. Used by --deploy to
# safely prune only files we generated (skill/command carry a `generated: true`
# frontmatter field; prompt carries a leading `generated: true; from:` comment).
is_generated_output() {
    [[ -f "$1" ]] || return 1
    grep -q 'generated: true' "$1" 2>/dev/null
}

# Validate a SKILL.md's YAML frontmatter against the committed schema's
# `required` keys — presence and non-emptiness of each top-level key, nothing
# more. The schema's `type`/`const`/`minLength` clauses document the contract for
# a human reader and for any real JSON Schema validator; this loop does not
# evaluate them, so do not read a passing --check as type validation. Fails
# loudly with file context so a malformed skill never renders.
validate_skill_frontmatter() {
    local f="$1" schema="$2" req
    [[ -f "$schema" ]] || { printf '[ERROR] skill schema not found: %s\n' "$schema" >&2; return 1; }
    if [[ "$(sed -n '1p' "$f")" != "---" ]]; then
        printf '[ERROR] %s:1: missing YAML frontmatter (expected --- on line 1)\n' "$f" >&2; return 1
    fi
    if ! awk 'NR>1 && /^---[[:space:]]*$/{found=1; exit} END{exit !found}' "$f"; then
        printf '[ERROR] %s:1: unterminated frontmatter (no closing ---)\n' "$f" >&2; return 1
    fi
    while IFS= read -r req; do
        if ! awk -v k="$req" '
                /^---[[:space:]]*$/ { n++; next }
                n==1 && $0 ~ ("^" k ":[[:space:]]*[^[:space:]]") { found=1 }
                n>=2 { exit }
                END { exit found?0:1 }' "$f"; then
            printf '[ERROR] %s: required frontmatter key "%s" missing or empty\n' "$f" "$req" >&2
            return 1
        fi
    done < <(jq -r '.required[]' "$schema")
}

# Read a single-line frontmatter value from a SKILL.md (empty if absent).
skill_field() {
    awk -v k="$2" '
        /^---[[:space:]]*$/ { n++; next }
        n==1 && $0 ~ ("^" k ":") { sub(/^[^:]*:[[:space:]]*/,""); print; exit }
        n>=2 { exit }' "$1"
}

# Build a markdown skill catalog (one bullet per skill targeting <agent>), for
# agents with no per-skill mechanism (e.g. copilot's single instructions file).
# Deterministic order (glob sorts). Args: <record_dir> <agent>
build_skill_catalog() {
    local recdir="$1" agent="$2" sk_dir name desc
    for sk_dir in "$recdir"/*/; do
        [[ -f "$sk_dir/SKILL.md" ]] || continue
        name="$(basename "$sk_dir")"
        skill_targets_agent "$sk_dir/SKILL.md" "$agent" || continue
        desc="$(skill_field "$sk_dir/SKILL.md" description)"
        printf -- '- **%s** — %s\n' "$name" "$desc"
    done
}

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

    # 3. skills (option A): vault 00_meta/skills/<n> -> committed record only.
    #    Rendering to agent-native $HOME paths happens at deploy time (--deploy),
    #    NOT here — records are the committed SSOT; deployed copies are derived.
    if has_skills; then
        local sk_vsub sk_recdir sk_schema sk_dir name rec
        sk_vsub="$(jq -r '.skills.vault_subpath' "$MANIFEST")"
        sk_recdir="$REPO_ROOT/$(jq -r '.skills.record_dir' "$MANIFEST")"
        sk_schema="$REPO_ROOT/$(jq -r '.skills.schema // "harness/skill-frontmatter.schema.json"' "$MANIFEST")"
        if [[ ! -d "$VAULT_PATH/$sk_vsub" ]]; then
            printf '[ERROR] --refresh needs the vault skills dir: %s\n' "$VAULT_PATH/$sk_vsub" >&2
            exit 2
        fi
        mkdir -p "$sk_recdir"
        # validate frontmatter, then copy the WHOLE skill dir to the record
        # (SKILL.md + any auxiliary reference files / scripts) verbatim, then
        # stamp SKILL.md alone with generated_* provenance (HARNESS-069).
        for sk_dir in "$VAULT_PATH/$sk_vsub"/*/; do
            [[ -f "$sk_dir/SKILL.md" ]] || continue
            validate_skill_frontmatter "$sk_dir/SKILL.md" "$sk_schema"
            name="$(basename "$sk_dir")"
            rm -rf "${sk_recdir:?}/$name"
            mkdir -p "$sk_recdir/$name"
            cp -rf "$sk_dir"* "$sk_recdir/$name/"
            inject_record_provenance "$sk_recdir/$name/SKILL.md" "$sk_vsub/$name/SKILL.md" "$(sha_of "$sk_dir/SKILL.md")"
            printf '[refresh] skill record: %s/%s\n' "$(jq -r '.skills.record_dir' "$MANIFEST")" "$name"
        done
        # drop stale records whose vault source no longer exists
        for rec in "$sk_recdir"/*/; do
            [[ -d "$rec" ]] || continue
            name="$(basename "$rec")"
            [[ -d "$VAULT_PATH/$sk_vsub/$name" ]] || { rm -rf "$rec"; printf '[refresh] dropped stale record: %s\n' "$name"; }
        done
    fi

    # 4. agents (ADR-027): vault 00_meta/agents/definitions/<n> -> committed record
    #    (frontmatter validated). Like skills, render to $HOME is at --deploy time.
    if has_agents; then
        local ag_vsub ag_recdir ag_schema ag_dir
        ag_vsub="$(jq -r '.agents.vault_subpath' "$MANIFEST")"
        ag_recdir="$REPO_ROOT/$(jq -r '.agents.record_dir' "$MANIFEST")"
        ag_schema="$REPO_ROOT/$(jq -r '.agents.schema // "harness/agent-frontmatter.schema.json"' "$MANIFEST")"
        if [[ ! -d "$VAULT_PATH/$ag_vsub" ]]; then
            printf '[ERROR] --refresh needs the vault agents dir: %s\n' "$VAULT_PATH/$ag_vsub" >&2
            exit 2
        fi
        mkdir -p "$ag_recdir"
        for ag_dir in "$VAULT_PATH/$ag_vsub"/*/; do
            [[ -f "$ag_dir/AGENT.md" ]] || continue
            validate_skill_frontmatter "$ag_dir/AGENT.md" "$ag_schema"
            name="$(basename "$ag_dir")"
            rm -rf "${ag_recdir:?}/$name"
            mkdir -p "$ag_recdir/$name"
            cp -rf "$ag_dir"* "$ag_recdir/$name/"
            inject_record_provenance "$ag_recdir/$name/AGENT.md" "$ag_vsub/$name/AGENT.md" "$(sha_of "$ag_dir/AGENT.md")"
            printf '[refresh] agent record: %s/%s\n' "$(jq -r '.agents.record_dir' "$MANIFEST")" "$name"
        done
        for rec in "$ag_recdir"/*/; do
            [[ -d "$rec" ]] || continue
            name="$(basename "$rec")"
            [[ -d "$VAULT_PATH/$ag_vsub/$name" ]] || { rm -rf "$rec"; printf '[refresh] dropped stale agent record: %s\n' "$name"; }
        done
    fi

    printf '[refresh] OK\n'
}

# --- deploy (offline): render committed records to per-agent $HOME paths ---
do_deploy() {
    require_tools
    if ! has_skills && ! has_agents; then
        printf '[deploy] no skills/agents block in manifest; nothing to deploy\n'
        return 0
    fi
    if has_skills; then deploy_skills; fi
    if has_agents; then deploy_agents; fi
    if has_doctrine && has_agents; then
        deploy_doctrine "$REPO_ROOT/$(jq -r '.agents.record_dir' "$MANIFEST")"
    fi
    printf '[deploy] OK\n'
}

# Render committed skill records to their per-agent $HOME paths (offline),
# de-symlinking first, then inject the copilot catalog.
deploy_skills() {
    local sk_vsub sk_recdir agent render dir requires sk_dir name outp destdir
    sk_vsub="$(jq -r '.skills.vault_subpath' "$MANIFEST")"
    sk_recdir="$REPO_ROOT/$(jq -r '.skills.record_dir' "$MANIFEST")"
    if [[ ! -d "$sk_recdir" ]]; then
        printf '[ERROR] no skill records at %s (run --refresh first)\n' "$sk_recdir" >&2
        exit 2
    fi

    # 1. render each record -> its per-agent $HOME path, de-symlinking first so a
    #    pre-existing vault symlink (BUG-100) becomes a regular copy.
    while IFS=$'\t' read -r agent render dir requires; do
        # Optional per-target "requires_command" (manifest-declared, not
        # hardcoded here): a tool this repo does not auto-install itself
        # (Copilot, per BUG-003's explicit "no auto-install" policy) only gets
        # its config deployed once it is genuinely present -- the same
        # detect-and-act rule setup-linux.sh already applies to Copilot's
        # instructions.md. Tools this repo DOES install (opencode, agy, pi)
        # have no requires_command and deploy unconditionally, same as before.
        if [[ -n "$requires" ]] && ! command -v "$requires" >/dev/null 2>&1; then
            printf '[deploy] skill target %s skipped: %s not on PATH\n' "$agent" "$requires"
            continue
        fi
        for sk_dir in "$sk_recdir"/*/; do
            [[ -f "$sk_dir/SKILL.md" ]] || continue
            name="$(basename "$sk_dir")"
            skill_targets_agent "$sk_dir/SKILL.md" "$agent" || continue
            outp="$HOME/$(skill_out_path "$dir" "$render" "$name")"
            case "$render" in
                command|prompt)
                    [[ -L "$outp" ]] && rm -f "$outp"
                    mkdir -p "$(dirname "$outp")"
                    render_skill "$render" "$sk_dir/SKILL.md" "$sk_vsub/$name/SKILL.md" > "$outp"
                    ;;
                *)
                    destdir="$(dirname "$outp")"   # $HOME/<dir>/<name>
                    [[ -L "$destdir" ]] && rm -f "$destdir"
                    mkdir -p "$destdir"
                    cp -rf "$sk_dir"* "$destdir/" 2>/dev/null || true   # aux files verbatim
                    render_skill "$render" "$sk_dir/SKILL.md" "$sk_vsub/$name/SKILL.md" > "$outp"
                    ;;
            esac
            printf '[deploy] skill -> %s\n' "$outp"
        done
    done < <(jq -r '.skills.deploy[] | "\(.agent)\t\(.render)\t\(.dir)\t\(.requires_command // "")"' "$MANIFEST")

    # 2. prune our own stale outputs (skill removed, or targets[] dropped this
    #    agent). Safe: only files carrying our provenance marker are removed.
    deploy_prune "$sk_recdir"

    # 3. copilot catalog: inject into the $HOME instructions file (not committed).
    if jq -e '.skills.catalog' "$MANIFEST" >/dev/null 2>&1; then
        local cat_agent cat_file tmpcat cat_sha cat_begin
        cat_agent="$(jq -r '.skills.catalog.agent' "$MANIFEST")"
        cat_file="$HOME/$(jq -r '.skills.catalog.file' "$MANIFEST")"
        if [[ -f "$cat_file" ]] && validate_markers "$cat_file" 2>/dev/null; then
            tmpcat="$(mktemp)"
            build_skill_catalog "$sk_recdir" "$cat_agent" > "$tmpcat"
            cat_sha="$(sha_of "$tmpcat")"
            cat_begin="$BEGIN_PREFIX (sha256:$cat_sha) — skill catalog from vault $sk_vsub; edit there + re-run setup, do NOT edit between markers -->"
            replace_region "$cat_file" "$cat_begin" "$tmpcat"
            rm -f "$tmpcat"
            printf '[deploy] catalog -> %s\n' "$cat_file"
        else
            printf '[deploy] catalog target absent or unmarked, skipping: %s\n' "$cat_file" >&2
        fi
    fi
}

# Remove previously-deployed outputs that are now stale: the skill no longer has
# a record, or its targets[] no longer includes that agent. Only touches files
# that carry our provenance marker (never user-authored skills).
#
# The marker is the only proof of ownership, so an output written by any
# PRE-provenance deploy mechanism is unprunable — it survives every re-deploy and
# keeps serving a stale copy (HARNESS-053: five skills frozen in ~/.gemini/skills
# since the symlink era, two of them fenced to another agent). Deleting unmarked
# files is not an option (that is how a hand-authored skill dies), so report the
# residue instead: warn when an unmarked entry SHADOWS a record name. Third-party
# skills that own their name — no record, no shadow — stay silent.
warn_unmanaged_output() {
    local recdir="$1" name="$2" path="$3"
    [[ -f "$recdir/$name/SKILL.md" ]] || return 0
    printf '[deploy] WARN unmanaged copy of a managed skill (no provenance marker, not prunable) -> %s\n' "$path" >&2
}

deploy_prune() {
    local sk_recdir="$1" agent render dir f sd name
    while IFS=$'\t' read -r agent render dir; do
        case "$render" in
            command|prompt)
                for f in "$HOME/$dir"/*.md; do
                    [[ -f "$f" ]] || continue
                    name="$(basename "$f" .md)"
                    is_generated_output "$f" || { warn_unmanaged_output "$sk_recdir" "$name" "$f"; continue; }
                    if [[ ! -f "$sk_recdir/$name/SKILL.md" ]] || ! skill_targets_agent "$sk_recdir/$name/SKILL.md" "$agent"; then
                        rm -f "$f"; printf '[deploy] pruned stale -> %s\n' "$f"
                    fi
                done
                ;;
            *)
                for sd in "$HOME/$dir"/*/; do
                    [[ -d "$sd" ]] || continue
                    name="$(basename "$sd")"
                    is_generated_output "$sd/SKILL.md" || { warn_unmanaged_output "$sk_recdir" "$name" "$sd"; continue; }
                    if [[ ! -f "$sk_recdir/$name/SKILL.md" ]] || ! skill_targets_agent "$sk_recdir/$name/SKILL.md" "$agent"; then
                        rm -rf "$sd"; printf '[deploy] pruned stale -> %s\n' "$sd"
                    fi
                done
                ;;
        esac
    done < <(jq -r '.skills.deploy[] | "\(.agent)\t\(.render)\t\(.dir)"' "$MANIFEST")
}

# --- agents (ADR-027): render neutral AGENT.md records to each harness's native
# agent path, then enforce skill consumption by injecting a presence region into
# each harness's always-loaded instructions file. Presence is UNIFORM across
# harnesses — one marked-region injection primitive, no provider-specific hook.
# The plugin primitives (chat.system.transform / session_start / PreToolUse) are
# the Action level, deferred to H-045. Mirrors deploy_skills; agent-md render is
# a single file. ---
deploy_agents() {
    local ag_vsub ag_recdir agent render dir ag_dir name outp
    ag_vsub="$(jq -r '.agents.vault_subpath' "$MANIFEST")"
    ag_recdir="$REPO_ROOT/$(jq -r '.agents.record_dir' "$MANIFEST")"
    if [[ ! -d "$ag_recdir" ]]; then
        printf '[ERROR] no agent records at %s (run --refresh first)\n' "$ag_recdir" >&2
        exit 2
    fi
    # 1. render each AGENT.md record -> its per-harness $HOME path (single file),
    #    de-symlinking first (BUG-100 safety).
    while IFS=$'\t' read -r agent render dir; do
        for ag_dir in "$ag_recdir"/*/; do
            [[ -f "$ag_dir/AGENT.md" ]] || continue
            name="$(basename "$ag_dir")"
            skill_targets_agent "$ag_dir/AGENT.md" "$agent" || continue
            outp="$HOME/$dir/$name.md"
            [[ -L "$outp" ]] && rm -f "$outp"
            mkdir -p "$(dirname "$outp")"
            render_agent "$ag_dir/AGENT.md" "$ag_vsub/$name/AGENT.md" > "$outp"
            printf '[deploy] agent -> %s\n' "$outp"
        done
    done < <(jq -r '.agents.deploy[] | "\(.agent)\t\(.render)\t\(.dir)"' "$MANIFEST")
    # 2. presence-level determinism: inject forced skills into each harness's
    #    always-loaded instructions file (uniform injection — no provider hook).
    if jq -e '.agents.presence' "$MANIFEST" >/dev/null 2>&1; then
        deploy_agent_presence "$ag_recdir"
    fi
}

# Build the agent-presence block for <agent>: one line per persona that targets
# this harness, naming its forced skills. Deterministic order (glob sorts). Empty
# output (no persona targets this harness) tells the caller to skip injection.
# Args: <record_dir> <agent>
build_agent_presence() {
    local ag_recdir="$1" agent="$2" ag_dir name skills_line first=1
    for ag_dir in "$ag_recdir"/*/; do
        [[ -f "$ag_dir/AGENT.md" ]] || continue
        name="$(basename "$ag_dir")"
        skill_targets_agent "$ag_dir/AGENT.md" "$agent" || continue
        if [[ "$first" == 1 ]]; then
            printf '## Active agent personas — forced skills\n\n'
            printf 'These personas enforce their skills by injection (determinism by code, not memory). When acting as one, you MUST consume its skills.\n\n'
            first=0
        fi
        skills_line="$(skill_field "$ag_dir/AGENT.md" skills)"
        printf -- '- **%s** — MUST consume: %s\n' "$name" "${skills_line:-none}"
    done
}

# --- doctrine (HARNESS-054) ---
# Most harnesses receive the whole cross-agent AGENTS.md (opencode, pi) or a full
# instructions file of their own (claude, copilot), so the enforced rules travel
# with it. Two cannot, and the reason is a documented platform limit rather than
# a preference:
#
#   agy   — Antigravity reads ~/.gemini/GEMINI.md and caps EACH rules file at
#           12000 characters; AGENTS.md is ~21851. It also shares that file with
#           Gemini CLI, so the region is injected, never overwritten.
#   codex — Codex stops adding instruction files once the global+project chain
#           reaches 32 KiB, so a full global copy would crowd out the repo's own
#           AGENTS.md — the more specific file, and the one you least want lost.
#
# Both therefore receive the COMPACT payload: the enforced rules plus the agent
# presence block, ~2 KB. Same content, same marker mechanism, smaller render —
# exclusion by demonstrated incompatibility, not by agent identity. Rationale and
# sources live in manifest.json next to each row.
has_doctrine() { jq -e '.doctrine.deploy' "$MANIFEST" >/dev/null 2>&1; }

deploy_doctrine() {
    local ag_recdir="$1" agent file cap shadow file_abs payload sha begin tmp chars ids
    mapfile -t ids < <(jq -r '.doctrine.inject[]' "$MANIFEST")
    while IFS=$'\t' read -r agent file cap shadow; do
        [[ -n "$agent" ]] || continue
        file_abs="$HOME/$file"
        payload="$(mktemp)"
        {
            printf '## Non-negotiable rules (harness-enforced)\n\n'
            render_region "${ids[@]}"
            printf '\n'
            build_agent_presence "$ag_recdir" "$agent"
        } > "$payload"

        # A shadow file wins over ours at read time, so a silent deploy here
        # would look successful while changing nothing the agent ever reads.
        if [[ -n "$shadow" && -e "$HOME/$shadow" ]]; then
            printf '[deploy] WARN %s shadows %s — the deployed doctrine is never read\n' \
                "$HOME/$shadow" "$file_abs" >&2
        fi

        mkdir -p "$(dirname "$file_abs")"
        if [[ ! -f "$file_abs" ]]; then
            printf '# Global rules\n\n> Cross-agent doctrine. The marked region is generated — edit the vault pattern and re-run setup.\n' \
                > "$file_abs"
            printf '[deploy] doctrine: created %s\n' "$file_abs"
        fi

        sha="$(sha_of "$payload")"
        begin="$BEGIN_PREFIX (sha256:$sha) — cross-agent doctrine from vault $(jq -r '.vault_subpath' "$MANIFEST"); edit there + re-run setup, do NOT edit between markers -->"
        tmp="$(mktemp)"
        if grep -q "^$BEGIN_PREFIX" "$file_abs" && grep -qF "$END_MARKER" "$file_abs"; then
            awk -v beginm="$begin" -v endm="$END_MARKER" -v bp="$BEGIN_PREFIX" -v cf="$payload" '
                index($0,bp)==1 { print beginm; while ((getline l < cf) > 0) print l; close(cf); skip=1; next }
                $0==endm { if (skip){ print; skip=0; next } }
                skip { next }
                { print }
            ' "$file_abs" > "$tmp"
        else
            cat "$file_abs" > "$tmp"
            { printf '\n%s\n' "$begin"; cat "$payload"; printf '%s\n' "$END_MARKER"; } >> "$tmp"
        fi
        mv "$tmp" "$file_abs"
        rm -f "$payload"
        printf '[deploy] doctrine -> %s\n' "$file_abs"

        # The cap covers the WHOLE file, user content included: the platform
        # truncates or rejects the file, and it does not care who wrote which half.
        if [[ "$cap" != 0 ]]; then
            chars="$(wc -m < "$file_abs")"
            if (( chars > cap )); then
                printf '[deploy] WARN %s is %s characters, over the %s the platform documents — content past the cap may never be read\n' \
                    "$file_abs" "$chars" "$cap" >&2
            fi
        fi
    done < <(jq -r '.doctrine.deploy[] | "\(.agent)\t\(.file)\t\(.char_cap // 0)\t\(.shadowed_by // "")"' "$MANIFEST")
}

# Inject (or refresh) the agent-presence region in a harness instructions file.
# Uses the AGENT-PRESENCE marker namespace, so it never disturbs the patterns /
# skill-catalog region (BEGIN_PREFIX). Replaces an existing presence region in
# place; appends a fresh one if absent. Skips a target file that does not exist.
# Args: <file_abs> <content_file>
inject_agent_presence() {
    local file="$1" content_file="$2" sha begin tmp
    if [[ ! -f "$file" ]]; then
        printf '[deploy] presence target absent, skipping: %s\n' "$file" >&2
        return 0
    fi
    sha="$(sha_of "$content_file")"
    begin="$AGENT_BEGIN_PREFIX (sha256:$sha) — agent presence from vault agent definitions; edit there + re-run setup, do NOT edit between markers -->"
    tmp="$(mktemp)"
    if grep -q "^$AGENT_BEGIN_PREFIX" "$file" && grep -qF "$AGENT_END_MARKER" "$file"; then
        # replace the existing presence region in place (mirrors replace_region)
        awk -v beginm="$begin" -v endm="$AGENT_END_MARKER" -v bp="$AGENT_BEGIN_PREFIX" -v cf="$content_file" '
            index($0,bp)==1 { print beginm; while ((getline l < cf) > 0) print l; close(cf); skip=1; next }
            $0==endm { if (skip){ print; skip=0; next } }
            skip { next }
            { print }
        ' "$file" > "$tmp"
    else
        # append a fresh presence region at the end of the file
        cat "$file" > "$tmp"
        { printf '\n%s\n' "$begin"; cat "$content_file"; printf '%s\n' "$AGENT_END_MARKER"; } >> "$tmp"
    fi
    mv "$tmp" "$file"
}

# Presence-level determinism for every configured harness: build the persona
# block for that harness and inject it into its instructions file. One uniform
# mechanism (marked-region injection) across claude / opencode / pi / copilot.
deploy_agent_presence() {
    local ag_recdir="$1" agent file file_abs tmp
    while IFS=$'\t' read -r agent file; do
        [[ -n "$agent" ]] || continue
        tmp="$(mktemp)"
        build_agent_presence "$ag_recdir" "$agent" > "$tmp"
        if [[ ! -s "$tmp" ]]; then rm -f "$tmp"; continue; fi   # no persona targets this harness
        file_abs="$HOME/$file"
        inject_agent_presence "$file_abs" "$tmp"
        rm -f "$tmp"
        printf '[deploy] presence -> %s (%s)\n' "$file_abs" "$agent"
    done < <(jq -r '.agents.presence[] | "\(.agent)\t\(.file)"' "$MANIFEST")
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

    # skills (option A): validate each committed record renders cleanly. There are
    # no committed agent outputs to diff — records are the SSOT and deployed copies
    # derive from them at --deploy time. Offline: needs only the committed record +
    # schema. (On-machine deployed-vs-record comparison lives in healthcheck.)
    if has_skills; then
        local sk_vsub sk_recdir sk_schema sk_dir name render tmp
        sk_vsub="$(jq -r '.skills.vault_subpath' "$MANIFEST")"
        sk_recdir="$REPO_ROOT/$(jq -r '.skills.record_dir' "$MANIFEST")"
        sk_schema="$REPO_ROOT/$(jq -r '.skills.schema // "harness/skill-frontmatter.schema.json"' "$MANIFEST")"
        if [[ ! -d "$sk_recdir" ]]; then
            printf '[DRIFT] no skill records at %s (run --refresh)\n' "$sk_recdir" >&2
            drift=1
        else
            for sk_dir in "$sk_recdir"/*/; do
                [[ -f "$sk_dir/SKILL.md" ]] || continue
                name="$(basename "$sk_dir")"
                if ! validate_skill_frontmatter "$sk_dir/SKILL.md" "$sk_schema"; then
                    drift=1; continue
                fi
                for render in $(jq -r '.skills.deploy[].render' "$MANIFEST" | sort -u); do
                    tmp="$(mktemp)"
                    if ! render_skill "$render" "$sk_dir/SKILL.md" "$sk_vsub/$name/SKILL.md" > "$tmp" 2>/dev/null; then
                        printf '[DRIFT] record %s fails to render as %s (run --refresh)\n' "$name" "$render" >&2
                        drift=1
                    fi
                    rm -f "$tmp"
                done
                printf '[check] OK -> record %s\n' "$name"
            done
        fi
    fi

    # agents (ADR-027): each committed AGENT.md record validates + renders cleanly.
    if has_agents; then
        local ag_vsub ag_recdir ag_schema ag_dir name tmp
        ag_vsub="$(jq -r '.agents.vault_subpath' "$MANIFEST")"
        ag_recdir="$REPO_ROOT/$(jq -r '.agents.record_dir' "$MANIFEST")"
        ag_schema="$REPO_ROOT/$(jq -r '.agents.schema // "harness/agent-frontmatter.schema.json"' "$MANIFEST")"
        if [[ ! -d "$ag_recdir" ]]; then
            printf '[DRIFT] no agent records at %s (run --refresh)\n' "$ag_recdir" >&2
            drift=1
        else
            for ag_dir in "$ag_recdir"/*/; do
                [[ -f "$ag_dir/AGENT.md" ]] || continue
                name="$(basename "$ag_dir")"
                if ! validate_skill_frontmatter "$ag_dir/AGENT.md" "$ag_schema"; then
                    drift=1; continue
                fi
                tmp="$(mktemp)"
                if ! render_agent "$ag_dir/AGENT.md" "$ag_vsub/$name/AGENT.md" > "$tmp" 2>/dev/null; then
                    printf '[DRIFT] agent record %s fails to render (run --refresh)\n' "$name" >&2
                    drift=1
                fi
                rm -f "$tmp"
                printf '[check] OK -> agent %s\n' "$name"
            done
        fi
    fi

    if [[ "$drift" -ne 0 ]]; then
        printf '[check] FAIL: harness drift detected\n' >&2
        exit 1
    fi
    printf '[check] OK: no harness drift\n'
}

# --- dispatch ---
case "${1:-}" in
    --refresh) do_refresh ;;
    --deploy)  do_deploy ;;
    --check)   do_check ;;
    -h|--help) usage ;;
    *)         usage >&2; exit 2 ;;
esac
