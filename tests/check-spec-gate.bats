#!/usr/bin/env bats
# Tests for scripts/check-spec-gate.sh — SDD Tier 4 enforcement gate

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_FIXTURE="/tmp/bats_specgate_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$REPO_FIXTURE"
    cd "$REPO_FIXTURE" || exit 1
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    echo "seed" > seed.txt
    git add seed.txt
    git commit -q -m "seed"
    git checkout -q -b feature
}

teardown() {
    cd / || true
    rm -rf "$REPO_FIXTURE"
}

_commit() {
    git add -A
    git commit -q -m "${1:-change}"
}

@test "check-spec-gate.sh --help shows usage and exits 0" {
    run "$SCRIPTS_DIR/check-spec-gate.sh" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"--base-ref"* ]]
}

@test "exits 2 when --base-ref missing" {
    run "$SCRIPTS_DIR/check-spec-gate.sh" --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"required"* ]]
}

@test "exits 2 when --head-ref missing" {
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main
    [ "$status" -eq 2 ]
    [[ "$output" == *"required"* ]]
}

@test "exits 2 on unknown argument" {
    run "$SCRIPTS_DIR/check-spec-gate.sh" --bogus
    [ "$status" -eq 2 ]
    [[ "$output" == *"Unknown"* ]]
}

@test "exits 0 when diff is below threshold (no spec needed)" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small change"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"below threshold"* ]] || [[ "$output" == *"OK"* ]]
}

@test "exits 0 when diff >= threshold AND specs folder is touched" {
    mkdir -p specs/SDD-999-test
    printf 'placeholder proposal\n' > specs/SDD-999-test/proposal.md
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change with spec"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"spec folder touched"* ]] || [[ "$output" == *"OK"* ]]
}

@test "exits 1 when diff >= threshold AND no specs folder" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change without spec"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
    [[ "$output" == *"AGENTS.md"* ]]
}

@test "exits 0 with skip-sdd label AND non-empty rationale" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change skipped"
    run env SDD_LABELS="skip-sdd" \
        SDD_PR_BODY=$'## Summary\nfoo\n\n## SDD skip rationale\nmechanical rename, no logic change.\n' \
        "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"skip-sdd"* ]]
}

@test "exits 1 when skip-sdd label present but rationale empty" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "skip without rationale"
    run env SDD_LABELS="skip-sdd" \
        SDD_PR_BODY=$'## Summary\nfoo\n\n## SDD skip rationale\n\n' \
        "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"rationale"* ]]
}

@test "exits 0 when dependencies label present (dependabot bypass)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "dep bump"
    run env SDD_LABELS="dependencies" SDD_PR_BODY="" \
        "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"dependencies"* ]]
}

@test "excludes tests/ from LOC count" {
    mkdir -p tests
    printf 'line %d\n' {1..200} > tests/big-test.bats
    printf 'line %d\n' {1..10} > small.txt
    _commit "tests do not count"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "excludes specs/archive/ from LOC count" {
    mkdir -p specs/archive/OLD-001-archived
    printf 'line %d\n' {1..200} > specs/archive/OLD-001-archived/proposal.md
    _commit "archive moves"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "excludes lockfiles from LOC count" {
    printf 'line %d\n' {1..200} > package-lock.json
    _commit "lockfile bump"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "--explain prints LOC breakdown" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature --explain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Threshold"* ]]
    [[ "$output" == *"Production LOC"* ]]
}

@test "--threshold flag overrides default" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature --threshold 5
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}

@test "specs/archive/ is not counted as a valid spec folder for the gate" {
    mkdir -p specs/archive/OLD-001-archived
    printf 'archived\n' > specs/archive/OLD-001-archived/proposal.md
    printf 'line %d\n' {1..60} > big.txt
    _commit "archive does not satisfy gate"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}
