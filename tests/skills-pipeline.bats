#!/usr/bin/env bats
# SDD-008 skill pipeline smoke test (AC8). Renders the committed skill records
# (harness/skills/) to a throwaway HOME and asserts the agent-native discovery
# locations are populated -- the vault -> record -> render -> deploy -> discovery
# round-trip at the filesystem level. Runs offline (no vault); the integration
# container exercises the full setup-linux.sh end-to-end.

load 'lib/refute'

# Required for `run !` in the BUG-771 test below. Declared rather than assumed:
# bats warns (BW02) on flag use without it, and CI resolves bats from
# versions.conf (BATS_VERSION=1.13.0), so the floor is well under what runs.
bats_require_minimum_version 1.5.0

# copilot base file so catalog injection has a marked target
seed_copilot_base() {
    mkdir -p "$1/.copilot"
    printf '## Skills\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n' \
        > "$1/.copilot/copilot-instructions.md"
}

# deploy_shared <name> <PATH>: deploy once into $BATS_FILE_TMPDIR/home-<name>,
# keeping the status and log for shared_home to report.
deploy_shared() {
    local home="$BATS_FILE_TMPDIR/home-$1" rc=0
    seed_copilot_base "$home"
    (cd "$BATS_TEST_DIRNAME/.." && env HOME="$home" PATH="$2" "$COUNTING_SCRIPT" --deploy) > "$home.log" 2>&1 || rc=$?
    echo "$rc" > "$home.status"
}

# shared_home <name>: point $SHARED at a home deployed once in setup_file, and
# fail with that deploy's log if it failed. Tests that only read a deploy's
# result use one of these instead of deploying again: 17 deploys became 6.
shared_home() {
    SHARED="$BATS_FILE_TMPDIR/home-$1"
    [ "$(cat "$SHARED.status")" = 0 ] || { cat "$SHARED.log" >&2; return 1; }
}

# Every --deploy in this file goes through a counting wrapper, so the budget
# test at the end measures invocations instead of grepping for them (CI-004
# AC2). The tests call "$SCRIPT" by absolute path, which a PATH stub never sees.
setup_file() {
    local real
    real="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/scripts/compile-harness.sh"
    export DEPLOY_COUNT="$BATS_FILE_TMPDIR/deploys"
    export COUNTING_SCRIPT="$BATS_FILE_TMPDIR/compile-harness.sh"
    : > "$DEPLOY_COUNT"
    printf '#!/usr/bin/env bash\ncase " $* " in *" --deploy "*) echo deploy >> %q ;; esac\nexec %q "$@"\n' \
        "$DEPLOY_COUNT" "$real" > "$COUNTING_SCRIPT"
    chmod +x "$COUNTING_SCRIPT"

    deploy_shared clean "$PATH"
    # The copilot skill target has a manifest-declared requires_command (BUG-771:
    # native skills must not create ~/.copilot on a box that never installed
    # Copilot). The copilot home needs a fake `copilot` on PATH; the BUG-771
    # test proving the gate needs PATH without one.
    mkdir -p "$BATS_FILE_TMPDIR/stub"
    printf '#!/usr/bin/env bash\nexit 0\n' > "$BATS_FILE_TMPDIR/stub/copilot"
    chmod +x "$BATS_FILE_TMPDIR/stub/copilot"
    deploy_shared copilot "$BATS_FILE_TMPDIR/stub:$PATH"

    # Read-only, enforced rather than trusted: a test that writes into a shared
    # home fails instead of changing what its siblings read.
    chmod -R a-w "$BATS_FILE_TMPDIR/home-clean" "$BATS_FILE_TMPDIR/home-copilot"
}

teardown_file() {
    chmod -R u+w "$BATS_FILE_TMPDIR/home-clean" "$BATS_FILE_TMPDIR/home-copilot" 2>/dev/null || true
}

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$COUNTING_SCRIPT"
    FAKEHOME="$(mktemp -d)"
    seed_copilot_base "$FAKEHOME"
    cd "$REPO" || exit 1
}

teardown() { cd / || true; rm -rf "$FAKEHOME"; }

