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

@test "exits 0 when diff >= threshold AND a substantive spec folder is touched" {
    mkdir -p specs/SDD-999-test
    printf 'proposal line %d\n' {1..12} > specs/SDD-999-test/proposal.md
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change with spec"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"spec folder touched"* ]] || [[ "$output" == *"OK"* ]]
}

@test "a trivial (sub-floor) active-spec touch does NOT satisfy the gate (#686/C25)" {
    mkdir -p specs/SDD-777-stale
    printf 'x\n' > specs/SDD-777-stale/proposal.md   # 1 line, below SPEC_FLOOR
    printf 'line %d\n' {1..60} > big.txt
    _commit "large PR with a trivial stale-spec alibi"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
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

@test "exits 0 when dependencies label present AND author is a bot (#686/C25)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "dep bump"
    run env SDD_LABELS="dependencies" SDD_PR_BODY="" SDD_PR_AUTHOR="dependabot[bot]" \
        "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"bot-authored"* ]]
}

@test "dependencies label from a NON-bot author does not bypass the gate (#686/C25)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "dep bump masquerade"
    run env SDD_LABELS="dependencies" SDD_PR_BODY="" SDD_PR_AUTHOR="some-human" \
        "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}

@test "fails closed (exit 2) when the base ref does not resolve (#686/C3)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "change against a bogus base"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref origin/does-not-exist --head-ref feature
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not be resolved"* ]] || [[ "$output" == *"fails closed"* ]]
}

@test "a hand-written *generated* path is counted, not excluded (#686/C25)" {
    # Previously any path merely containing "generated" was silently excluded.
    printf 'line %d\n' {1..60} > internal_generated_names.go
    _commit "large generated-named file, no spec"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}

@test "excludes tests/ from LOC count" {
    mkdir -p tests
    printf 'line %d\n' {1..200} > tests/big-test.bats
    printf 'line %d\n' {1..10} > small.txt
    _commit "tests do not count"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "excludes Go *_test.go from LOC count (#517)" {
    mkdir -p cli/internal/foo
    printf 'line %d\n' {1..200} > cli/internal/foo/foo_test.go
    printf 'line %d\n' {1..10} > small.txt
    _commit "go test file does not count as production"
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

@test "excludes docs/**/*.md from LOC count (doc-only ADR/lesson/runbook)" {
    mkdir -p docs/adr
    printf 'line %d\n' {1..200} > docs/adr/adr-999-example.md
    _commit "doc-only ADR"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "does NOT exclude non-markdown under docs/ (a script still counts)" {
    mkdir -p docs
    printf 'line %d\n' {1..60} > docs/example.sh
    _commit "script under docs counts as production"
    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
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

# Regression for #397: git compresses a spec-archive rename into
# "specs/{ => archive}/<id>/proposal.md". The raw compressed path starts with
# "specs/{", dodging the specs/archive/* exclusion and falsely reading as an
# active-spec touch. A PR bundling that archive move with >=50 LOC of production
# code and NO genuine active spec must still FAIL the gate.
@test "spec-archive rename bundled with production code does not evade the gate (#397)" {
    # Put the spec on main so the move is a real rename across main...feature.
    git checkout -q main
    mkdir -p specs/HARNESS-026-foo
    printf 'proposal line %d\n' {1..30} > specs/HARNESS-026-foo/proposal.md
    git add -A && git commit -q -m "add spec on main"
    git checkout -q feature
    git merge -q main

    mkdir -p specs/archive/HARNESS-026-foo
    git mv specs/HARNESS-026-foo/proposal.md specs/archive/HARNESS-026-foo/proposal.md
    printf 'code line %d\n' {1..60} > feature.go
    _commit "archive move bundled with unrelated production code"

    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}

# The compressed rename must normalize so a PURE archive move still passes — but
# because it is below threshold (archived files excluded), NOT because of a
# phantom active-spec touch. --explain proves "Spec folder touched: no".
@test "pure spec-archive rename passes for the right reason: no false active-spec touch (#397)" {
    git checkout -q main
    mkdir -p specs/HARNESS-026-bar
    printf 'proposal line %d\n' {1..30} > specs/HARNESS-026-bar/proposal.md
    git add -A && git commit -q -m "add spec on main"
    git checkout -q feature
    git merge -q main

    mkdir -p specs/archive/HARNESS-026-bar
    git mv specs/HARNESS-026-bar/proposal.md specs/archive/HARNESS-026-bar/proposal.md
    _commit "pure archive move"

    run "$SCRIPTS_DIR/check-spec-gate.sh" --base-ref main --head-ref feature --explain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Spec folder touched: no"* ]]
    [[ "$output" == *"Production LOC (added+removed): 0"* ]]
}
