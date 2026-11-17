#!/usr/bin/env bats
# GUARD: the autopilot skill routes a run into a persistent lane only on the
# activation record, and never creates that record itself (#3426, #3432).
#
# `.dotf/lane.json` is the CLI-129 queue contract: a repository can carry it and
# run no lanes, so testing it sent runs down the lane branch on repositories
# that never opted in. The committed marker `.dotf/lanes.json` declares a
# repository but arms nothing: a machine runs lanes once `dotf converge --only
# lanes` writes the activation record into the git common dir, which is the fact
# the lane guard arms from (#3437), so the skill tests that record. An
# unattended run must not run `dotf lane init` to create it.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SKILL="$REPO/harness/skills/autopilot/SKILL.md"
    # The record's name is read from the Go constant, not restated here.
    RECORD="$(sed -n 's/^const ActivationFile = "\(.*\)"$/\1/p' "$REPO/cli/internal/lane/activate.go")"
}

@test "the activation record constant is readable from activate.go" {
    [ "$RECORD" = "dotf-lanes-active" ] || { printf 'got [%s]\n' "$RECORD" >&2; false; }
}

@test "autopilot decides on a lane by testing exactly the activation record in the git common dir" {
    [ -n "$RECORD" ] || { printf 'ActivationFile not read from activate.go\n' >&2; false; }
    cond="$(grep -E '^[[:space:]]*if \[ -f .*git-common-dir' "$SKILL")" || false
    [ -n "$cond" ] || false
    n="$(printf '%s\n' "$cond" | wc -l | tr -d ' ')"
    [ "$n" = "1" ] || { printf 'expected one lane condition, got %s\n' "$n" >&2; false; }
    [ "${cond#*"/$RECORD"}" != "$cond" ] || { printf 'lane condition does not name %s: %s\n' "$RECORD" "$cond" >&2; false; }
}

@test "autopilot tests no .dotf lane file in executable lines" {
    # Comment lines explain the difference; a command must not decide on either file.
    bad="$(grep -nE '\.dotf/lanes?\.json' "$SKILL" | grep -v ':[[:space:]]*#' || true)"
    [ -z "$bad" ] || { printf '%s\n' "$bad" >&2; false; }
}

@test "autopilot never runs dotf lane init" {
    if grep -v '^[[:space:]]*#' "$SKILL" | grep -qE 'dotf lane init'; then
        false
    fi
}

@test "autopilot still switches and claims the lane" {
    grep -q 'dotf lane switch "\$BRANCH"' "$SKILL" || false
    grep -q 'dotf lane claim "\$ISSUE_NUM"' "$SKILL" || false
}

@test "a failed lane switch parks as wip-cap and ends the run, never falling through to worktree add" {
    # The lane branch is the text between the record test and its `else`.
    branch="$(awk '/^[[:space:]]*if \[ -f .*git-common-dir/{f=1; next} f && /^[[:space:]]*else[[:space:]]*$/{exit} f' "$SKILL")"
    [ -n "$branch" ] || false
    # The switch's exit status is tested, and the failure arm names the park and exits.
    printf '%s\n' "$branch" | grep -qE '^[[:space:]]*if ! .*dotf lane switch "\$BRANCH"' || false
    printf '%s\n' "$branch" | grep -q 'parked(wip-cap)' || false
    printf '%s\n' "$branch" | grep -qE '^[[:space:]]*exit [0-9]+' || false
    # No worktree creation inside the lane branch.
    if printf '%s\n' "$branch" | grep -v '^[[:space:]]*#' | grep -q 'dotf worktree add'; then false; fi
}

@test "wip-cap is a declared run-level stop" {
    grep -q '`wip-cap`' "$SKILL" || false
}

@test "the wip-cap exit comes before the lease-lost path" {
    cap="$(grep -n 'parked(wip-cap)' "$SKILL" | grep -v ':[[:space:]]*#' | head -1 | cut -d: -f1)"
    lost="$(grep -n 'worktree resolution failed -> park(lease-lost)' "$SKILL" | head -1 | cut -d: -f1)"
    [ -n "$cap" ] && [ -n "$lost" ] || false
    [ "$cap" -lt "$lost" ] || false
}
