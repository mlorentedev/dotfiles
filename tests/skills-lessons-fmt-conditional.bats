#!/usr/bin/env bats
# GUARD: no skill tells an agent to run `dotf lessons fmt` unconditionally (DOCS-021, #2470).
#
# `dotf lessons fmt` quotes ids, adds titles and moves dates across the whole lessons tree. In a
# repository with its own lesson format and formatter (kubelab) it rewrote 577 files, so an agent
# that followed the skill to the letter opened a 577-file PR. Every mention in a skill record must
# therefore carry the condition: only where the repo uses dotf's lesson format.

setup() {
    SKILLS="$BATS_TEST_DIRNAME/../harness/skills"
}

@test "every skill line that names dotf lessons fmt carries the repo-format condition" {
    bad=$(grep -rn 'dotf lessons fmt' "$SKILLS" --include=SKILL.md \
        | grep -v "only where the repo uses dotf's" || true)
    if [ -n "$bad" ]; then
        printf 'unconditional dotf lessons fmt in a skill:\n%s\n' "$bad" >&2
        false
    fi
}

@test "the guard is not vacuous: at least one skill names dotf lessons fmt" {
    n=$(grep -rn 'dotf lessons fmt' "$SKILLS" --include=SKILL.md | wc -l | tr -d ' ')
    [ "$n" -ge 1 ] || false
}
