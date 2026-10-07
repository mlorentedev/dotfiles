#!/usr/bin/env bats
# scripts/run-bats.sh against the real bats and the real GNU parallel, on a
# fixture suite of its own. The stubbed cases (tests/run-bats.bats) prove what the
# script does when its environment is broken; these prove that a working
# environment is accepted and the flags it builds are ones bats survives.

# bats file_tags=os-sensitive

setup() {
    RUN="$BATS_TEST_DIRNAME/../scripts/run-bats.sh"
    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX"
    command -v parallel >/dev/null 2>&1 || skip "GNU parallel is not installed"
    # Fixtures are written with printf, never a heredoc: bats preprocesses every
    # line of THIS file that starts with `@test` or `# bats`, heredoc bodies
    # included, which would turn the fixture's tests and tag into this file's own.
    printf '# bats file_%s=fixture-tag\n@test "a one" { true; }\n@test "a two" { true; }\n' tags > "$FX/a.bats"
    printf '@test "b one" { true; }\n' > "$FX/b.bats"
}

# A nested bats must not inherit this run's BATS_* state, and bats puts its own
# libexec dir first on PATH, where a file also named `bats` is not the CLI: a
# nested `bats` found there dies with "bats_readlinkf: command not found".
nested() {
    local clean
    clean="$(printf '%s\n' "$PATH" | tr ':' '\n' | grep -vxF "${BATS_LIBEXEC:-/nonexistent}" | paste -sd: -)"
    env -i PATH="$clean" HOME="$HOME" "$RUN" "$@"
}

@test "run-bats: the parallelism is the CPU count getconf reports, as a positive integer" {
    run "$RUN" --print-jobs
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[1-9][0-9]*$ ]]
    [ "$output" = "$(getconf _NPROCESSORS_ONLN)" ]
}

@test "run-bats: runs a fixture suite in parallel and passes it" {
    run nested "$FX/a.bats" "$FX/b.bats"
    [ "$status" -eq 0 ]
    [[ "$output" == *"ok"*"a one"* ]]
    [[ "$output" == *"ok"*"b one"* ]]
}

@test "run-bats: --filter-tags runs only the tagged tests" {
    run nested --filter-tags fixture-tag "$FX/a.bats" "$FX/b.bats"
    [ "$status" -eq 0 ]
    [[ "$output" == *"a one"* ]]
    [[ "$output" != *"b one"* ]]
}

@test "run-bats: a tag no test carries fails the run instead of passing it empty" {
    run nested --filter-tags no-such-tag "$FX/a.bats" "$FX/b.bats"
    [ "$status" -ne 0 ]
    [[ "$output" == *"no test carries the tag"* ]]
}

@test "run-bats: --expect-bash fails when the bash on PATH is another major version" {
    run nested --expect-bash 2 "$FX/b.bats"
    [ "$status" -ne 0 ]
    [[ "$output" == *"expected major version 2"* ]]
}
