#!/usr/bin/env bats
# Tests for AI-023: hive-vault auto-upgrade timer (Linux systemd --user) +
# daily Scheduled Task (Windows). The upgrade policy that feeds the Phase C
# daemon's restart-on-upgrade (hive#176).

load 'lib/refute'

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
    refute_grep '^ExecStart=uv ' "$DOTFILES_DIR/systemd/hive-upgrade.service"
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

# Two tests pinning the legacy-cron strip lived here until OPS-040: that setup
# removed a `uv tool upgrade hive-vault` crontab line once the timer owned
# upgrade policy, and that the removal was guarded on the line existing. The
# block they pinned is gone -- probed absent from `crontab -l` on msi, which is
# the only OS it could ever have run on, crontab being the mechanism.
#
# Nothing replaces them. Asserting the ABSENCE of the strip would pin a deletion
# rather than an invariant, and the invariant that mattered -- the timer is the
# single upgrade owner -- is already covered by the enable/version-gate tests
# above and the non-fatal test below.

@test "setup-linux.sh timer install is non-fatal (warns, never hard-exits)" {
    grep -qF 'hive-upgrade.timer enable failed (non-fatal' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid bash after the timer block" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

# --- setup-windows.ps1 wiring ---

@test "setup-windows.ps1 registers a 15-min DotfilesHiveUpgrade task (Linux parity)" {
    grep -qF 'DotfilesHiveUpgrade' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'RepetitionInterval (New-TimeSpan -Minutes 15)' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 hive-upgrade task runs the orchestration script (not uv directly)" {
    grep -qF 'windows\hive-upgrade.ps1' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'New-ScheduledTaskAction -Execute "powershell.exe"' "$DOTFILES_DIR/setup-windows.ps1"
}

# Same version gate as Linux: the task must register inside the
# `[version]$hiveVer -ge [version]'1.32.0'` branch (after that check).
@test "setup-windows.ps1 hive-upgrade task is inside the version gate" {
    gate_line=$(grep -n "ge \[version\]'1.32.0'" "$DOTFILES_DIR/setup-windows.ps1" | head -1 | cut -d: -f1)
    task_line=$(grep -n 'Register-HiveScheduledTask -TaskName $hiveUpgradeTask' "$DOTFILES_DIR/setup-windows.ps1" | head -1 | cut -d: -f1)
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

# dotfiles#230: hive Scheduled Tasks must run windowless. A task registered with
# no explicit principal defaults to the Interactive logon type, runs in the
# desktop session, and pops a console window every 15-min tick. An S4U principal
# runs the task in session 0 (no desktop -> no window) -- but REGISTERING an S4U
# task requires an elevated caller, so the helper picks the strongest achievable
# logon type: S4U when elevated, Interactive (with -WindowStyle Hidden) otherwise.
@test "setup-windows.ps1 registers hive tasks via the strongest-principal helper" {
    grep -qF 'function Register-HiveScheduledTask' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF -- '-LogonType $script:HiveTaskLogonType' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF "WindowsBuiltInRole]::Administrator" "$DOTFILES_DIR/setup-windows.ps1"
}

# Register-ScheduledTask raises "Access is denied" as a NON-terminating error;
# without -ErrorAction Stop it slips past the call sites' try/catch and setup
# prints a false SUCCESS for a task that was never (re)registered.
@test "setup-windows.ps1 hive task registration failures are terminating" {
    helper=$(sed -n '/function Register-HiveScheduledTask/,/^}/p' "$DOTFILES_DIR/setup-windows.ps1")
    [ -n "$helper" ]
    printf '%s' "$helper" | grep -qF -- '-ErrorAction Stop'
}

# The upgrade task must go through the helper (S4U guaranteed), never the bare
# Register-ScheduledTask (which would default to the windowed Interactive logon).
@test "setup-windows.ps1 routes the hive-upgrade task through the S4U helper" {
    grep -qF 'Register-HiveScheduledTask -TaskName $hiveUpgradeTask' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep_fixed 'Register-ScheduledTask -TaskName $hiveUpgradeTask' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 hive-upgrade action is windowless (-WindowStyle Hidden)" {
    grep -qF -- '-WindowStyle Hidden' "$DOTFILES_DIR/setup-windows.ps1"
}

# Re-running setup on a box that still has a drifted principal must repair it --
# the idempotence check has to inspect the logon type, not just the action's
# Execute/Arguments. The expected type is the strongest achievable one (S4U
# elevated / Interactive non-admin), so non-admin boxes stay idempotent instead
# of failing the S4U re-registration on every run.
@test "setup-windows.ps1 self-heals a drifted hive-upgrade principal" {
    grep -qF '$existingHiveLogon -eq $script:HiveTaskLogonType' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- Windows daemon upgrade orchestration (ADR-015 / hive#176) ---
# Windows cannot replace a running executable, so the upgrade cannot be a bare
# `uv tool upgrade`: it must stop the daemon first and defer if a client session
# holds the install. This logic lives in windows/hive-upgrade.ps1 (SSOT).

@test "windows/hive-upgrade.ps1 orchestration script exists" {
    [ -f "$DOTFILES_DIR/windows/hive-upgrade.ps1" ]
}

@test "hive-upgrade.ps1 defers when a client session holds the install" {
    grep -qF 'deferring' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    grep -qF 'holders' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
}

@test "hive-upgrade.ps1 stops the daemon, upgrades, then restarts" {
    grep -qF 'Stop-ScheduledTask' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    grep -qF 'tool upgrade hive-vault' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    grep -qF 'Start-ScheduledTask' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
}

# At a 15-min cadence the daemon must not be restarted every tick: only act when
# a newer version is actually published (compare installed vs latest first).
@test "hive-upgrade.ps1 only acts when a newer version is published" {
    grep -qF 'pypi.org/pypi/hive-vault' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    grep -qF '[version]$installed -ge [version]$latest' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
}

# A `uv tool ... --reinstall` removes the locked venv dir (os error 5) and
# corrupts the install -- the script documents the hazard in a comment but must
# never actually run it (so the check is on a real uv command, not the word).
# AI-028 / #791: the three step-0 outcomes used to collapse into one silent
# `exit 0`, so a machine with NO install was indistinguishable from a healthy
# idle one. It ran every 15 minutes for months reporting LastTaskResult 0 while
# the hive MCP was dead. These four cases pin each outcome to its own signal.

@test "hive-upgrade.ps1 is loud and non-zero when no install is found" {
    run grep -A3 -F 'no hive-vault install found' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    [ "$status" -eq 0 ]
    [[ "$output" == *"exit 1"* ]]
}

@test "hive-upgrade.ps1 stays silent when the install is already current" {
    # The deliberate 15-min no-op: no output, exit 0, daemon untouched.
    run grep -A2 -F '[version]$installed -ge [version]$latest' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    [ "$status" -eq 0 ]
    [[ "$output" == *"exit 0"* ]]
    [[ "$output" != *"Write-Output"* ]]
}

@test "hive-upgrade.ps1 reports an unreachable PyPI without failing the tick" {
    run grep -A3 -F 'could not resolve the latest' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
    [ "$status" -eq 0 ]
    [[ "$output" == *"exit 0"* ]]
}

@test "hive-upgrade.ps1 does not collapse no-install into the already-current guard" {
    refute_grep_fixed '-not $installed -or' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
}

@test "hive-upgrade.ps1 never runs uv tool ... --reinstall" {
    refute_grep '\$uv tool.*--reinstall' "$DOTFILES_DIR/windows/hive-upgrade.ps1"
}

@test "hive-upgrade.ps1 is ASCII-only (PSScriptAnalyzer CI)" {
    LC_ALL=C grep -nP '[^\x00-\x7F]' "$DOTFILES_DIR/windows/hive-upgrade.ps1" && return 1
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
