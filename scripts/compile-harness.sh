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
            if (fm==1) {
                print
                print "generated: true"
                print "generated_from: " gf
                print "generated_sha: " gs
                next
            }
            if (fm==2) {
                print
                next
            }
        }
        fm==1 {
            if (/^[a-zA-Z0-9_-]+:/) {
                if (kind == "command" && /^name:/) {
                    keep = 0
                    next
                }
                # The record (HARNESS-069) already carries its own generated_* fields.
                # Strip them here so deploy injects one fresh set above.
                if (/^generated(_from|_sha)?:/) {
                    keep = 0
                    next
                }
                # Native skill execution fields recognized by agent runtimes (Claude Code, AGY, OpenCode, Copilot)
                if (/^(name|description|allowed-tools|when_to_use|model|effort|context|argument-hint|arguments|user-invocable|disable-model-invocation):/) {
                    keep = 1
                    print
                    next
                }
                # Drop neutral/store-only keys (id, type, status, created, owner, paths, keywords, requires, targets, source, license, etc.)
                # In particular, dropping `paths:` ensures Claude Code discovers skills as unconditional at session start.
                keep = 0
                next
            }
            # Continuation line (indented multiline value)
            if (keep == 1) {
                print
            }
            next
        }
        { print }
    ' "$record"
}

# Resolve a neutral model tier (top|mid|low) to $agent's model id.
#
# Shells out to `dotf` rather than reading harness/model-map.json with jq, and
# that is the whole point: jq here would restate the resolution rules in a second
# place and skip the schema validation entirely, so a map the Go validator
# rejects would render clean. Routing rules true in one place only (ADR-035).
#
# THREE OUTCOMES, and the difference between the last two is the whole design:
#   0  resolved — the model id is on stdout
#   1  the resolver RAN AND REFUSED — an undeclared tier, or a map that is absent
#      or schema-invalid. A genuine routing error, and C15's case: fail loudly.
#   2  the resolver is UNAVAILABLE — no dotf on PATH, or a dotf too old to carry
#      this subcommand (#1158, measured: the deployed binary routinely predates
#      the tree). NOT C15: C15 governs a map that cannot be READ, and an absent
#      binary is a bootstrap state, not an unreadable map. setup-linux.sh installs
#      dotf best-effort (`install_dotf || log_warning`), so treating this as fatal
#      would make a warned-past dependency a hard prerequisite of the entire
#      harness deploy. The caller warns loudly and renders without the model line,
#      which is what this script did unconditionally before — never worse than the
#      status quo, and it still SAYS SO, which is the honest-degradation bar
#      (ADR-032).
#
# Telling outcome 1 from outcome 2 needs a CAPABILITY probe, not the exit status:
# measured 2026-08-21, a dotf predating this subcommand answers
# `harness resolve-tier top --harness claude` with `unknown flag: --harness` and
# exit 1 — byte-identical in status to a genuine routing refusal. Asking whether
# the binary KNOWS the subcommand is the only question whose answer does not
# depend on the arguments it failed to parse. TestHarnessHelpListsResolveTier
# pins the string this greps for, so the probe cannot rot into always-false.
# Generalised over the subcommand name, because there are now two consumers of
# this file's registries and each needs the same "is the binary new enough"
# question asked about ITS OWN subcommand. TestHarnessHelpListsSubcommands pins
# both strings from the Go side.
dotf_knows_subcommand() {
    local want="$1" help
    help="$(dotf harness --help 2>/dev/null)" || return 1
    # Matched from a here-string rather than through a pipe. `cmd | grep -q`
    # closes the pipe the moment it matches, and under `set -o pipefail` a
    # producer killed by the resulting SIGPIPE makes the pipeline exit 141 —
    # reporting "too old" for a binary that just proved it is current. No pipe,
    # no such failure mode, and the anchored match is unchanged.
    grep -q "^[[:space:]]*${want}[[:space:]]" <<<"$help"
}

