#!/usr/bin/env bats
# Tests for scripts/check-spec-gate.sh — SDD Tier 4 enforcement gate

load 'lib/both-shells'
load 'lib/fixture-template'

# Built once per file and copied per test (CI-012 N3).
_seed_fixture() {
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    echo "seed" > seed.txt
    git add seed.txt
    git commit -q -m "seed"
    git checkout -q -b feature
}

setup_file() { fixture_template_build repo _seed_fixture; }

teardown_file() { fixture_template_release; }

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_FIXTURE="/tmp/bats_specgate_$$_${BATS_TEST_NUMBER:-0}"
    rm -rf "$REPO_FIXTURE"
    fixture_template_copy repo "$REPO_FIXTURE"
    cd "$REPO_FIXTURE" || exit 1
}

teardown() {
    cd / || true
    rm -rf "$REPO_FIXTURE"
}

_commit() {
    git add -A
    git commit -q -m "${1:-change}"
}

# Every case runs the gate under zsh and bash, which must agree.
_both() { run_both_shells "$SCRIPTS_DIR/check-spec-gate.sh" "$@"; }

@test "check-spec-gate.sh --help shows usage and exits 0" {
    _both --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]] || false
    [[ "$output" == *"--base-ref"* ]] || false
}

@test "exits 2 when --base-ref missing" {
    _both --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"required"* ]] || false
}

@test "exits 2 when --head-ref missing" {
    _both --base-ref main
    [ "$status" -eq 2 ]
    [[ "$output" == *"required"* ]] || false
}

@test "exits 2 on unknown argument" {
    _both --bogus
    [ "$status" -eq 2 ]
    [[ "$output" == *"Unknown"* ]] || false
}

@test "exits 0 when diff is below threshold (no spec needed)" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small change"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"below threshold"* ]] || [[ "$output" == *"OK"* ]] || false
}

@test "exits 0 when diff >= threshold AND a substantive spec folder is touched" {
    mkdir -p specs/SDD-999-test
    printf 'proposal line %d\n' {1..12} > specs/SDD-999-test/proposal.md
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change with spec"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"spec folder touched"* ]] || [[ "$output" == *"OK"* ]] || false
}

@test "a trivial (sub-floor) active-spec touch does NOT satisfy the gate (#686/C25)" {
    mkdir -p specs/SDD-777-stale
    printf 'x\n' > specs/SDD-777-stale/proposal.md   # 1 line, below SPEC_FLOOR
    printf 'line %d\n' {1..60} > big.txt
    _commit "large PR with a trivial stale-spec alibi"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "exits 1 when diff >= threshold AND no specs folder" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change without spec"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
    [[ "$output" == *"AGENTS.md"* ]] || false
}

@test "exits 0 with skip-sdd label AND non-empty rationale" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "big change skipped"
    SDD_LABELS="skip-sdd" \
        SDD_PR_BODY=$'## Summary\nfoo\n\n## SDD skip rationale\nmechanical rename, no logic change.\n' \
        _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"skip-sdd"* ]] || false
}

@test "exits 1 when skip-sdd label present but rationale empty" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "skip without rationale"
    SDD_LABELS="skip-sdd" \
        SDD_PR_BODY=$'## Summary\nfoo\n\n## SDD skip rationale\n\n' \
        _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"rationale"* ]] || false
}

@test "exits 0 when dependencies label present AND author is a bot (#686/C25)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "dep bump"
    SDD_LABELS="dependencies" SDD_PR_BODY="" SDD_PR_AUTHOR="dependabot[bot]" \
        _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"bot-authored"* ]] || false
}

@test "dependencies label from a NON-bot author does not bypass the gate (#686/C25)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "dep bump masquerade"
    SDD_LABELS="dependencies" SDD_PR_BODY="" SDD_PR_AUTHOR="some-human" \
        _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "fails closed (exit 2) when the base ref does not resolve (#686/C3)" {
    printf 'line %d\n' {1..60} > big.txt
    _commit "change against a bogus base"
    _both --base-ref origin/does-not-exist --head-ref feature
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not be resolved"* ]] || [[ "$output" == *"fails closed"* ]] || false
}

