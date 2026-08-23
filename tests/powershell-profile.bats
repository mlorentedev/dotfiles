#!/usr/bin/env bats
# Tests for powershell/profile.ps1 (structural)

load 'winpath'
load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PROFILE_SCRIPT="$DOTFILES_DIR/powershell/profile.ps1"
}

@test "profile.ps1 exists" {
    [[ -f "$PROFILE_SCRIPT" ]]
}

# --- AI tool aliases ---

@test "profile.ps1 has claude alias" {
    grep -q 'Set-Alias.*-Name c -Value claude' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has agy alias (g -> agy, post-SDD-007 Antigravity migration)" {
    # gemini-cli replaced by Google Antigravity CLI ('agy'). The single-letter
    # alias 'g' carries forward — same finger pattern, new binary.
    grep -q 'Set-Alias.*-Name g -Value agy' "$PROFILE_SCRIPT"
}

# --- OpenCode alias (admin-conditional on Windows; aider sunset) ---

@test "profile.ps1 defines conditional oc alias for opencode (plain, no --pure)" {
    # Guarded by Get-Command -- non-admin Windows machines without opencode
    # skip the alias rather than producing broken references. Plain Set-Alias
    # (no --pure) because empirically opencode launched bare from a Windows
    # terminal works correctly with MCPs+plugins (2026-05-26). Linux keeps
    # --pure kept from the abandoned DX-003 terminal-hang investigation.
    grep -qE 'if \(Get-Command opencode' "$PROFILE_SCRIPT"
    grep -qF 'Set-Alias -Name oc -Value opencode' "$PROFILE_SCRIPT"
    # Regression guard: ensure --pure isn't quietly reintroduced for Windows.
    refute_grep 'function oc\s+\{\s*opencode --pure' "$PROFILE_SCRIPT"
}

@test "profile.ps1 no longer defines aider tier functions (sunset)" {
    refute_grep '^function (ai|aic|aia) ' "$PROFILE_SCRIPT"
}

# --- Cross-machine path resolution (ADR-025): no hardcoded repo literal ---

@test "profile.ps1 dbg resolves nan-debug.sh via DOTFILES_REPO_DIR, not a hardcoded Projects\\dotfiles literal" {
    # Mirror of the Linux AC4 (shell-wrapper-dedup.bats): a relocated clone must
    # still find the repo script. The old literal 'Projects\dotfiles\scripts'
    # broke on any machine whose repo is not under ~/Projects/dotfiles.
    grep -qF '$env:DOTFILES_REPO_DIR' "$PROFILE_SCRIPT"
    refute_grep_fixed "'Projects\\dotfiles\\scripts\\nan-debug.sh'" "$PROFILE_SCRIPT"
}

# --- On-demand secrets (ADR-028: no ambient export; wrap AI CLIs via dotf secrets run) ---

@test "profile.ps1 does NOT auto-source load-secrets and wraps AI CLIs via dotf secrets run (ADR-028)" {
    # ADR-028 "not always exposed": secrets are NOT loaded into the ambient
    # session. opencode (reads {env:NAN_API_KEY} from opencode.jsonc), pi and agy
    # are wrapped to launch through `dotf secrets run`, so the secret lives only
    # in their child process — never the session.
    refute_grep 'load-secrets\.ps1' "$PROFILE_SCRIPT"
    grep -qE 'function opencode \{ dotf secrets run' "$PROFILE_SCRIPT"
}

@test "parity: neither .bashrc nor profile.ps1 auto-loads secrets; both wrap the AI CLIs" {
    refute_grep 'source .*load-secrets\.sh' "$DOTFILES_DIR/.bashrc"
    refute_grep 'load-secrets\.ps1' "$PROFILE_SCRIPT"
    grep -qE 'opencode\(\) \{ dotf secrets run' "$DOTFILES_DIR/.bashrc"
    grep -qE 'function opencode \{ dotf secrets run' "$PROFILE_SCRIPT"
}

# --- Cross-OS oc behaviour (intentional asymmetry, documented) ---

@test "oc is defined on both POSIX and PowerShell (intentional asymmetry on --pure)" {
    # Same finger pattern, different underlying invocation:
    #   Linux : oc() { opencode --pure ...} in .zsh/functions.sh (DX-003 hang workaround)
    #   Windows: Set-Alias -Name oc -Value opencode  (no hang observed)
    # Linux oc moved to functions.sh (shared bash/zsh) per REFACTOR-010; the
    # intentional --pure asymmetry is preserved. Once the Linux hang root cause is
    # found (DX-003, abandoned), the --pure workaround drops everywhere.
    grep -qE '^oc\(\) \{ opencode --pure' "$DOTFILES_DIR/.zsh/functions.sh"
    grep -qF 'Set-Alias -Name oc -Value opencode' "$PROFILE_SCRIPT"
}

# --- Other functions ---

@test "profile.ps1 has project-init function" {
    grep -q 'function project-init' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has prompt function" {
    grep -q 'function prompt' "$PROFILE_SCRIPT"
}

# --- WIN-001: hc function (healthcheck alias mirror of Linux `hc`) ---

@test "profile.ps1 defines hc function (CLI-018: dotf doctor alias)" {
    grep -qE '^function hc' "$PROFILE_SCRIPT"
}

@test "profile.ps1 hc function wraps dotf doctor (CLI-018)" {
    # hc was a healthcheck.ps1 launcher; after CLI-018 it runs dotf doctor.
    grep -A6 '^function hc' "$PROFILE_SCRIPT" | grep -qF 'dotf doctor'
    refute_grep_fixed 'healthcheck.ps1' "$PROFILE_SCRIPT"
}

@test "profile.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    pwsh -NoProfile -Command "Get-Content '$(_winpath "$PROFILE_SCRIPT")' | Out-Null; \$null = [System.Management.Automation.PSParser]::Tokenize((Get-Content '$(_winpath "$PROFILE_SCRIPT")' -Raw), [ref]\$null)"
}
