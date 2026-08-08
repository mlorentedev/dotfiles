#!/usr/bin/env bats
# BUG-055: the layer tests/precommit-fallback.bats structurally cannot provide.
#
# That suite drives a STUB `pre-commit`, by design — it makes the dispatcher's
# *invocation* assertable, which a real tool (reporting only pass/fail) never
# would. The cost is that a stub accepts any argument list, so "the command we
# built" and "a command the real tool survives" are different claims and only the
# first was ever tested. A fallback that omitted --hook-dir satisfied all 15 cases
# there and still aborted `git commit` in every repo on the machine.
#
# So these cases assert the only thing that cannot be faked: run the real
# pre-commit, through a real `git commit`, and check THE COMMIT EXISTS afterwards.
#
# CI installs pre-commit for this file (see .github/workflows/ci.yml). The skip
# below is for a developer box without it, never for CI — a real check that
# silently skips in CI is worth less than no check at all.

setup() {
    command -v pre-commit >/dev/null 2>&1 || skip "pre-commit not installed"
    REPO="$BATS_TEST_DIRNAME/.."
    DISPATCH="$REPO/git-hooks"
    WORK="$(mktemp -d)"
    FIXTURE="$WORK/repo"
    mkdir -p "$FIXTURE"
    git -C "$FIXTURE" init -q
    git -C "$FIXTURE" config user.email t@t.io
    git -C "$FIXTURE" config user.name tester
}

teardown() { [ -z "${WORK:-}" ] || rm -rf "$WORK"; }

# A repo pre-commit manages but never installed hooks into — the shape GUARD-001
# forces, because `pre-commit install` refuses outright while core.hooksPath is
# set ("Cowardly refusing to install hooks with core.hooksPath set"). Every stage
# therefore takes the dispatcher's fallback.
add_precommit_config() {
    cat > "$FIXTURE/.pre-commit-config.yaml" <<'YAML'
repos:
  - repo: local
    hooks:
      # Quoted: YAML parses a bare true/false as a bool and pre-commit rejects it
      # with InvalidConfigError. The stubbed suite never notices -- a stub reads no
      # config at all -- which is one more thing only a real run can tell us.
      - id: demo
        name: demo
        entry: "true"
        language: system
        stages: [pre-commit]
YAML
}

# Commit the way the box does: through the dispatcher, via core.hooksPath.
dispatched_commit() {
    git -C "$FIXTURE" -c core.hooksPath="$DISPATCH" commit -m "${1:-chore: test}"
}

@test "BUG-055: a real git commit succeeds with a config and no installed hooks" {
    add_precommit_config
    echo hello > "$FIXTURE/f.txt"
    git -C "$FIXTURE" add -A

    run dispatched_commit
    [ "$status" -eq 0 ]

    # The assertion that matters. The dispatcher exiting non-zero is only the
    # symptom; what the bug actually did was leave the repo with no commit, and
    # asserting on the hook's status alone would miss a stage that fails silently.
    run git -C "$FIXTURE" rev-parse --verify HEAD
    [ "$status" -eq 0 ]
}

@test "BUG-055: no stage of a commit crashes pre-commit (nothing on stderr about TypeError)" {
    # The original failure surfaced only as an unhandled TypeError from a stage
    # the repo had not installed — commit-msg, which no config here declares.
    # A stage with nothing to run must be a clean no-op, not a crash.
    add_precommit_config
    echo hello > "$FIXTURE/f.txt"
    git -C "$FIXTURE" add -A

    run dispatched_commit
    [[ "$output" != *"TypeError"* ]]
    [[ "$output" != *"An unexpected error has occurred"* ]]
}

@test "BUG-055: a failing hook still blocks the commit (the fix does not disarm the gate)" {
    # Red-team direction: a fallback that swallowed every status would satisfy the
    # green cases above by never gating anything. The gate has to still bite.
    cat > "$FIXTURE/.pre-commit-config.yaml" <<'YAML'
repos:
  - repo: local
    hooks:
      - id: always-fails
        name: always-fails
        entry: "false"
        language: system
        stages: [pre-commit]
YAML
    echo hello > "$FIXTURE/f.txt"
    git -C "$FIXTURE" add -A

    run dispatched_commit
    [ "$status" -ne 0 ]

    run git -C "$FIXTURE" rev-parse --verify HEAD
    [ "$status" -ne 0 ]
}