resolve_model_tier() {
    local tier="$1" agent="$2" out
    type -P dotf >/dev/null 2>&1 || return 2
    dotf_knows_subcommand resolve-tier || return 2
    # The resolver's stderr is deliberately NOT swallowed. It is the only place
    # that distinguishes "this tier declares no model for this harness" from
    # "the map itself is unreadable" — naming the ghost pool, the bad keyword or
    # the missing schema. The caller's own message cannot know which, so hiding
    # this one made a schema-invalid map read as a tier problem. stdout is still
    # captured, so only the diagnosis flows.
    if ! out="$(dotf harness resolve-tier "$tier" --harness "$agent" --repo-root "$REPO_ROOT")"; then
        return 1
    fi
    # A model id is one non-empty token. Behind the capability probe above, a
    # whitespace-bearing answer can no longer mean "stale binary printing help" —
    # the resolver ran and answered, so the defect is in the MAP (#1168). Saying
    # "dotf is too old" here sent the operator to rebuild a binary that is fine,
    # while the real fault sat in model-map.json. Exit 3, its own outcome, so the
    # caller can name the value and the file. The schema permits it because its
    # non-blank pattern anchors only the first character (#1159).
    case "$out" in
        '') return 2 ;;
        *[[:space:]]*)
            printf '[ERROR] %s gives tier "%s" on harness "%s" the id "%s" — a model id is one token with no whitespace\n' \
                "harness/model-map.json" "$tier" "$agent" "$out" >&2
            return 3 ;;
    esac
    printf '%s\n' "$out"
}

# Print the frontmatter `model:` line for one record on one harness, or nothing.
#
# Empty output is the answer in TWO cases that must stay distinguishable from a
# failure: the record declares no tier, and the resolver is unavailable. Both
# render exactly as this script did before tiers were consumed at all. Only a
# resolver that RAN AND REFUSED returns non-zero, because only then is something
# actually wrong with the map.
#
# Extracted from deploy_agents to keep that loop responsible for deployment
# ordering alone (the repo's <40-line / complexity<10 rule).
agent_model_line() {
    local record="$1" agent="$2" name="$3" tier model_id rc=0
    tier="$(skill_field "$record" model)"
    [ -n "$tier" ] || return 0
    # Assigned apart from `local` on purpose: `local x="$(cmd)"` reports local's
    # own status, so a failed resolve would be invisible and render `model: `
    # with nothing after it. `|| rc=$?` keeps set -e from taking the branch away.
    model_id="$(resolve_model_tier "$tier" "$agent")" || rc=$?
    case "$rc" in
        0) printf 'model: %s\n' "$model_id" ;;
        2) printf '[WARN] cannot resolve model tier "%s" for harness "%s": dotf is absent, or predates the resolve-tier subcommand\n' \
               "$tier" "$agent" >&2
           printf '       %s deploys WITHOUT a model line; install/rebuild dotf and re-run --deploy\n' "$name" >&2
           ;;
        3) # The map answered with something that is not a model id, and
           # resolve_model_tier already named the value and the file. Still fatal
           # (bad data, C15) but never "rebuild dotf".
           return 1 ;;
        *) # The cause is on the line(s) the resolver already printed: an
           # undeclared tier and an unreadable map both land here, and only it
           # knows which. Point at that rather than asserting a cause this frame
           # cannot tell apart.
           printf '[ERROR] resolving tier "%s" for harness "%s" failed; see the resolver error above\n' \
               "$tier" "$agent" >&2
           return 1 ;;
    esac
}

