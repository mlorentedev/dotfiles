#!/usr/bin/env bats
# GUARD (OPS-043): the REFACTOR-002 path vars must stay pre-exported in both
# setup scripts, immediately before each hands off to `dotf doctor`.
#
# WHAT THIS GUARDS AGAINST
# ------------------------
# Issue #1337 asked for these six exports to be deleted, describing them as
# duplication because "doctor covers the same" a few lines later. They are not
# duplication. They exist because the shell running setup has not re-sourced
# .zshrc/.bashrc, so the values the deployed rc files WILL set on the next login
# are not in the current environment. Without the pre-export, doctor reads the
# stale environment and reports contract-variable warnings on every fresh setup
# — which is BUG-021, closed by adding these lines, and mirrored into
# setup-windows.ps1 for the same reason.
#
# Deleting them therefore does not remove a redundant check; it reopens a fixed
# bug in a way that looks green in code review and only shows up as noise on the
# next clean box. OPS-043 deleted the two shell verification calls that WERE
# duplication (check_deployed, check_dependencies) and deliberately kept these,
# so this guard records the distinction where the next reader of #1337 will hit
# it.
#
# WHY BOTH SCRIPTS
# ----------------
# The Windows twin carries the same fix at setup-windows.ps1:2185. A guard that
# only knew about Linux would let the regression back in on the platform where
# it is harder to notice.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# The six vars, as named in env-contract.json. AGY_HOME is the SSOT spelling;
# it lives inside GEMINI_HOME for backwards compat with gemini-cli.
preexported_vars() {
    printf '%s\n' SCRIPTS_DIR GEMINI_HOME AGY_HOME COPILOT_HOME OPENCODE_HOME DOTFILES_REPO_DIR
}

@test "setup-linux.sh pre-exports every REFACTOR-002 path var (BUG-021)" {
    script="$DOTFILES_DIR/setup-linux.sh"
    [ -f "$script" ]

    missing=""
    while IFS= read -r var; do
        if ! grep -qE "^export ${var}=" "$script"; then
            missing="$missing $var"
        fi
    done <<EOF
$(preexported_vars)
EOF

    if [ -n "$missing" ]; then
        printf 'setup-linux.sh is missing pre-exports for:%s\n' "$missing" >&2
        printf 'These are not duplication of dotf doctor -- see BUG-021 and #1337.\n' >&2
        return 1
    fi
}

@test "setup-windows.ps1 pre-exports every REFACTOR-002 path var (BUG-021)" {
    script="$DOTFILES_DIR/setup-windows.ps1"
    [ -f "$script" ]

    missing=""
    while IFS= read -r var; do
        # PowerShell spells it $env:NAME = ... ; accept any assignment form.
        if ! grep -qE "env:${var}" "$script"; then
            missing="$missing $var"
        fi
    done <<EOF
$(preexported_vars)
EOF

    if [ -n "$missing" ]; then
        printf 'setup-windows.ps1 is missing pre-exports for:%s\n' "$missing" >&2
        printf 'These mirror the Linux BUG-021 fix -- see #1337.\n' >&2
        return 1
    fi
}

@test "the pre-exports still precede the dotf doctor handoff in setup-linux.sh" {
    script="$DOTFILES_DIR/setup-linux.sh"

    # The exports are only useful if doctor runs AFTER them in the same shell.
    # Reordering them below the handoff would leave the lines present and the
    # bug reopened, which a presence-only guard would call green.
    last_export=$(grep -nE '^export DOTFILES_REPO_DIR=' "$script" | tail -1 | cut -d: -f1)
    doctor_call=$(grep -nE '^\s*dotf doctor' "$script" | tail -1 | cut -d: -f1)

    [ -n "$last_export" ]
    [ -n "$doctor_call" ]
    [ "$last_export" -lt "$doctor_call" ]
}