@test "a hand-written *generated* path is counted, not excluded (#686/C25)" {
    # Previously any path merely containing "generated" was silently excluded.
    printf 'line %d\n' {1..60} > internal_generated_names.go
    _commit "large generated-named file, no spec"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "excludes tests/ from LOC count" {
    mkdir -p tests
    printf 'line %d\n' {1..200} > tests/big-test.bats
    printf 'line %d\n' {1..10} > small.txt
    _commit "tests do not count"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "excludes Go *_test.go from LOC count (#517)" {
    mkdir -p cli/internal/foo
    printf 'line %d\n' {1..200} > cli/internal/foo/foo_test.go
    printf 'line %d\n' {1..10} > small.txt
    _commit "go test file does not count as production"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "excludes specs/archive/ from LOC count" {
    mkdir -p specs/archive/OLD-001-archived
    printf 'line %d\n' {1..200} > specs/archive/OLD-001-archived/proposal.md
    _commit "archive moves"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

# #2698: an autopilot run branch has to push its disarm report, which is prose of
# the same class as docs/*.md. Only the report is excluded, never the envelope:
# .agent/unattended.yml is the run's authority file (a PR touching it is never
# merged), so it stays counted. Not a broad .agent/** glob (#686/C25).
@test "excludes an autopilot run report from LOC count (#2698)" {
    mkdir -p .agent/runs/run-20261017T0807Z
    printf 'line %d\n' {1..120} > .agent/runs/run-20261017T0807Z/report.md
    _commit "run report only"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "counts a report nested deeper than .agent/runs/<id>/ (#2698)" {
    # `*` in a case pattern also matches `/`, so the exclusion must be pinned to
    # exactly one directory component (pr-agent/CodeRabbit on #2730).
    mkdir -p .agent/runs/run-1/nested
    printf 'line %d\n' {1..120} > .agent/runs/run-1/nested/report.md
    _commit "nested report"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "still counts the run envelope and other files under .agent/ (#2698)" {
    mkdir -p .agent/runs/run-20261017T0807Z
    printf 'line %d\n' {1..60} > .agent/unattended.yml
    printf 'line %d\n' {1..60} > .agent/runs/run-20261017T0807Z/notes.sh
    _commit "envelope and a script under .agent"
    _both --base-ref main --head-ref feature --explain
    [ "$status" -eq 1 ]
    [[ "$output" == *".agent/unattended.yml"* ]] || false
    [[ "$output" == *".agent/runs/run-20261017T0807Z/notes.sh"* ]] || false
}

@test "excludes docs/**/*.md from LOC count (doc-only ADR/lesson/runbook)" {
    mkdir -p docs/adr
    printf 'line %d\n' {1..200} > docs/adr/adr-999-example.md
    _commit "doc-only ADR"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "does NOT exclude non-markdown under docs/ (a script still counts)" {
    mkdir -p docs
    printf 'line %d\n' {1..60} > docs/example.sh
    _commit "script under docs counts as production"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "excludes lockfiles from LOC count" {
    printf 'line %d\n' {1..200} > package-lock.json
    _commit "lockfile bump"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

# #1904: #1898 deleted a stray 407-line diff.patch at the root, and the gate
# counted it as production code. Patch-tool output (.patch, .diff, and the
# .orig/.rej a failed apply leaves behind) is never source, whether added or
# removed.
@test "excludes patch-tool output, added or deleted, from LOC count (#1904)" {
    printf 'line %d\n' {1..200} > stray.patch
    printf 'line %d\n' {1..200} > stray.diff
    git add stray.patch stray.diff
    git commit -q -m "stray patches on main"
    git branch -f main HEAD
    git rm -q stray.patch stray.diff
    printf 'orig %d\n' {1..200} > leftover.go.orig
    printf 'rej %d\n' {1..200} > leftover.go.rej
    _commit "delete the patches, leave a failed apply's output"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
}

@test "deleting a source file still counts as production LOC (#1904 stays narrow)" {
    printf 'line %d\n' {1..200} > tool.sh
    git add tool.sh
    git commit -q -m "a script on main"
    git branch -f main HEAD
    git rm -q tool.sh
    _commit "delete the script"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "--explain prints LOC breakdown" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small"
    _both --base-ref main --head-ref feature --explain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Threshold"* ]] || false
    [[ "$output" == *"Production LOC"* ]] || false
}

@test "--threshold flag overrides default" {
    printf 'line %d\n' {1..10} > small.txt
    _commit "small"
    _both --base-ref main --head-ref feature --threshold 5
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
}

@test "specs/archive/ is not counted as a valid spec folder for the gate" {
    mkdir -p specs/archive/OLD-001-archived
    printf 'archived\n' > specs/archive/OLD-001-archived/proposal.md
    printf 'line %d\n' {1..60} > big.txt
    _commit "archive does not satisfy gate"
    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
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

    _both --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]] || false
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

    _both --base-ref main --head-ref feature --explain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Spec folder touched: no"* ]] || false
    [[ "$output" == *"Production LOC (added+removed): 0"* ]] || false
}
