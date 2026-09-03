#!/usr/bin/env bats
# CLI-072 (#1460): the GUARD-001 hooks step must run AFTER dotf is installed, on
# both setup scripts.
#
# This guard exists because the inversion was real and silent. setup-windows.ps1
# installed hooks ~270 lines before Install-Dotf, which was harmless while the
# step sourced a PowerShell twin and became a silent failure the moment it
# invoked `dotf`: the surrounding Write-Warn would have reported "hooks install
# incomplete" on every fresh box and setup would have carried on without the
# memory-sink guard.
#
# Nothing in the repo spanned those two call sites, which is exactly why the
# ordering could drift unnoticed. This test is that span.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

# line_of <file> <grep-pattern>: 1-based line number of the FIRST match, or empty.
line_of() {
    grep -n -- "$2" "$1" 2>/dev/null | head -1 | cut -d: -f1
}

@test "setup-linux.sh installs dotf before it installs the git hooks" {
    local install_line hooks_line
    install_line=$(line_of "$REPO/setup-linux.sh" '^ *install_dotf ||')
    hooks_line=$(line_of "$REPO/setup-linux.sh" 'hooks install --source')

    [ -n "$install_line" ]
    [ -n "$hooks_line" ]
    # A dotf invoked before install_dotf resolves to nothing on a fresh machine.
    [ "$install_line" -lt "$hooks_line" ]
}

@test "setup-windows.ps1 installs dotf before it installs the git hooks" {
    local install_line hooks_line
    install_line=$(line_of "$REPO/setup-windows.ps1" 'Install-Dotf))')
    hooks_line=$(line_of "$REPO/setup-windows.ps1" 'hooks install --source')

    [ -n "$install_line" ]
    [ -n "$hooks_line" ]
    # This is the assertion that would have failed before CLI-072 moved the block.
    [ "$install_line" -lt "$hooks_line" ]
}

@test "both setups invoke dotf hooks install, not the retired shell twins" {
    run grep -c 'hooks install --source' "$REPO/setup-linux.sh"
    [ "$status" -eq 0 ]
    [ "$output" -ge 1 ]

    run grep -c 'hooks install --source' "$REPO/setup-windows.ps1"
    [ "$status" -eq 0 ]
    [ "$output" -ge 1 ]
}

@test "neither setup still INVOKES the retired install-git-hooks twins" {
    # The port is finished when nothing CALLS the deleted scripts. Comments that
    # name them in the past tense are deliberately allowed: recording where the
    # behaviour went is what lesson 259 asks for, and a reference is only stale
    # when it claims the thing still exists.
    #
    # So comment lines are stripped before the check. Both shells comment with
    # '#', which is what makes one filter serve both files.
    run bash -c "grep -v '^[[:space:]]*#' '$REPO/setup-linux.sh' | grep -n 'install-git-hooks'"
    [ "$status" -ne 0 ]

    run bash -c "grep -v '^[[:space:]]*#' '$REPO/setup-windows.ps1' | grep -n 'install-git-hooks'"
    [ "$status" -ne 0 ]
}

@test "both setups pass an explicit source and dotfiles dir" {
    # When the repo IS the deploy dir, source and destination are the same
    # directory. Naming both is what lets the #695 self-mirror guard see that
    # case instead of the installer clearing its own source.
    run grep -E 'hooks install --source .* --dotfiles-dir' "$REPO/setup-linux.sh"
    [ "$status" -eq 0 ]

    run grep -E 'hooks install --source .* --dotfiles-dir' "$REPO/setup-windows.ps1"
    [ "$status" -eq 0 ]
}
