#!/usr/bin/env bats
# CI's shellcheck runs at the versions.conf pin, over every shell file the repo
# ships (#1330). A floating linter reports "clean" on code the pinned one
# rejects (BUG-071), and a file outside the scan is never linted at all.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    # The lint job's lines: from its key to the next job's key.
    LINT=$(awk '/^  [a-z-]+:$/ { in_lint = ($0 == "  lint:") } in_lint' "$REPO/.github/workflows/ci.yml")
    [ -n "$LINT" ] || { echo "no lint job in ci.yml" >&2; return 1; }
}

@test "ci: the lint job installs shellcheck at SHELLCHECK_VERSION and checks it" {
    printf '%s\n' "$LINT" | grep -q 'SHELLCHECK_VERSION'
    printf '%s\n' "$LINT" | grep -q 'koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/'
    printf '%s\n' "$LINT" | grep -qE 'shellcheck --version'
}

@test "ci: the lint job uses no shellcheck action, whose bundled version floats" {
    run grep -n 'action-shellcheck' "$REPO/.github/workflows/ci.yml"
    [ "$status" -eq 1 ]
}

@test "ci: shellcheck covers the hooks, the installer and the setup script" {
    local p
    for p in 'scripts/' 'setup-linux.sh' 'git-hooks/' '.github/hooks/' 'install.sh'; do
        printf '%s\n' "$LINT" | grep -A40 'name: Check for shellcheck issues' | grep -qF "$p" \
            || { echo "shellcheck step does not name $p" >&2; return 1; }
    done
}
