#!/usr/bin/env bats
# SDD-008 skill pipeline smoke test (AC8). Renders the committed skill records
# (harness/skills/) to a throwaway HOME and asserts the agent-native discovery
# locations are populated -- the vault -> record -> render -> deploy -> discovery
# round-trip at the filesystem level. Runs offline (no vault); the integration
# container exercises the full setup-linux.sh end-to-end.

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

@test "AC8 smoke: /spec is discoverable for claude + opencode after deploy" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # claude: native skill dir carrying the spec SKILL.md (name: preserved)
    [ -f "$FAKEHOME/.claude/skills/spec/SKILL.md" ]
    grep -q '^name: spec' "$FAKEHOME/.claude/skills/spec/SKILL.md"
    # opencode: spec command file present (name: dropped — keyed off filename)
    [ -f "$FAKEHOME/.config/opencode/commands/spec.md" ]
    ! grep -q '^name:' "$FAKEHOME/.config/opencode/commands/spec.md"
    # AC1: neither deployed path is a symlink
    [ ! -L "$FAKEHOME/.claude/skills/spec" ]
    [ ! -L "$FAKEHOME/.config/opencode/commands/spec.md" ]
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
    ! grep -q '^name:' "$FAKEHOME/.gemini/prompts/spec.md"   # frontmatter stripped
}

@test "AC8 smoke: a Claude-only skill is NOT exposed to opencode/agy" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    # creating-skills is targets:[claude]
    [ -f "$FAKEHOME/.claude/skills/creating-skills/SKILL.md" ]
    [ ! -f "$FAKEHOME/.config/opencode/commands/creating-skills.md" ]
    [ ! -d "$FAKEHOME/.gemini/skills/creating-skills" ]
}

@test "AC1 smoke: no deployed skill path is a symlink" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [ -z "$(find "$FAKEHOME/.claude/skills" "$FAKEHOME/.config/opencode/commands" "$FAKEHOME/.gemini/skills" "$FAKEHOME/.gemini/prompts" -type l 2>/dev/null)" ]
}

@test "AC6 smoke: the copilot catalog lists /spec but not the Claude-only skill" {
    run env HOME="$FAKEHOME" "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    grep -qF -- '**spec**' "$FAKEHOME/.copilot/copilot-instructions.md"
    ! grep -q 'creating-skills' "$FAKEHOME/.copilot/copilot-instructions.md"
}
