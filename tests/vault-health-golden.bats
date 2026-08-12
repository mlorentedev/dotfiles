#!/usr/bin/env bats
# Golden characterization corpus for vault-health.sh (CLI-021 increment 2, #672).
#
# The shell is the ORACLE (hashes pinned in tests/golden/vault-health/ORACLE).
# These lock its observable behaviour byte for byte so the Go port can be proved
# equivalent rather than asserted to be.
#
# What this corpus pins that the crystallize one did not have to: **the argv the
# script passes to `obsidian`**. Four of the seven sections talk to that binary,
# and its output is stubbed — so a port could change which flags it sends while
# stdout stayed identical, and every stdout-only assertion would still pass. The
# argv log is therefore a compared artefact, not a debugging aid.
#
# One oracle DEFECT was captured faithfully rather than fixed silently, per the
# CLI-021 proposal — a port that "improves while translating" cannot be
# characterization-tested — and ticketed separately (#891):
#   obsidian_cmd() appends `--vault "$VAULT_NAME"`, and four of its five callers
#   passed the same flag again, so those invocations carried `--vault` twice.
#   Visible only in expected/obsidian-argv; stdout was identical either way.
# #891 fixed the oracle and deliberately recaptured this corpus to match — see
# that commit for the reason. Left in this header as a record of the class:
# the same shape can recur, and the next instance should also get its own
# ticket + deliberate recapture rather than a silent "cleanup" folded into
# an unrelated change.

setup() {
    HERE="$BATS_TEST_DIRNAME/golden/vault-health"
    export GVH_ORACLE_SH="$BATS_TEST_DIRNAME/../scripts/vault-health.sh"
    # shellcheck source=golden/vault-health/lib.sh disable=SC1091
    . "$HERE/lib.sh"
    ACTUAL="$(mktemp -d)"
}

teardown() {
    rm -rf "$ACTUAL"
}

assert_golden() {
    local name="$1" case_dir="$HERE/cases/$1"
    [ -d "$case_dir/expected" ] || {
        printf 'no golden captured for %s - run tests/golden/vault-health/capture.sh\n' "$name" >&2
        return 1
    }
    gvh_run_case "$case_dir" "$ACTUAL/$name"
    local artefact
    for artefact in exit stdout obsidian-argv; do
        if ! diff -u "$case_dir/expected/$artefact" "$ACTUAL/$name/$artefact"; then
            printf 'golden mismatch: %s/%s\n' "$name" "$artefact" >&2
            return 1
        fi
    done
}

@test "a healthy vault passes every check" {
    assert_golden all-pass
}

@test "the GUI being down exits 2, not 1" {
    assert_golden gui-down
}

@test "obsidian missing from PATH exits 1 before any check runs" {
    assert_golden no-obsidian
}

@test "orphans in the 30-50 band warn" {
    assert_golden orphans-warn
}

@test "orphans over 50 percent fail" {
    assert_golden orphans-fail
}

@test "unresolved links at or under 10 warn" {
    assert_golden unresolved-warn
}

@test "unresolved links over 10 fail" {
    assert_golden unresolved-fail
}

@test "frontmatter coverage tiers - pass, warn and fail in one run" {
    assert_golden frontmatter-tiers
}

# Exactly ON the tier edges. Added after mutation testing: moving the orphan
# threshold from 30 to 25 turned only one unrelated case red, because every
# case sat comfortably inside a band. A boundary that no fixture lands on is a
# boundary no test defends.

@test "orphans at exactly 30 percent pass and dead-ends at exactly 50 percent warn" {
    assert_golden orphans-boundary
}

@test "exactly 10 unresolved links warn rather than fail" {
    assert_golden unresolved-boundary
}

@test "frontmatter at exactly 80 percent passes and exactly 50 percent warns" {
    assert_golden frontmatter-boundary
}

@test "a clean git working tree passes section 1" {
    assert_golden worktree-clean
}

@test "a file deleted from disk but still in HEAD fails section 1" {
    assert_golden worktree-deleted
}

@test "a missing vault directory fails connectivity and skips the rest" {
    assert_golden missing-vault-dir
}

@test "--verbose adds the detail listings" {
    assert_golden verbose
}

@test "backlog drift in a task file fails section 7" {
    assert_golden backlog-drift
}

# ── corpus integrity ──────────────────────────────────────────────────────────

@test "every case directory has a golden and a test" {
    local d name missing="" ungolden=""
    for d in "$HERE"/cases/*/; do
        name="$(basename "$d")"
        [ -d "$d/expected" ] || ungolden="$ungolden $name"
        grep -q "assert_golden $name\$" "$BATS_TEST_DIRNAME/vault-health-golden.bats" \
            || missing="$missing $name"
    done
    [ -z "$ungolden" ] || { printf 'cases with no captured golden:%s\n' "$ungolden" >&2; return 1; }
    [ -z "$missing" ] || { printf 'cases with a golden but no test:%s\n' "$missing" >&2; return 1; }
}

@test "the oracle content hashes are recorded and still match the working tree" {
    # Pinned by CONTENT, not by commit SHA: a SHA is unavailable under CI's
    # shallow checkout and changes on every rebase without a byte changing
    # (docs/lessons.md, 2026-08-09). This check therefore never calls git.
    local oracle="$HERE/ORACLE"
    [ -f "$oracle" ] || { printf 'ORACLE missing\n' >&2; return 1; }

    local seen=0 hash f current
    while read -r hash f; do
        case "$hash" in '#'*|'') continue ;; esac
        current="$(sha256sum "$BATS_TEST_DIRNAME/../$f" | cut -d' ' -f1)"
        if [ "$current" != "$hash" ]; then
            printf 'oracle drift: %s\n  ORACLE: %s\n  tree:   %s\n' "$f" "$hash" "$current" >&2
            printf 'Re-run tests/golden/vault-health/capture.sh, deliberately, with the reason in the commit.\n' >&2
            return 1
        fi
        seen=$((seen + 1))
    done < "$oracle"

    # A vacuous pass is the failure mode this guards against: an ORACLE that
    # parsed to zero rows would "match" everything.
    [ "$seen" -eq 3 ]
}

@test "the argv log no longer records a duplicated --vault flag (#891)" {
    # Was: this test pinned the oracle's observable defect (obsidian_cmd()
    # already appends --vault, and four of its five callers passed it again),
    # so a port that silently "cleaned it up" would go red with an explanation
    # instead of a bare diff. #891 fixed the oracle itself (dropped the
    # redundant flag from the four callers) and this corpus was deliberately
    # recaptured (tests/golden/vault-health/capture.sh) to match — so this test
    # now pins the FIXED shape, and a regression back to double-passing the
    # flag is what would turn it red.
    ! grep -q -- '--vault knowledge --vault knowledge' \
        "$HERE/cases/all-pass/expected/obsidian-argv"
    grep -qx -- '--no-sandbox orphans --vault knowledge' \
        "$HERE/cases/all-pass/expected/obsidian-argv"
    # The connectivity probe always passed it once; unaffected by the fix.
    grep -qx -- '--no-sandbox vault --vault knowledge' \
        "$HERE/cases/all-pass/expected/obsidian-argv"
}
