#!/usr/bin/env bats
# Tests for AI-023: hive-vault auto-upgrade timer (Linux systemd --user) +
# daily Scheduled Task (Windows). The upgrade policy that feeds the Phase C
# daemon's restart-on-upgrade (hive#176).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# --- systemd unit files (SSOT) ---

@test "systemd unit files exist (timer + oneshot service)" {
    [ -f "$DOTFILES_DIR/systemd/hive-upgrade.timer" ]
    [ -f "$DOTFILES_DIR/systemd/hive-upgrade.service" ]
}

@test "hive-upgrade.timer fires every 15 min wall-clock with catch-up" {
    grep -qF 'OnCalendar=*:0/15' "$DOTFILES_DIR/systemd/hive-upgrade.timer"
    grep -qF 'Persistent=true' "$DOTFILES_DIR/systemd/hive-upgrade.timer"
    grep -qF 'RandomizedDelaySec=' "$DOTFILES_DIR/systemd/hive-upgrade.timer"
}

@test "hive-upgrade.timer installs to timers.target" {
    grep -qF 'WantedBy=timers.target' "$DOTFILES_DIR/systemd/hive-upgrade.timer"
}

@test "hive-upgrade.service is a oneshot running uv tool upgrade hive-vault" {
    grep -qF 'Type=oneshot' "$DOTFILES_DIR/systemd/hive-upgrade.service"
    grep -qF 'ExecStart=%h/.local/bin/uv tool upgrade hive-vault' "$DOTFILES_DIR/systemd/hive-upgrade.service"
}

# The ExecStart must be an ABSOLUTE path: a --user oneshot gets a minimal PATH
# that may omit ~/.local/bin, so a bare `uv` would fail every slot.
@test "hive-upgrade.service ExecStart is absolute, not a bare uv on PATH" {
    ! grep -qE '^ExecStart=uv ' "$DOTFILES_DIR/systemd/hive-upgrade.service"
    grep -qE '^ExecStart=%h/' "$DOTFILES_DIR/systemd/hive-upgrade.service"
}

# --- setup-linux.sh wiring ---

@test "setup-linux.sh deploys both units into the systemd --user dir" {
    grep -qF 'systemd/hive-upgrade.service' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'systemd/hive-upgrade.timer' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'systemd/user' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh enables the timer with --now" {
    grep -qF 'enable --now hive-upgrade.timer' "$DOTFILES_DIR/setup-linux.sh"
}

# The timer must install inside the SAME hive >= 1.32.0 gate as `hive service
# install` -- never route an auto-upgrade onto an old hive. The version gate
# uses `sort -V`; the timer enable line must come AFTER it.
@test "setup-linux.sh timer install is inside the version gate (after sort -V)" {
    gate_line=$(grep -n 'sort -V' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    enable_line=$(grep -n 'enable --now hive-upgrade.timer' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$gate_line" ] && [ -n "$enable_line" ]
    [ "$enable_line" -gt "$gate_line" ]
}

# SSOT cleanup: once the timer is the owner, the legacy manual cron must go.
@test "setup-linux.sh removes the legacy uv tool upgrade hive-vault cron" {
    grep -qF "grep -v 'uv tool upgrade hive-vault' | crontab -" "$DOTFILES_DIR/setup-linux.sh"
}

# The cron strip must be guarded (only remove when a matching line exists) and
# only run after a successful timer enable -- an old-hive machine that skips the
# gate keeps its cron so it is never left with no upgrade owner.
@test "setup-linux.sh cron strip is guarded on the line existing" {
    grep -qF "crontab -l 2>/dev/null | grep -q 'uv tool upgrade hive-vault'" "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh timer install is non-fatal (warns, never hard-exits)" {
    grep -qF 'hive-upgrade.timer enable failed (non-fatal' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid bash after the timer block" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

# --- setup-windows.ps1 wiring ---

@test "setup-windows.ps1 registers a daily DotfilesHiveUpgrade task" {
    grep -qF 'DotfilesHiveUpgrade' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'New-ScheduledTaskTrigger -Daily' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 hive-upgrade task runs uv tool upgrade hive-vault" {
    grep -qF 'tool upgrade hive-vault' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'uv.exe' "$DOTFILES_DIR/setup-windows.ps1"
}

# Same version gate as Linux: the task must register inside the
# `[version]$hiveVer -ge [version]'1.32.0'` branch (after that check).
@test "setup-windows.ps1 hive-upgrade task is inside the version gate" {
    gate_line=$(grep -n "ge \[version\]'1.32.0'" "$DOTFILES_DIR/setup-windows.ps1" | head -1 | cut -d: -f1)
    task_line=$(grep -n 'Register-ScheduledTask -TaskName $hiveUpgradeTask' "$DOTFILES_DIR/setup-windows.ps1" | head -1 | cut -d: -f1)
    [ -n "$gate_line" ] && [ -n "$task_line" ]
    [ "$task_line" -gt "$gate_line" ]
}

# PSScriptAnalyzer fails CI on non-ASCII in .ps1 (em dash / arrows / smart quotes).
@test "setup-windows.ps1 hive-upgrade block is ASCII-only" {
    block=$(sed -n '/DotfilesHiveUpgrade/,/Could not register hive-upgrade/p' "$DOTFILES_DIR/setup-windows.ps1")
    [ -n "$block" ]
    printf '%s' "$block" | LC_ALL=C grep -nP '[^\x00-\x7F]' && return 1
    return 0
}

# --- cross-OS parity ---

@test "parity: both setup scripts install an auto-upgrade mechanism for hive-vault" {
    grep -qF 'enable --now hive-upgrade.timer' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'DotfilesHiveUpgrade' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "parity: both setup scripts gate the upgrade install on hive >= 1.32.0" {
    grep -qF '1.32.0' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF '1.32.0' "$DOTFILES_DIR/setup-windows.ps1"
}
