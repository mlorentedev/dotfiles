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

@test "every pre-export precedes the dotf doctor handoff in setup-linux.sh" {
    script="$DOTFILES_DIR/setup-linux.sh"

    # The exports are only useful if doctor runs AFTER them in the same shell.
    # Reordering ONE of them below the handoff leaves every line present and the
    # bug reopened for that variable, which a presence-only guard calls green --
    # and so does an ordering guard that samples a single variable.
    doctor_call=$(grep -nE '^[[:space:]]*dotf doctor' "$script" | tail -1 | cut -d: -f1)
    [ -n "$doctor_call" ]

    late=""
    while IFS= read -r var; do
        line=$(grep -nE "^export ${var}=" "$script" | tail -1 | cut -d: -f1)
        if [ -z "$line" ] || [ "$line" -gt "$doctor_call" ]; then
            late="$late $var"
        fi
    done <<EOF
$(preexported_vars)
EOF

    if [ -n "$late" ]; then
        printf 'these pre-exports do not precede the dotf doctor call (line %s):%s\n' "$doctor_call" "$late" >&2
        return 1
    fi
}

@test "every pre-export precedes the dotf doctor handoff in setup-windows.ps1" {
    script="$DOTFILES_DIR/setup-windows.ps1"

    # The Windows twin has the same ordering requirement and the same failure
    # mode. Matching the ASSIGNMENT (`$env:VAR =`) rather than any mention of
    # the name: GEMINI_HOME appears on the AGY_HOME line too, so a bare name
    # match would read the wrong line number.
    doctor_call=$(grep -nE '^[[:space:]]*& dotf doctor' "$script" | tail -1 | cut -d: -f1)
    [ -n "$doctor_call" ]

    late=""
    while IFS= read -r var; do
        line=$(grep -nE "\\\$env:${var}[[:space:]]*=" "$script" | tail -1 | cut -d: -f1)
        if [ -z "$line" ] || [ "$line" -gt "$doctor_call" ]; then
            late="$late $var"
        fi
    done <<EOF
$(preexported_vars)
EOF

    if [ -n "$late" ]; then
        printf 'these pre-exports do not precede the & dotf doctor call (line %s):%s\n' "$doctor_call" "$late" >&2
        return 1
    fi
}
