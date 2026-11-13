#!/usr/bin/env bats
# GUARD: the autopilot skill routes a run into a persistent lane only on the
# ADR-060 marker, and never creates that marker itself (#3426).
#
# `.dotf/lane.json` is the CLI-129 queue contract: a repository can carry it and
# run no lanes, so testing it sent runs down the lane branch on repositories
# that never opted in. `dotf lane init` writes the marker into the main
# checkout, which arms the lane guard for every worktree on the machine, so an
# unattended run must not run it.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SKILL="$REPO/harness/skills/autopilot/SKILL.md"
    # The marker is read from the Go constant, not restated here.
    MARKER="$(sed -n 's/^const LanesMarkerFile = "\(.*\)"$/\1/p' "$REPO/cli/internal/lane/named.go")"
}

@test "the lane marker constant is readable from named.go" {
    [ "$MARKER" = ".dotf/lanes.json" ] || { printf 'got [%s]\n' "$MARKER" >&2; false; }
}

@test "autopilot decides on a lane by testing exactly LanesMarkerFile" {
    cond="$(grep -E '^[[:space:]]*if \[ -[a-z] [^]]*\.dotf/lanes?\.json' "$SKILL")" || false
    [ -n "$cond" ] || false
    n="$(printf '%s\n' "$cond" | wc -l | tr -d ' ')"
    [ "$n" = "1" ] || { printf 'expected one lane condition, got %s\n' "$n" >&2; false; }
    [ "${cond#*"$MARKER"}" != "$cond" ] || { printf 'lane condition does not name %s: %s\n' "$MARKER" "$cond" >&2; false; }
}

@test "autopilot names no .dotf lane marker other than LanesMarkerFile in executable lines" {
    # Comment lines may explain the difference; a command must not name the other file.
    bad="$(grep -nE '\.dotf/lanes?\.json' "$SKILL" | grep -v ':[[:space:]]*#' | grep -vF "$MARKER" || true)"
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
