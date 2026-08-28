#!/usr/bin/env bats
# Characterization of `dotf vault health` against the frozen golden corpus
# (CLI-021 increment 2, #490) — the sibling of knowledge-crystallize-go-parity.bats.
#
# Unlike that file, vault-health.sh is NOT deleted here: this increment builds
# the port BESIDE the shell twin (no caller repointed, nothing cut over — that
# is CLI-023 / #492). This suite runs gvh_run_case's "go" mode against the
# frozen goldens under tests/golden/vault-health/cases/*/expected — the SAME
# artefacts tests/vault-health-golden.bats checks the live shell against. A
# shell edit alone cannot silently pass or fail this file: only the shell suite
# re-derives from the live script, and its ORACLE hash check catches drift.
#
# Skips (never fails) when the Go toolchain is absent, so a shell-only checkout
# still runs the rest of the suite — same reasoning as the crystallize parity
# file (#807 / BUG-055: a check that skips instead of running is worse than one
# that is simply absent).

setup() {
    HERE="$BATS_TEST_DIRNAME/golden/vault-health"
    # shellcheck source=golden/vault-health/lib.sh disable=SC1091
    . "$HERE/lib.sh"
    ACTUAL="$(mktemp -d)"
}

teardown() {
    rm -rf "$ACTUAL"
}

# Build once per test file run, into a cached location, so 16 cases do not pay
# 16 compilations.
_build_dotf() {
    command -v go >/dev/null 2>&1 || skip "go toolchain not installed"
    GVH_DOTF_BIN="${BATS_FILE_TMPDIR:-/tmp}/dotf-vault-health-parity"
    export GVH_DOTF_BIN
    if [ ! -x "$GVH_DOTF_BIN" ]; then
        # A missing toolchain skips (checked above); a toolchain that FAILS to
        # build the CLI is a real defect and must fail the suite, not read as
        # 16 harmless skips.
        ( cd "$BATS_TEST_DIRNAME/../cli" && go build -o "$GVH_DOTF_BIN" ./cmd/dotf ) \
            || return 1
    fi
    export GVH_IMPL_MODE=go
}

assert_parity() {
    local name="$1" case_dir="$HERE/cases/$1"
    _build_dotf
    gvh_run_case "$case_dir" "$ACTUAL/$name"
    local artefact
    for artefact in exit stdout obsidian-argv; do
        if ! diff -u "$case_dir/expected/$artefact" "$ACTUAL/$name/$artefact"; then
            printf 'go/shell parity broken: %s/%s\n' "$name" "$artefact" >&2
            return 1
        fi
    done
}

@test "parity: a healthy vault passes every check" {
    assert_parity all-pass
}

@test "parity: the GUI being down exits 2, not 1" {
    assert_parity gui-down
}

@test "parity: obsidian missing from PATH exits 1 before any check runs" {
    assert_parity no-obsidian
}

@test "parity: orphans in the 30-50 band warn" {
    assert_parity orphans-warn
}

@test "parity: orphans over 50 percent fail" {
    assert_parity orphans-fail
}

@test "parity: unresolved links at or under 10 warn" {
    assert_parity unresolved-warn
}

@test "parity: unresolved links over 10 fail" {
    assert_parity unresolved-fail
}

@test "parity: frontmatter coverage tiers - pass, warn and fail in one run" {
    assert_parity frontmatter-tiers
}

@test "parity: orphans at exactly 30 percent pass and dead-ends at exactly 50 percent warn" {
    assert_parity orphans-boundary
}

@test "parity: exactly 10 unresolved links warn rather than fail" {
    assert_parity unresolved-boundary
}

@test "parity: frontmatter at exactly 80 percent passes and exactly 50 percent warns" {
    assert_parity frontmatter-boundary
}

@test "parity: a clean git working tree passes section 1" {
    assert_parity worktree-clean
}

@test "parity: a file deleted from disk but still in HEAD fails section 1" {
    assert_parity worktree-deleted
}

@test "parity: a missing vault directory fails connectivity and skips the rest" {
    assert_parity missing-vault-dir
}

@test "parity: --verbose adds the detail listings" {
    assert_parity verbose
}

@test "parity: backlog drift in a task file fails section 7 and aborts (oracle defect #1314)" {
    assert_parity backlog-drift
}

@test "the parity set covers every golden case" {
    local d name missing=""
    for d in "$HERE"/cases/*/; do
        name="$(basename "$d")"
        grep -q "assert_parity $name\$" "$BATS_TEST_DIRNAME/vault-health-go-parity.bats" \
            || missing="$missing $name"
    done
    [ -z "$missing" ] || {
        printf 'cases with a golden but no parity test —%s\n' "$missing" >&2
        return 1
    }
}