# Print the frontmatter capability line for one record on one harness, or nothing.
#
# Exactly the shape of agent_model_line, one field over, and the same three
# outcomes for the same reasons: a record declaring no capabilities renders
# without the field, an unavailable resolver warns and degrades, and only a
# resolver that ran and refused is fatal.
#
# The whole LINE comes back from dotf, field name included, because the native
# field differs per harness — `tools:` for claude, `permission:` for opencode —
# and this frame has no business knowing which.
agent_capability_line() {
    local record="$1" agent="$2" name="$3" caps rc=0 line
    caps="$(skill_field "$record" capabilities)"
    [ -n "$caps" ] || return 0
    # Strip the YAML flow-sequence brackets: the record writes
    # `capabilities: [read, search]`, the CLI takes a comma list.
    #
    # The brackets are ESCAPED. Unescaped, `${caps#[}` is a pattern opening a
    # bracket expression that never closes: bash tolerates it and strips the
    # literal, zsh aborts with `bad pattern: [` at RUN time. `zsh -n` does not
    # catch it because it is a pattern error, not a syntax error — so the usual
    # syntax check gives false confidence here. Measured 2026-08-22.
    caps="${caps#\[}"; caps="${caps%\]}"; caps="${caps// /}"
    [ -n "$caps" ] || return 0
    type -P dotf >/dev/null 2>&1 || rc=2
    if [ "$rc" -eq 0 ] && ! dotf_knows_subcommand resolve-capabilities; then rc=2; fi
    if [ "$rc" -eq 0 ]; then
        line="$(dotf harness resolve-capabilities "$caps" --harness "$agent" --repo-root "$REPO_ROOT")" || rc=1
    fi
    case "$rc" in
        0) printf '%s\n' "$line" ;;
        2) printf '[WARN] cannot resolve capabilities "%s" for harness "%s": dotf is absent, or predates the resolve-capabilities subcommand\n' \
               "$caps" "$agent" >&2
           printf '       %s deploys WITHOUT a capability line; install/rebuild dotf and re-run --deploy\n' "$name" >&2
           ;;
        *) printf '[ERROR] resolving capabilities "%s" for harness "%s" failed; see the resolver error above\n' \
               "$caps" "$agent" >&2
           return 1 ;;
    esac
}

