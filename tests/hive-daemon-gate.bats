#!/usr/bin/env bats
# The hive daemon gate in both setup scripts. The 15-minute auto-upgrade timers
# these tests once lived beside were retired (#2857, hive ADR-020); the gate in
# front of `hive service install` stays.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "parity: both setups probe hive's version through dotf tools version, not uv tool list (AI-034, #791)" {
    # hive moved to its own installer, so `uv tool list` shows no hive-vault on a
    # healthy box; both gates reported "hive <unknown> predates 'hive service'"
    # while `hive --version` answered 3.0.0 (work box, 2026-08-27).
    grep -qF 'dotf tools version hive' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'dotf tools version hive' "$DOTFILES_DIR/setup-windows.ps1"
    run grep -F "hive_ver=\$(uv tool list" "$DOTFILES_DIR/setup-linux.sh"
    [ "$status" -ne 0 ]
    run grep -F "uv tool list 2>\$null | Select-String -Pattern '^hive-vault" "$DOTFILES_DIR/setup-windows.ps1"
    [ "$status" -ne 0 ]
}

@test "no setup script installs an unattended hive upgrade (#2857, hive ADR-020)" {
    run grep -nE 'hive-upgrade|DotfilesHiveUpgrade|uv tool upgrade hive-vault' \
        "$DOTFILES_DIR/setup-linux.sh" "$DOTFILES_DIR/setup-windows.ps1"
    [ "$status" -ne 0 ]
    [ ! -e "$DOTFILES_DIR/systemd/hive-upgrade.timer" ]
    [ ! -e "$DOTFILES_DIR/windows/hive-upgrade.ps1" ]
}
