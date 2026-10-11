#!/usr/bin/env bats
# The GUARD-001 dispatcher is converge's `git-hooks` step on every OS (#2013
# X1), so neither setup script installs it any more: each delegates to the
# step, as it does for git-config.
#
# The step used to live in both setup twins, and macOS has no setup script, so
# a Mac never got the dispatcher: the from-zero job found it missing on
# macos-latest. The native step runs before legacy-setup on Linux and Windows
# too. A setup block that came back would install twice and drift from the step
# again, with a second set of guards nobody tests. The delegation keeps a setup
# script run on its own complete: test-windows runs setup-windows.ps1 directly,
# and without it the doctor gate found no dispatcher (#2340).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "neither setup script installs the git hooks" {
    for f in setup-linux.sh setup-windows.ps1; do
        [ -f "$REPO/$f" ] || { echo "$f is missing" >&2; false; }
        run bash -c "grep -v '^[[:space:]]*#' '$REPO/$f' | grep -nE 'hooks install|install-git-hooks'"
        [ "$status" -eq 1 ] || { echo "$f still installs the hooks: $output" >&2; false; }
    done
}

@test "converge registers the git-hooks step" {
    grep -qF 'gitHooks{run: o.GitRun' "$REPO/cli/internal/converge/records.go" || false
}

@test "both setup scripts delegate the hooks to the converge step" {
    for f in setup-linux.sh setup-windows.ps1; do
        tr -d '\r' < "$REPO/$f" | grep -v '^[[:space:]]*#' | grep -qE 'converge --only git-hooks[[:space:]]*(\|\||$)' \
            || { echo "$f does not run converge --only git-hooks" >&2; false; }
    done
}
