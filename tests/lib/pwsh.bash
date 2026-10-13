#!/usr/bin/env bash
# require_pwsh: the gate for a test that runs pwsh.  Load with: load 'lib/pwsh'
#
# `command -v pwsh` answers whether a file named pwsh is on PATH, not whether it
# runs. A mise shim for a tool with no active version is such a file: every call
# exits with "No version is set for shim: pwsh", so on a Mac where powershell
# once went through mise the PowerShell tests failed as product failures on a
# clean main (#2375). The gate runs pwsh once instead, and skips when it cannot.
#
# Where a job promises pwsh (DOTFILES_CI_EXPECT_PWSH=1, set by the CI steps that
# run these files), a pwsh that is missing or broken fails the test rather than
# skipping it: a gate that turns a broken runner into green skips is how the
# Pester suites once skipped as green (TEST-006).
require_pwsh() {
    local why out
    if ! command -v pwsh >/dev/null 2>&1; then
        why="pwsh not on PATH"
    elif ! out=$(pwsh -NoProfile -NonInteractive -Command 'exit 0' 2>&1 </dev/null); then
        why="pwsh is on PATH but does not run: $(printf '%s\n' "$out" | head -n 1)"
    else
        return 0
    fi
    if [ -n "${DOTFILES_CI_EXPECT_PWSH:-}" ]; then
        printf 'DOTFILES_CI_EXPECT_PWSH is set and %s\n' "$why" >&2
        return 1
    fi
    skip "$why"
}
