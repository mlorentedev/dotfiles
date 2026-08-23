#!/usr/bin/env bats
# SDD-008 skill pipeline smoke test (AC8). Renders the committed skill records
# (harness/skills/) to a throwaway HOME and asserts the agent-native discovery
# locations are populated -- the vault -> record -> render -> deploy -> discovery
# round-trip at the filesystem level. Runs offline (no vault); the integration
# container exercises the full setup-linux.sh end-to-end.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/compile-harness.sh"
    FAKEHOME="$(mktemp -d)"
    # copilot base file so catalog injection has a marked target
    mkdir -p "$FAKEHOME/.copilot"
    printf '## Skills\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n' \
        > "$FAKEHOME/.copilot/copilot-instructions.md"
    cd "$REPO" || exit 1
}

teardown() { cd / || true; rm -rf "$FAKEHOME"; }

# The copilot skill target has a manifest-declared requires_command (BUG-771:
# native skills must not create ~/.copilot on a box that never installed
# Copilot). Tests that exercise the deploy itself need a fake `copilot` on
# PATH; tests proving the gate need PATH left alone.
stub_copilot() {
    mkdir -p "$FAKEHOME/stub"
    printf '#!/usr/bin/env bash\nexit 0\n' > "$FAKEHOME/stub/copilot"
    chmod +x "$FAKEHOME/stub/copilot"
}

@test "AC8 smoke: /spec is discoverable for claude + opencode after deploy" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # claude: native skill dir carrying the spec SKILL.md (name: preserved)
    [ -f "$FAKEHOME/.claude/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$FAKEHOME/.claude/skills/spec/SKILL.md"
    # opencode: spec command file present (name: dropped — keyed off filename)
    [ -f "$FAKEHOME/.config/opencode/commands/spec.md" ]
    refute_grep '^name:' "$FAKEHOME/.config/opencode/commands/spec.md"
    # AC1: neither deployed path is a symlink
    [ ! -L "$FAKEHOME/.claude/skills/spec" ]
    [ ! -L "$FAKEHOME/.config/opencode/commands/spec.md" ]
}