# path_without_copilot: $PATH with every directory holding a `copilot` removed.
#
# The absence test used to assume the absence -- its comment read "this PATH has
# no copilot on it (true of this test suite's own environment already)", which is
# an assumption about the developer's machine written down as a fact. It is false
# on any box that installed Copilot: here it arrives via nvm at
# ~/.nvm/versions/node/*/bin/copilot, the gate correctly declines to skip, and
# the assertion fails on a clean tree (TEST-007, #1409). Measured: 1 failure with
# it on PATH, 0 with the directory removed.
#
# Filtering the entries that hold it, rather than replacing PATH wholesale, keeps
# every other tool the deploy needs reachable on machines that do not put them in
# /usr/bin. A loop, not a single removal, because a binary can appear more than
# once. Not named `path`: in zsh that identifier is tied to $PATH itself, and
# assigning a string to it wipes the command search path (see the prohibited
# pattern table in .claude/CLAUDE.md).
path_without_copilot() {
    _pwc_out=""
    _pwc_ifs="$IFS"; IFS=":"
    for _pwc_dir in $PATH; do
        [ -n "$_pwc_dir" ] || continue
        [ -x "$_pwc_dir/copilot" ] && continue
        _pwc_out="${_pwc_out:+$_pwc_out:}$_pwc_dir"
    done
    IFS="$_pwc_ifs"
    printf '%s' "$_pwc_out"
}

