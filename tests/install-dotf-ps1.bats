#!/usr/bin/env bats
# Tests for scripts/install-dotf.ps1 — the Windows dotf release fetch/verify/install
# (WIN-006). PowerShell twin of install-dotf.sh. Structural + convention checks (the
# repo's .ps1 test style — see healthcheck-ps1.bats; PowerShell's Invoke-WebRequest
# has no file:// support, so the .sh's fixture-driven behavioral path can't be
# mirrored here). Behavioral correctness is the direct mirror of install-dotf.sh
# (bats-tested on Linux) plus a real-release smoke during the WIN-006 verification.

load 'lib/refute'

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    PS1="$SCRIPTS_DIR/install-dotf.ps1"
    SETUP_WIN="$BATS_TEST_DIRNAME/../setup-windows.ps1"
}

@test "install-dotf.ps1 exists" {
    [ -f "$PS1" ]
}

@test "has comment-based help (.SYNOPSIS / .DESCRIPTION / .EXAMPLE)" {
    grep -q '\.SYNOPSIS' "$PS1"
    grep -q '\.DESCRIPTION' "$PS1"
    grep -q '\.EXAMPLE' "$PS1"
}

@test "defines Install-Dotf, Get-DotfArch, Get-DotfVersion" {
    grep -qE 'function +Install-Dotf' "$PS1"
    grep -qE 'function +Get-DotfArch' "$PS1"
    grep -qE 'function +Get-DotfVersion' "$PS1"
}

@test "uses advanced functions ([CmdletBinding()])" {
    grep -q '\[CmdletBinding()\]' "$PS1"
}

@test "installs user-space into ~/.local/bin (no admin)" {
    grep -qF '.local\bin' "$PS1"
}

@test "fetches the windows zip and verifies sha256 (parity with install-dotf.sh)" {
    grep -qF 'dotf_${Version}_windows_${arch}.zip' "$PS1"
    grep -qF 'checksums.txt' "$PS1"
    grep -qF 'Get-FileHash' "$PS1"
    grep -qF 'checksum mismatch' "$PS1"
}

@test "maps amd64 + arm64 arch tokens" {
    grep -qF 'amd64' "$PS1"
    grep -qF 'arm64' "$PS1"
}

@test "idempotent: skips when the pinned version is already on PATH" {
    grep -qF 'already installed; skipping' "$PS1"
}

@test "resolves DOTF_VERSION from env then versions.conf" {
    grep -qF 'DOTF_VERSION' "$PS1"
    grep -qF 'versions.conf' "$PS1"
}

@test "standalone run-guard installs when executed, not when dot-sourced" {
    grep -qF 'MyInvocation.InvocationName' "$PS1"
}

@test "replaces a locked dotf.exe by moving it aside, not by overwriting it" {
    # BUG-037 parity. Windows refuses to overwrite or delete a *running* image,
    # but it does allow renaming one, so the swap has to park the live exe and
    # move the staged one into place — the analogue of install-dotf.sh's mv.
    # Behavioral coverage lives in install-dotf-ps1.Tests.ps1 (Pester).
    grep -qE 'function +Set-DotfBinary' "$PS1"
    grep -qF 'Move-Item' "$PS1"
    # The naive one-shot copy straight onto the live target must be gone.
    refute_grep_fixed "Copy-Item -Path \$exe" "$PS1"
}

@test "setup-windows.ps1 dot-sources install-dotf.ps1 and calls Install-Dotf" {
    grep -qF 'install-dotf.ps1' "$SETUP_WIN"
    grep -qF 'Install-Dotf' "$SETUP_WIN"
}