@test "HARNESS-075: deployed claude skills drop paths: and neutral keys, ensuring unconditional discovery" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # Claude Code treats `paths:` frontmatter as a conditional skill (deferred until a matching path is touched).
    # Deployed skills must not carry `paths:` or other store-only metadata in their top-level frontmatter.
    [ -d "$FAKEHOME/.claude/skills" ]
    for f in "$FAKEHOME"/.claude/skills/*/SKILL.md; do
        [ -f "$f" ] || continue
        fm="$(awk '/^---[[:space:]]*$/{n++; next} n==1{print} n>=2{exit}' "$f")"
        run grep -nE '^(paths|keywords|requires|id|type|status|created|owner|targets):' <<<"$fm"
        [ "$status" -eq 1 ] || { printf 'store-only frontmatter left in %s:\n%s\n' "$f" "$output" >&2; return 1; }
    done
}

@test "SDD-011: deployed /spec carries the Agent-Side Activation Rule (claude + opencode)" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # The agent-side proactive trigger must survive the vault -> record -> render
    # chain; if the record is regenerated from a SKILL.md missing the section, the
    # proactive /spec proposal behavior silently regresses. Guard both renders.
    grep -q '^## Agent-Side Activation Rule' "$FAKEHOME/.claude/skills/spec/SKILL.md"
    grep -q '^## Agent-Side Activation Rule' "$FAKEHOME/.config/opencode/commands/spec.md"
}

@test "AC8 smoke: agy gets /spec as a native skill + a flat prompt" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.gemini/skills/spec/SKILL.md" ]
    [ -f "$FAKEHOME/.gemini/prompts/spec.md" ]
    refute_grep '^name:' "$FAKEHOME/.gemini/prompts/spec.md"   # frontmatter stripped
}

@test "HANDOFF-001: /handoff deploys cross-agent (claude + opencode + agy) with its checklist" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.claude/skills/handoff/SKILL.md" ]
    grep -q '^name: handoff' "$FAKEHOME/.claude/skills/handoff/SKILL.md"
    [ -f "$FAKEHOME/.config/opencode/commands/handoff.md" ]
    [ -f "$FAKEHOME/.gemini/skills/handoff/SKILL.md" ]
    # the continuity-block checklist survives the vault -> record -> render chain
    grep -q '## Session Handoff' "$FAKEHOME/.claude/skills/handoff/SKILL.md"
}

@test "AC8 smoke: a Claude-only skill is NOT exposed to opencode/agy" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # crystallize is targets:[claude] (agent-local auto-memory store)
    [ -f "$FAKEHOME/.claude/skills/crystallize/SKILL.md" ]
    [ ! -f "$FAKEHOME/.config/opencode/commands/crystallize.md" ]
    [ ! -d "$FAKEHOME/.gemini/skills/crystallize" ]
}

@test "AI-022: pi gets /spec as a native skill (regular copy, not a symlink)" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.pi/agent/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$FAKEHOME/.pi/agent/skills/spec/SKILL.md"
    [ ! -L "$FAKEHOME/.pi/agent/skills/spec" ]
}

@test "AI-022: a Claude-only skill is NOT exposed to pi" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # crystallize is targets:[claude] (agent-local auto-memory store)
    [ ! -d "$FAKEHOME/.pi/agent/skills/crystallize" ]
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
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -z "$(find "$FAKEHOME/.claude/skills" "$FAKEHOME/.config/opencode/commands" "$FAKEHOME/.gemini/skills" "$FAKEHOME/.gemini/prompts" "$FAKEHOME/.copilot/skills" -type l 2>/dev/null)" ]
}

@test "AC6 smoke: the copilot catalog lists /spec but not the Claude-only skill" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    grep -qF -- '**spec**' "$FAKEHOME/.copilot/copilot-instructions.md"
    refute_grep_fixed '**crystallize**' "$FAKEHOME/.copilot/copilot-instructions.md"
}

@test "HARNESS-051: copilot gets native /spec and /handoff skills" {
    stub_copilot
    run env HOME="$FAKEHOME" PATH="$FAKEHOME/stub:$PATH" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.copilot/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$FAKEHOME/.copilot/skills/spec/SKILL.md"
    [ -f "$FAKEHOME/.copilot/skills/handoff/SKILL.md" ]
    grep -q '^name: handoff' "$FAKEHOME/.copilot/skills/handoff/SKILL.md"
    [ ! -L "$FAKEHOME/.copilot/skills/spec" ]
    [ ! -L "$FAKEHOME/.copilot/skills/handoff" ]
}

@test "HARNESS-051: copilot target filtering and auxiliary files are preserved" {
    stub_copilot
    run env HOME="$FAKEHOME" PATH="$FAKEHOME/stub:$PATH" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ ! -d "$FAKEHOME/.copilot/skills/crystallize" ]
    [ -f "$FAKEHOME/.copilot/skills/systematic-debugging/root-cause-tracing.md" ]
}

@test "BUG-771: copilot native skills are not deployed when the copilot binary is absent" {
    # No stub_copilot here -- this PATH has no copilot on it (true of this
    # test suite's own environment already, which is exactly the class of
    # box the gate exists for: setup-linux.sh never auto-installs Copilot).
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
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
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    local f cap chars
    while IFS=$'\t' read -r f cap; do
        [ -f "$FAKEHOME/$f" ]
        grep -q 'Working code is not a finished change' "$FAKEHOME/$f"
        chars="$(wc -m < "$FAKEHOME/$f")"
        [ "$chars" -lt "$cap" ] || { echo "$f is $chars chars, at or over its $cap cap"; return 1; }
    done < <(jq -r '.doctrine.deploy[] | "\(.file)\t\(.char_cap)"' harness/manifest.json)
}

@test "HARNESS-056: the checklist binds the standing orders instead of restating them" {
    # a second source of truth is the failure mode this change exists to avoid
    grep -q 'not a second source of truth' harness/skills/verification-before-completion/SKILL.md
    grep -q 'The Closing Pass' harness/skills/verification-before-completion/SKILL.md
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
    local victim="harness/skills/audit/SKILL.md" backup
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
