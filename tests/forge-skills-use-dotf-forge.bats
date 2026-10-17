#!/usr/bin/env bats
# GUARD: the skills that write to the forge do it through `dotf forge`, and the
# Claude template allows exactly that surface (TOOL-030, AC7; ADR-052).
#
# A skill that tells an agent to run `gh issue comment --body "<prose>"` puts the
# prose on the command line, where a permission deny glob matches it (BUG-152,
# #2634). `dotf forge` takes the body from a file, so the command line carries no
# prose and one verb works on GitHub and Gitea alike.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SKILLS="$REPO/harness/skills"
}

@test "new-ticket creates its issue with dotf forge, body from a file" {
    grep -q 'dotf forge issue create' "$SKILLS/new-ticket/SKILL.md" || false
    if grep -q 'gh issue create --repo' "$SKILLS/new-ticket/SKILL.md"; then false; fi
}

@test "new-ticket passes its label flags as an array, which zsh does not word-split from a string" {
    grep -q 'LABEL_FLAGS=()' "$SKILLS/new-ticket/SKILL.md" || false
    if grep -qE 'LABEL_FLAGS=""|[^"{]\$LABEL_FLAGS' "$SKILLS/new-ticket/SKILL.md"; then false; fi
}

@test "autopilot assigns every temp file it hands to dotf forge" {
    for v in CLAIM_FILE PR_BODY_FILE PARK_FILE; do
        grep -q "$v=\$(mktemp)" "$SKILLS/autopilot/SKILL.md" || { printf '%s is used and never assigned\n' "$v" >&2; false; }
    done
}

@test "pr-review-triage posts the triage record with dotf forge" {
    grep -q 'dotf forge pr comment' "$SKILLS/pr-review-triage/SKILL.md" || false
    if grep -q 'gh pr comment' "$SKILLS/pr-review-triage/SKILL.md"; then false; fi
}

@test "autopilot claims, parks and opens its PR with dotf forge" {
    grep -q 'dotf forge issue comment' "$SKILLS/autopilot/SKILL.md" || false
    grep -q 'dotf forge pr create' "$SKILLS/autopilot/SKILL.md" || false
    if grep -qE 'gh (issue comment|pr create)' "$SKILLS/autopilot/SKILL.md"; then false; fi
}

@test "no forge-writing skill passes a body as a command-line argument" {
    for s in new-ticket pr-review-triage autopilot; do
        if grep -qE 'dotf forge [a-z]+ [a-z]+.* --body ' "$SKILLS/$s/SKILL.md"; then
            printf '%s passes --body inline\n' "$s" >&2
            false
        fi
    done
}

@test "the Claude template allows the dotf forge write verbs" {
    for rule in 'dotf forge issue:\*' 'dotf forge pr create:\*' 'dotf forge pr comment:\*' 'dotf forge pr edit:\*'; do
        grep -q "\"Bash($rule)\"" "$REPO/ai/claude/settings.json" || { printf 'missing allow: %s\n' "$rule" >&2; false; }
    done
}

@test "the Claude template allows no merge or close verb of dotf forge" {
    if grep -qE 'dotf forge[^"]*(merge|close)' "$REPO/ai/claude/settings.json"; then false; fi
}
