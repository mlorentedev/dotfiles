#!/usr/bin/env bats
# OPS-001: structure + wiring guards for the self-deploy timer.
# Mirrors hive-upgrade-timer.bats: assert the units, the opt-in gate, and
# cross-OS parity as permanent regression assertions (incident -> guard).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# --- systemd units (SSOT) ---

@test "systemd unit files exist (timer + oneshot service)" {
    [ -f "$DOTFILES_DIR/systemd/dotfiles-selfupdate.timer" ]
    [ -f "$DOTFILES_DIR/systemd/dotfiles-selfupdate.service" ]
}

@test "dotfiles-selfupdate.timer fires daily with catch-up" {
    grep -qF 'OnCalendar=daily' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.timer"
    grep -qF 'Persistent=true' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.timer"
    grep -qF 'RandomizedDelaySec=' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.timer"
}

@test "dotfiles-selfupdate.timer installs to timers.target" {
    grep -qF 'WantedBy=timers.target' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.timer"
}

@test "dotfiles-selfupdate.service is a oneshot running 'dotf update'" {
    grep -qF 'Type=oneshot' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.service"
    grep -qF 'dotf update' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.service"
}

# A --user oneshot gets a minimal PATH; ExecStart must be an absolute path
# (resolved via %h), never a bare script name on PATH.
@test "dotfiles-selfupdate.service ExecStart is absolute (%h-resolved)" {
    grep -qE '^ExecStart=%h/' "$DOTFILES_DIR/systemd/dotfiles-selfupdate.service"
}

# --- setup-linux.sh opt-in wiring (tri-state gate) ---

@test "setup-linux.sh gates the timer on DOTFILES_AUTODEPLOY" {
    grep -qF 'DOTFILES_AUTODEPLOY' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh deploys both units into the systemd --user dir" {
    grep -qF 'systemd/dotfiles-selfupdate.service' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'systemd/dotfiles-selfupdate.timer' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'systemd/user' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh enables the timer with --now when opted in" {
    grep -qF 'enable --now dotfiles-selfupdate.timer' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh disables + removes the timer on opt-out (=0)" {
    grep -qF 'disable --now dotfiles-selfupdate.timer' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh self-deploy block is non-fatal (warns, never hard-exits)" {
    grep -qF 'dotfiles-selfupdate.timer enable failed (non-fatal' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid bash after the self-deploy block" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

# --- setup-windows.ps1 parity ---

@test "setup-windows.ps1 registers a daily DotfilesSelfUpdate task gated on the env var" {
    grep -qF 'DotfilesSelfUpdate' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'DOTFILES_AUTODEPLOY' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 self-deploy task runs 'dotf update'" {
    grep -qF 'dotf.exe' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'Argument "update"' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- cross-OS parity ---

@test "parity: both setup scripts install a self-deploy mechanism gated on DOTFILES_AUTODEPLOY" {
    grep -qF 'enable --now dotfiles-selfupdate.timer' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'DotfilesSelfUpdate' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'DOTFILES_AUTODEPLOY' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'DOTFILES_AUTODEPLOY' "$DOTFILES_DIR/setup-windows.ps1"
}
