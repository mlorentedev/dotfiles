#!/usr/bin/env bats
#
# HARNESS-112. The adversarial-review skill states its verdict rule TWICE — once
# in the Reality classification and once in the Verdict list — and the two
# drifted apart.
#
#   Reality rule : "weight each finding by severity x reality. A REAL
#                   Blocker/Major forces FAIL"
#   Verdict list : "FAIL - at least one blocker or major"   <- reality dropped
#
# A reviewer following the second returns FAIL on a Major it has itself labelled
# THEORETICAL. Measured on BUG-093 round 4: a race-window narrowing with no
# demonstrated exploit cost a full merge-and-re-review cycle.
#
# These assert the two statements still agree. The contradiction was invisible
# to every existing check because both halves are individually reasonable
# English.

load 'lib/refute'

setup() {
    REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
    SKILL="$REPO_ROOT/harness/skills/adversarial-review/SKILL.md"
}

@test "guard: the adversarial-review skill exists where the gate expects it" {
    [ -f "$SKILL" ]
}

@test "guard: the verdict rule qualifies Major by reality, not severity alone" {
    # The FAIL line must name REAL. Without it, the list contradicts the Reality
    # section above it and a reviewer can follow either.
    run grep -n 'FAIL.*at least one' "$SKILL"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q 'REAL'
}

@test "guard: the unqualified 'any blocker or major' rule is gone" {
    # The exact phrasing that produced the contradiction. Fixed-string, and via
    # refute_grep_fixed because a bare `! grep` that is not the last line of a
    # @test cannot fail it (#1034).
    refute_grep_fixed 'at least one blocker or major OR rubric' "$SKILL"
}

@test "guard: PASS WITH GAPS is reachable for a THEORETICAL-only Major" {
    run grep -c 'THEORETICAL' "$SKILL"
    [ "$status" -eq 0 ]
    [ "$output" -ge 2 ]
    # The verdict list itself must offer the non-blocking route, or the Reality
    # tag has nowhere to land.
    run grep -n 'PASS WITH GAPS' "$SKILL"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -qi 'THEORETICAL'
}

@test "guard: a Blocker still blocks regardless of its reality tag" {
    # The softening is deliberately scoped to Majors. If this line goes, the
    # gate has been widened past what HARNESS-112 argued for.
    run grep -n 'Blocker' "$SKILL"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q 'any reality'
}

@test "guard: the Major severity definition defers to reality, like the verdict list" {
    # A THIRD statement of the same rule, one section above the verdict list, and
    # it drifted the same way: while the list sent a THEORETICAL Major to PASS
    # WITH GAPS, this definition demanded a fix and a re-review for every Major.
    # The tests above could not see it — they only read the verdict list. Found
    # by review on #1543, which is why the guard now reads both halves.
    run grep -n '^- \*\*Major\*\*:' "$SKILL"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q 'REAL'
    printf '%s\n' "$output" | grep -q 'THEORETICAL'
}

@test "guard: the Reality classification the verdict defers to is still present" {
    run grep -q 'severity . reality' "$SKILL"
    [ "$status" -eq 0 ]
}
