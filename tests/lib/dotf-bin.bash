#!/usr/bin/env bash
# Resolve the `dotf` binary a bats file runs against.  Load with: load 'lib/dotf-bin'
#
# WHY THIS EXISTS
# ---------------
# Five files ran the real binary and each built its own copy per file run, three
# of them with `go build ... || skip`.  That skip is the defect: a compile error
# in cli/ turned those tests into SKIPS, so the `test` job stayed green while
# saying nothing about them (BUG-055 / #807, the class their headers warn about).
#
# THE RULE (dotf_bin_resolve <fallback-path>)
# -------------------------------------------
#   1. DOTF_BIN set     -> use it.  CI builds once, before bats, and exports it.
#                          Set but not an executable file is a FAILURE, never a
#                          fall-through to a rebuild: the caller asked for that
#                          binary, and quietly using a different one tests
#                          something else.
#   2. no Go toolchain  -> SKIP locally, FAIL when $CI is set.  A shell-only
#                          checkout is a supported machine and the file headers
#                          promise the rest of the suite still runs there; CI
#                          installs Go on purpose, so a missing toolchain there is
#                          a broken runner, not a reason to run less.
#   3. otherwise        -> build into <fallback-path> (cached by existence, so a
#                          file pays one build).  A build that FAILS is a failure
#                          everywhere, locally included: a tree that does not
#                          compile is a defect, not an environment.
#
# On success DOTF_BIN is exported and names the binary.

dotf_bin_resolve() {
    local fallback="$1" cli
    if [ -n "${DOTF_BIN:-}" ]; then
        if [ ! -x "$DOTF_BIN" ] || [ -d "$DOTF_BIN" ]; then
            printf 'DOTF_BIN is set to %s, which is not an executable file\n' "$DOTF_BIN" >&2
            return 1
        fi
        export DOTF_BIN
        return 0
    fi
    if ! command -v go >/dev/null 2>&1; then
        if [ -n "${CI:-}" ]; then
            printf 'the go toolchain is missing in CI: a runner without it cannot build dotf, and skipping would hide that\n' >&2
            return 1
        fi
        skip "go toolchain not installed"
    fi
    if [ ! -x "$fallback" ]; then
        cli="$BATS_TEST_DIRNAME/../cli"
        mkdir -p "$(dirname "$fallback")"
        if ! ( cd "$cli" && go build -o "$fallback" ./cmd/dotf ); then
            printf 'go build ./cmd/dotf failed in %s: a tree that does not compile is a defect, not a reason to skip\n' "$cli" >&2
            return 1
        fi
    fi
    DOTF_BIN="$fallback"
    export DOTF_BIN
}