# --- agents (kind: render, ADR-027) ---
# Render one neutral AGENT.md record to a harness-native agent file on stdout.
# agent-md (claude/opencode): keep only name/description in the frontmatter (the
# native required subset), inject `generated_*` provenance, emit the already-
# resolved model line, body verbatim. The remaining neutral-only / deferred keys
# (kind, capabilities, skills, targets) are dropped here — capability mapping is
# #560; consumption is enforced by the emitted hook (deploy_agent_hooks), not by
# frontmatter.
#
# `model_line` is passed in, ALREADY RESOLVED, rather than resolved here. That
# keeps this a pure renderer, and it is what lets --check stay environment-free:
# whether a tier resolves depends on the deploy machine (is dotf installed, is
# model-map.json readable), while whether a RECORD renders depends only on the
# record. A drift gate that conflated the two would report drift on a perfectly
# good record just because the machine running CI has no dotf. Empty renders no
# model line, which is also what a record declaring no tier produces.
render_agent() {
    local record="$1" srcpath="$2" model_line="${3:-}" cap_line="${4:-}" sha
    sha="$(sha_of "$record")"
    awk -v gf="$srcpath" -v gs="$sha" -v ml="$model_line" -v cl="$cap_line" '
        /^---[[:space:]]*$/ {
            fm++
            if (fm==1) { print; print "generated: true"; print "generated_from: " gf; print "generated_sha: " gs; next }
            if (fm==2) { if (ml != "") print ml; if (cl != "") print cl; print; next }
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
    # Instruction files FIRST (HARNESS-058/#828): deploy_skills injects the
    # copilot catalog and deploy_agents injects the AGENT-PRESENCE region into
    # these SAME files below. A full-file copy after either would wipe out what
    # they just wrote.
    if jq -e '.agents.presence' "$MANIFEST" >/dev/null 2>&1; then
        deploy_instructions || exit 2
    fi
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

# Copy each agents.presence[] entry's full instruction-file SSOT (already
# injected with the enforced-pattern region by --refresh) to its per-agent
# $HOME path (HARNESS-058/#828). Until now this copy only happened in
# setup-linux.sh, so a standalone `--deploy` run (no full setup) left
# claude/opencode/pi/copilot stale after a merge while agy/codex (the compact
# doctrine payload) stayed current — the same command now converges all six
# surfaces. De-symlinks first (BUG-100 safety, same as deploy_skills/deploy_agents).
# Entries with no `source` (none today) are skipped -- presence-only injection,
# same as before this change.
deploy_instructions() {
    local agent file source requires dest rc=0
    while IFS=$'\t' read -r agent file source requires; do
        [[ -n "$source" ]] || continue
        if [[ -n "$requires" ]] && ! command -v "$requires" >/dev/null 2>&1; then
            printf '[deploy] instructions target %s skipped: %s not on PATH\n' "$agent" "$requires"
            continue
        fi
        if [[ ! -f "$REPO_ROOT/$source" ]]; then
            printf '[ERROR] instruction source missing: %s\n' "$REPO_ROOT/$source" >&2
            rc=1
            continue
        fi
        dest="$HOME/$file"
        [[ -L "$dest" ]] && rm -f "$dest"
        mkdir -p "$(dirname "$dest")"
        cp -f "$REPO_ROOT/$source" "$dest"
        printf '[deploy] instructions -> %s\n' "$dest"
    done < <(jq -r '.agents.presence[] | "\(.agent)\t\(.file)\t\(.source // "")\t\(.requires_command // "")"' "$MANIFEST")
    # A missing source is a manifest/repo defect, not a transient skip -- an
    # [ERROR] line followed by [deploy] OK was the same silently-contradicting
    # shape deploy_agent_presence had (fixed earlier this PR); propagate it so
    # `do_deploy` stops instead of claiming success.
    return "$rc"
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
    local ag_vsub ag_recdir agent render dir ag_dir name outp tmp_agent model_line cap_line
    local failed=() deployed=0
    ag_vsub="$(jq -r '.agents.vault_subpath' "$MANIFEST")"
    ag_recdir="$REPO_ROOT/$(jq -r '.agents.record_dir' "$MANIFEST")"
    if [[ ! -d "$ag_recdir" ]]; then
        printf '[ERROR] no agent records at %s (run --refresh first)\n' "$ag_recdir" >&2
        exit 2
    fi
    # 1. render each AGENT.md record -> its per-harness $HOME path (single file),
    #    de-symlinking first (BUG-100 safety).
    #
    #    Rendered to a temp file and moved on success, never straight to $outp:
    #    a redirect truncates the target BEFORE the renderer runs, so a failed
    #    model-tier resolution would leave an EMPTY agent definition behind —
    #    a file naming no model, which is the failure this render was changed to
    #    prevent. Fail loud, and leave the previous definition intact.
    #
    #    The temp file sits BESIDE the target rather than in $TMPDIR, and is
    #    created by the same redirect as before. Two reasons, both behavioural:
    #    mktemp creates 0600, which would deploy agent files with different
    #    permissions from every other deployed artifact (664 here), and a mv out
    #    of $TMPDIR can cross filesystems, degrading the atomic rename this
    #    relies on into a copy.
    while IFS=$'\t' read -r agent render dir; do
        for ag_dir in "$ag_recdir"/*/; do
            [[ -f "$ag_dir/AGENT.md" ]] || continue
            name="$(basename "$ag_dir")"
            skill_targets_agent "$ag_dir/AGENT.md" "$agent" || continue
            outp="$HOME/$dir/$name.md"
            [[ -L "$outp" ]] && rm -f "$outp"
            mkdir -p "$(dirname "$outp")"
            # COLLECT, do not abort (#1169). Returning on the first bad record
            # left every agent behind it undeployed, for a defect in one — and
            # which ones those were depended on directory iteration order. The
            # operator now sees the complete list in a single run, the same
            # reason `dotf pr triage-queue` reports the whole queue. The exit is
            # still non-zero, and the message says plainly that some agents DID
            # deploy, because a non-zero exit that half-succeeded must not read
            # as "nothing happened".
            if ! model_line="$(agent_model_line "$ag_dir/AGENT.md" "$agent" "$name")"; then
                failed+=("$agent/$name"); continue
            fi
            if ! cap_line="$(agent_capability_line "$ag_dir/AGENT.md" "$agent" "$name")"; then
                failed+=("$agent/$name"); continue
            fi
            tmp_agent="$outp.tmp.$$"
            if ! render_agent "$ag_dir/AGENT.md" "$ag_vsub/$name/AGENT.md" "$model_line" "$cap_line" > "$tmp_agent"; then
                rm -f "$tmp_agent"
                printf '[ERROR] agent render failed: %s -> %s\n' "$ag_dir/AGENT.md" "$outp" >&2
                failed+=("$agent/$name"); continue
            fi
            if ! mv "$tmp_agent" "$outp"; then
                rm -f "$tmp_agent"
                printf '[ERROR] could not replace %s\n' "$outp" >&2
                failed+=("$agent/$name"); continue
            fi
            deployed=$((deployed + 1))
            printf '[deploy] agent -> %s\n' "$outp"
        done
    done < <(jq -r '.agents.deploy[] | "\(.agent)\t\(.render)\t\(.dir)"' "$MANIFEST")
    if [ "${#failed[@]}" -ne 0 ]; then
        printf '[ERROR] %s agent record(s) failed to render: %s\n' "${#failed[@]}" "${failed[*]}" >&2
        # Say which it was. "every OTHER agent deployed" is false when none did,
        # and a summary that overstates what survived is worse than none — the
        # operator decides whether to roll back on exactly this sentence.
        if [ "$deployed" -gt 0 ]; then
            printf '        %s agent(s) DID deploy; re-run --deploy after fixing the records above\n' "$deployed" >&2
        else
            printf '        NO agent deployed; re-run --deploy after fixing the records above\n' >&2
        fi
        return 1
    fi
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
        return 1
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
        # inject_agent_presence returns 1 (a genuine no-op, not a set -e-worthy
        # error) when the target file is absent -- only log success when it
        # actually wrote something, else "presence target absent, skipping"
        # was immediately followed by a contradicting success line.
        if inject_agent_presence "$file_abs" "$tmp"; then
            printf '[deploy] presence -> %s (%s)\n' "$file_abs" "$agent"
        fi
        rm -f "$tmp"
    done < <(jq -r '.agents.presence[] | "\(.agent)\t\(.file)"' "$MANIFEST")
}

# --- coverage: every enforced region reaches every surface, or says why not ---
#
# Why this is not already covered by the region diff below: that diff renders its
# expected side FROM the target's own `inject` list (target_inject), so an id
# missing from the list is missing from BOTH sides and the target reports OK. It
# verifies that injected text matches its record — a consistency check — and is
# structurally blind to a surface being skipped entirely. That blindness is the
# bug this function exists for (HARNESS-072); the risk had been filed against
# the diff, which could never have caught it.
#
# Invariant: for every enforced id x every surface, the id is either injected or
# carries an `opt_out` entry naming that surface with a reason. Silence is not an
# exit — the same rule the Definition of Done applies to people.
#
# A surface is a `targets[].file`, plus the reserved key "doctrine" for the one
# shared inject list feeding every doctrine payload.
check_coverage() {
    local id surfaces surface injected reason gap=0
    mapfile -t surfaces < <(
        jq -r '.targets[].file' "$MANIFEST"
        jq -e '.doctrine' "$MANIFEST" >/dev/null 2>&1 && printf 'doctrine\n'
    )
    while IFS= read -r id; do
        for surface in "${surfaces[@]}"; do
            if [[ "$surface" == doctrine ]]; then
                injected="$(jq -r --arg i "$id" \
                    '[.doctrine.inject[]? | select(. == $i)] | length' "$MANIFEST")"
            else
                injected="$(jq -r --arg i "$id" --arg f "$surface" \
                    '[.targets[] | select(.file == $f) | .inject[] | select(. == $i)] | length' "$MANIFEST")"
            fi
            [[ "$injected" != 0 ]] && continue
            reason="$(jq -r --arg i "$id" --arg s "$surface" \
                '.enforced[] | select(.id == $i) | .opt_out[$s] // empty' "$MANIFEST")"
            if [[ -z "$reason" ]]; then
                printf '[GAP] enforced region "%s" reaches neither surface "%s" nor an opt_out for it\n' "$id" "$surface" >&2
                printf '      -> inject it there, or record enforced[id=%s].opt_out["%s"] with the reason\n' "$id" "$surface" >&2
                gap=1
            else
                printf '[check] OK -> %s excluded from %s (%s)\n' "$id" "$surface" "$reason"
            fi
        done
    done < <(jq -r '.enforced[].id' "$MANIFEST")
    return "$gap"
}

do_check() {
    require_tools
    local file ids drift=0 expected actual
    check_coverage || drift=1
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