@test "AC8 smoke: /spec is discoverable for claude + opencode after deploy" {
    shared_home clean
    # claude: native skill dir carrying the spec SKILL.md (name: preserved)
    [ -f "$SHARED/.claude/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$SHARED/.claude/skills/spec/SKILL.md"
    # opencode: spec command file present (name: dropped — keyed off filename)
    [ -f "$SHARED/.config/opencode/commands/spec.md" ]
    refute_grep '^name:' "$SHARED/.config/opencode/commands/spec.md"
    # AC1: neither deployed path is a symlink
    [ ! -L "$SHARED/.claude/skills/spec" ]
    [ ! -L "$SHARED/.config/opencode/commands/spec.md" ]
}

@test "HARNESS-075: deployed claude skills drop paths: and neutral keys, ensuring unconditional discovery" {
    shared_home clean
    # Claude Code treats `paths:` frontmatter as a conditional skill (deferred until a matching path is touched).
    # Deployed skills must not carry `paths:` or other store-only metadata in their top-level frontmatter.
    [ -d "$SHARED/.claude/skills" ]
    for f in "$SHARED"/.claude/skills/*/SKILL.md; do
        [ -f "$f" ] || continue
        fm="$(awk '/^---[[:space:]]*$/{n++; next} n==1{print} n>=2{exit}' "$f")"
        run grep -nE '^(paths|keywords|requires|id|type|status|created|owner|targets):' <<<"$fm"
        [ "$status" -eq 1 ] || { printf 'store-only frontmatter left in %s:\n%s\n' "$f" "$output" >&2; return 1; }
    done
}

@test "SDD-011: deployed /spec carries the Agent-Side Activation Rule (claude + opencode)" {
    shared_home clean
    # The agent-side proactive trigger must survive the vault -> record -> render
    # chain; if the record is regenerated from a SKILL.md missing the section, the
    # proactive /spec proposal behavior silently regresses. Guard both renders.
    grep -q '^## Agent-Side Activation Rule' "$SHARED/.claude/skills/spec/SKILL.md"
    grep -q '^## Agent-Side Activation Rule' "$SHARED/.config/opencode/commands/spec.md"
}

@test "AC8 smoke: agy gets /spec as a native skill + a flat prompt" {
    shared_home clean
    [ -f "$SHARED/.gemini/skills/spec/SKILL.md" ]
    [ -f "$SHARED/.gemini/prompts/spec.md" ]
    refute_grep '^name:' "$SHARED/.gemini/prompts/spec.md"   # frontmatter stripped
}

@test "HANDOFF-001: /handoff deploys cross-agent (claude + opencode + agy) with its checklist" {
    shared_home clean
    [ -f "$SHARED/.claude/skills/handoff/SKILL.md" ]
    grep -q '^name: handoff' "$SHARED/.claude/skills/handoff/SKILL.md"
    [ -f "$SHARED/.config/opencode/commands/handoff.md" ]
    [ -f "$SHARED/.gemini/skills/handoff/SKILL.md" ]
    # the continuity-block checklist survives the vault -> record -> render chain
    grep -q '## Session Handoff' "$SHARED/.claude/skills/handoff/SKILL.md"
}

@test "AC8 smoke: a Claude-only skill is NOT exposed to opencode/agy" {
    shared_home clean
    # crystallize is targets:[claude] (agent-local auto-memory store)
    [ -f "$SHARED/.claude/skills/crystallize/SKILL.md" ]
    [ ! -f "$SHARED/.config/opencode/commands/crystallize.md" ]
    [ ! -d "$SHARED/.gemini/skills/crystallize" ]
}

@test "AI-022: pi gets /spec as a native skill (regular copy, not a symlink)" {
    shared_home clean
    [ -f "$SHARED/.pi/agent/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$SHARED/.pi/agent/skills/spec/SKILL.md"
    [ ! -L "$SHARED/.pi/agent/skills/spec" ]
}

@test "AI-022: a Claude-only skill is NOT exposed to pi" {
    shared_home clean
    # crystallize is targets:[claude] (agent-local auto-memory store)
    [ ! -d "$SHARED/.pi/agent/skills/crystallize" ]
}

@test "AI-022: deploy leaves pi-installed sibling symlinks alone" {
    # pi's own installer manages skills as symlinks into ~/.agents/skills; our
    # deploy must only de-symlink destinations it owns, never prune foreign links.
    mkdir -p "$FAKEHOME/.agents/skills/userskill" "$FAKEHOME/.pi/agent/skills"
    printf -- '---\nname: userskill\n---\nuser-installed\n' \
        > "$FAKEHOME/.agents/skills/userskill/SKILL.md"
    ln -s "$FAKEHOME/.agents/skills/userskill" "$FAKEHOME/.pi/agent/skills/userskill"
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -L "$FAKEHOME/.pi/agent/skills/userskill" ]
    [ -f "$FAKEHOME/.pi/agent/skills/userskill/SKILL.md" ]
}

@test "AC1 smoke: no deployed skill path is a symlink" {
    shared_home clean
    [ -z "$(find "$SHARED/.claude/skills" "$SHARED/.config/opencode/commands" "$SHARED/.gemini/skills" "$SHARED/.gemini/prompts" "$SHARED/.copilot/skills" -type l 2>/dev/null)" ]
}

@test "AC6 smoke: the copilot catalog lists /spec but not the Claude-only skill" {
    shared_home clean
    grep -qF -- '**spec**' "$SHARED/.copilot/copilot-instructions.md"
    refute_grep_fixed '**crystallize**' "$SHARED/.copilot/copilot-instructions.md"
}

@test "HARNESS-051: copilot gets native /spec and /handoff skills" {
    shared_home copilot
    [ -f "$SHARED/.copilot/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$SHARED/.copilot/skills/spec/SKILL.md"
    [ -f "$SHARED/.copilot/skills/handoff/SKILL.md" ]
    grep -q '^name: handoff' "$SHARED/.copilot/skills/handoff/SKILL.md"
    [ ! -L "$SHARED/.copilot/skills/spec" ]
    [ ! -L "$SHARED/.copilot/skills/handoff" ]
}

@test "HARNESS-051: copilot target filtering and auxiliary files are preserved" {
    shared_home copilot
    [ ! -d "$SHARED/.copilot/skills/crystallize" ]
    [ -f "$SHARED/.copilot/skills/systematic-debugging/root-cause-tracing.md" ]
}

@test "BUG-771: copilot native skills are not deployed when the copilot binary is absent" {
    # No copilot stub here, and the absence is MADE TRUE rather than assumed:
    # this is the class of box the gate exists for (setup-linux.sh never
    # auto-installs Copilot), but the developer running the suite may well have
    # it. Asserting "the gate fires when the binary is absent" against a PATH
    # that still carries the binary tests the opposite of the intent.
    nocopilot="$(path_without_copilot)"

    # Fail loudly if the fixture did not achieve the absence, instead of
    # quietly re-testing the present-binary path and reporting a pass.
    #
    # `run !` rather than `run` plus a status check: a bare `run` of a lookup
    # that is SUPPOSED to fail makes bats emit a BW01 "command not found"
    # warning on every suite run, and a warning nobody can act on is how real
    # ones stop being read. `run -127` would silence it by pinning an exact
    # code, which is worse -- `command -v` answers 1 or 127 depending on the sh.
    run ! env PATH="$nocopilot" sh -c 'command -v copilot'

    run env HOME="$FAKEHOME" PATH="$nocopilot" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ ! -e "$FAKEHOME/.copilot/skills" ]
    # The catalog injection is a separate, un-gated feature (it only edits an
    # already-present instructions.md) and must be unaffected.
    grep -qF -- '**spec**' "$FAKEHOME/.copilot/copilot-instructions.md"
}

@test "HARNESS-051: copilot deploy prunes only generated stale skills" {
    mkdir -p "$FAKEHOME/.copilot/skills/user-skill" "$FAKEHOME/.copilot/skills/stale-skill"
    printf -- '---\nname: user-skill\ndescription: user managed\n---\n' \
        > "$FAKEHOME/.copilot/skills/user-skill/SKILL.md"
    printf -- '---\ngenerated: true\nname: stale-skill\ndescription: stale generated\n---\n' \
        > "$FAKEHOME/.copilot/skills/stale-skill/SKILL.md"
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.copilot/skills/user-skill/SKILL.md" ]
    [ ! -d "$FAKEHOME/.copilot/skills/stale-skill" ]
}

# SKILL-001 review: a retired skill's name is not ours alone. Another tool put a
# skill named `audit` into the same directories, and a check that read only the
# name went red. Deploy removes what it rendered, which carries the vault source
# it came from, and a foreign skill of the same name survives it.
@test "SKILL-001: deploy prunes a retired skill it rendered and keeps a foreign one of the same name" {
    mkdir -p "$FAKEHOME/.claude/skills/audit" "$FAKEHOME/.pi/agent/skills/enrich-us" "$FAKEHOME/.gemini/prompts"
    printf -- '---\nname: audit\ndescription: another tool installed this\n---\nforeign\n' \
        > "$FAKEHOME/.claude/skills/audit/SKILL.md"
    printf -- '---\ngenerated: true\ngenerated_from: 00_meta/skills/enrich-us/SKILL.md\nname: enrich-us\n---\nours\n' \
        > "$FAKEHOME/.pi/agent/skills/enrich-us/SKILL.md"
    printf '<!-- generated: true; from: 00_meta/skills/writing-plans/SKILL.md; sha256:0 -->\nours\n' \
        > "$FAKEHOME/.gemini/prompts/writing-plans.md"
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    grep -q '^foreign$' "$FAKEHOME/.claude/skills/audit/SKILL.md"
    [ ! -e "$FAKEHOME/.pi/agent/skills/enrich-us" ]
    [ ! -e "$FAKEHOME/.gemini/prompts/writing-plans.md" ]
}

# HARNESS-056: the Definition of Done is doctrine, so it must reach EVERY
# surface, not only the ones whose instruction file happens to be convenient.
# These assert the real records, not a fixture.

@test "HARNESS-056: every enforced target injects the definition-of-done" {
    local missing=0 f
    while read -r f; do
        jq -e --arg f "$f" '.targets[] | select(.file==$f) | .inject | index("definition-of-done")' \
            harness/manifest.json >/dev/null || { echo "target does not inject it: $f"; missing=1; }
    done < <(jq -r '.targets[].file' harness/manifest.json)
    jq -e '.doctrine.inject | index("definition-of-done")' harness/manifest.json >/dev/null \
        || { echo "compact doctrine payload does not carry it"; missing=1; }
    [ "$missing" -eq 0 ]
}

@test "HARNESS-056: the injected region reaches the committed instruction files" {
    grep -q 'Working code is not a finished change' AGENTS.md
    grep -q 'Working code is not a finished change' ai/claude/CLAUDE.md
}

@test "HARNESS-056: the compact doctrine payload carries it and stays under its cap" {
    shared_home clean
    local f cap chars bytes
    while IFS=$'\t' read -r f cap; do
        [ -f "$SHARED/$f" ]
        grep -q 'Working code is not a finished change' "$SHARED/$f"
        chars="$(wc -m < "$SHARED/$f")"
        [ "$chars" -lt "$cap" ] || { echo "$f is $chars chars, at or over its $cap cap"; return 1; }

        # BOTH units, because asserting one was how this went unnoticed.
        #
        # The cap is documented in CHARACTERS and this test read `wc -m`, so on
        # 2026-09-05 it passed at 11974 while the file was 12047 BYTES — 47 over
        # the same cap. Nothing has verified which unit the platform counts, and
        # the documented failure mode is that the overflow is dropped SILENTLY.
        # A guard measuring one unit cannot report the other one crossing.
        #
        # deploy_doctrine now normalises typographic punctuation in a capped
        # payload, so the two measures track each other. This assertion is what
        # keeps that true: reintroduce a multi-byte character and the byte count
        # separates from the character count and lands here first.
        bytes="$(wc -c < "$SHARED/$f")"
        [ "$bytes" -lt "$cap" ] || { echo "$f is $bytes BYTES, at or over its $cap cap (chars: $chars)"; return 1; }
    done < <(jq -r '.doctrine.deploy[] | "\(.file)\t\(.char_cap)"' harness/manifest.json)
}

@test "HARNESS-056: the checklist binds the standing orders instead of restating them" {
    # a second source of truth is the failure mode this change exists to avoid
    # the closing pass moved into adversarial-review when SKILL-001 retired
    # verification-before-completion
    grep -q 'not a second source of truth' harness/skills/adversarial-review/SKILL.md
    grep -q '^## Closing pass (Definition of Done)' harness/skills/adversarial-review/SKILL.md
}

# HERMES-018: one frontmatter contract for the whole library, enforced by the
# engine rather than by convention. These assert the real records.

@test "HERMES-018: the schema requires the store's frontmatter law, not just name+description" {
    local k
    for k in name description id type status created owner; do
        jq -e --arg k "$k" '.required | index($k)' harness/skill-frontmatter.schema.json >/dev/null \
            || { echo "schema does not require: $k"; return 1; }
    done
}

@test "HERMES-018: every committed skill record satisfies the contract" {
    local d name missing=0 k
    for d in harness/skills/*/; do
        [ -f "$d/SKILL.md" ] || continue
        name="$(basename "$d")"
        for k in name description id type status created owner; do
            awk -v key="$k" '/^---[[:space:]]*$/{n++; next} n==1 && $0 ~ "^" key ": *[^ ]" {found=1} n>=2{exit} END{exit !found}' \
                "$d/SKILL.md" || { echo "$name is missing $k"; missing=1; }
        done
    done
    [ "$missing" -eq 0 ]
}

@test "HERMES-018: --check rejects a record that drops a required key" {
    local victim="harness/skills/test/SKILL.md" backup
    backup="$(mktemp)"
    cp "$victim" "$backup"
    sed -i '/^owner: /d' "$victim"
    run bash scripts/compile-harness.sh --check
    cp "$backup" "$victim"; rm -f "$backup"
    [ "$status" -ne 0 ]
    [[ "$output" == *"owner"* ]]
}

@test "HERMES-018: a vendored skill carries provenance and an attribution row" {
    local d name
    for d in harness/skills/*/; do
        [ -f "$d/SKILL.md" ] || continue
        grep -q '^source:' "$d/SKILL.md" || continue
        name="$(basename "$d")"
        grep -q '^license:' "$d/SKILL.md" || { echo "$name has source but no license"; return 1; }
        grep -qF "\`$name\`" harness/skills/ATTRIBUTION.md || { echo "$name is vendored but absent from ATTRIBUTION.md"; return 1; }
    done
}

# The shared homes are only safe if nothing writes to them.
@test "a write into a shared setup_file home fails" {
    [ "$(id -u)" -ne 0 ] || skip "root ignores the mode bits that keep the shared homes read-only"
    shared_home clean
    run touch "$SHARED/.written-by-a-test"
    [ "$status" -ne 0 ]
    # An append, not a touch: the owner may reset a timestamp without write access.
    run sh -c 'printf x >> "$1"' _ "$SHARED/.claude/skills/spec/SKILL.md"
    [ "$status" -ne 0 ]
}

# Last in the file on purpose: it reads the count every test above left. 17
# deploys per run was the baseline; the shared homes in setup_file bring it to 6.
@test "skills-pipeline deploys at most 6 times per run" {
    local n
    n="$(wc -l < "$DEPLOY_COUNT")"
    [ "$n" -le 6 ] || { echo "compile-harness.sh --deploy ran $n times, budget 6" >&2; return 1; }
}
