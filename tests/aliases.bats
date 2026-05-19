#!/usr/bin/env bats
# Tests for .zsh/aliases.zsh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export ALIASES_FILE="$DOTFILES_DIR/.zsh/aliases.zsh"
}

@test "aliases.zsh exists" {
    [[ -f "$ALIASES_FILE" ]]
}

@test "aliases.zsh valid bash syntax" {
    bash -n "$ALIASES_FILE"
}

@test "aliases.zsh valid zsh syntax" {
    zsh -n "$ALIASES_FILE"
}

# --- DevOps aliases ---

@test "aliases.zsh defines kubectl alias" {
    grep -q 'alias k="kubectl"' "$ALIASES_FILE"
}

@test "aliases.zsh defines helm alias" {
    grep -q 'alias h="helm"' "$ALIASES_FILE"
}

@test "aliases.zsh defines terraform alias" {
    grep -q 'alias tf="terraform"' "$ALIASES_FILE"
}

# --- OpenCode alias (replaces aider tiers, sunset) ---

@test "aliases.zsh defines oc alias for opencode" {
    grep -q '^alias oc="opencode"' "$ALIASES_FILE"
}

@test "aliases.zsh defines oclog alias for live opencode log tailing" {
    # Surfaced 2026-05-17 during AI-011-validation -- visibility into
    # opencode TUI while it shows "thinking..." for a long time.
    grep -q "^alias oclog='tail -F" "$ALIASES_FILE"
    grep -q 'opencode/log' "$ALIASES_FILE"
}

@test "aliases.zsh no longer defines aider tier aliases (sunset)" {
    ! grep -qE '^alias (ai|aic|aia)=' "$ALIASES_FILE"
}

# --- Copilot CLI v2 aliases (BUG-003: rename from ghcs/ghce) ---
# The old aliases wrapped 'gh copilot suggest/explain' which do not exist in
# the new standalone 'copilot' CLI. Rename to 'cop' (interactive agentic CLI)
# and 'cops' (single-shot non-interactive prompt with --allow-all-tools).

@test "aliases.zsh defines cop alias for copilot" {
    grep -qE '^alias cop="copilot"' "$ALIASES_FILE"
}

@test "aliases.zsh defines cops function for single-shot copilot prompt" {
    grep -qE '^cops\(\)' "$ALIASES_FILE"
    grep -qF 'copilot -p' "$ALIASES_FILE"
    grep -qF -- '--allow-all-tools' "$ALIASES_FILE"
}

# --- qq / qf: cross-platform quick-question wrappers ---
# Names are "qq"/"qf" (not "??"/"?f") because PowerShell 7+ reserves "??" as
# the null-coalescing operator; this is the cross-platform compromise.
# zsh aliases use `noglob` so `qq por que tardas tanto?` works without quotes.

@test "aliases.zsh defines qq alias pinning qwen3.6-plus via noglob" {
    grep -qE "^alias qq='noglob _qq_call opencode-go/qwen3.6-plus" "$ALIASES_FILE"
}

@test "aliases.zsh defines qf alias pinning deepseek-v4-flash via noglob" {
    grep -qE "^alias qf='noglob _qq_call opencode-go/deepseek-v4-flash" "$ALIASES_FILE"
}

@test "aliases.zsh _qq_call helper invokes opencode run" {
    grep -qE '^_qq_call\(\)' "$ALIASES_FILE"
    grep -qF 'opencode run' "$ALIASES_FILE"
}

@test "aliases.zsh _qq_call prints usage when called with no args" {
    # Single usage string parameterized by the alias name ($name).
    grep -qF 'usage: %s' "$ALIASES_FILE"
}

@test "aliases.zsh no longer defines ghcs/ghce (renamed in BUG-003)" {
    # Anchor to start-of-line + alias/function definition forms only -- comments
    # mentioning the old names (e.g. "replaces ghcs/ghce wrappers") are fine.
    ! grep -qE '^(alias ghcs|ghcs\(\)|function ghcs)' "$ALIASES_FILE"
    ! grep -qE '^(alias ghce|ghce\(\)|function ghce)' "$ALIASES_FILE"
}

# --- Knowledge aliases ---

@test "aliases.zsh defines kc alias" {
    grep -q 'alias kc=' "$ALIASES_FILE"
}

@test "aliases.zsh defines kca alias" {
    grep -q 'alias kca=' "$ALIASES_FILE"
}

# --- Navigation aliases ---

@test "aliases.zsh gprj points to Projects directory" {
    grep 'alias gprj=' "$ALIASES_FILE" | grep -q 'Projects'
}

@test "aliases.zsh gprj does not point to Apps" {
    ! grep 'alias gprj=' "$ALIASES_FILE" | grep -q 'Apps'
}

# --- Git shortcuts ---

@test "aliases.zsh defines gs alias" {
    grep -q 'alias gs=' "$ALIASES_FILE"
}

@test "aliases.zsh defines gd alias" {
    grep -q 'alias gd=' "$ALIASES_FILE"
}

@test "aliases.zsh defines gl alias" {
    grep -q 'alias gl=' "$ALIASES_FILE"
}

@test "aliases.zsh defines gp alias" {
    grep -q 'alias gp=' "$ALIASES_FILE"
}

# --- Parity: navigation aliases in profile.ps1 ---

@test "parity: gprj points to Projects in both aliases.zsh and profile.ps1" {
    ps_file="$DOTFILES_DIR/powershell/profile.ps1"
    grep 'alias gprj=' "$ALIASES_FILE" | grep -q 'Projects'
    grep 'function gprj' "$ps_file" | grep -q 'Projects'
}

@test "parity: git shortcuts exist in both aliases.zsh and profile.ps1" {
    ps_file="$DOTFILES_DIR/powershell/profile.ps1"
    grep -q 'alias gs=' "$ALIASES_FILE"
    grep -q 'function gs' "$ps_file"
    grep -q 'alias gd=' "$ALIASES_FILE"
    grep -q 'function gd' "$ps_file"
    grep -q 'alias gl=' "$ALIASES_FILE"
    grep -q 'function gl' "$ps_file"
    grep -q 'alias gp=' "$ALIASES_FILE"
    grep -q 'function gp' "$ps_file"
}

@test "parity: cop and cops exist in both aliases.zsh and profile.ps1" {
    ps_file="$DOTFILES_DIR/powershell/profile.ps1"
    grep -qE '^alias cop="copilot"' "$ALIASES_FILE"
    grep -qE 'Set-Alias -Name cop -Value copilot' "$ps_file"
    grep -qE '^cops\(\)' "$ALIASES_FILE"
    grep -qE 'function cops' "$ps_file"
}

@test "parity: qq + qf wrappers exist in aliases.zsh, .bashrc, and profile.ps1" {
    ps_file="$DOTFILES_DIR/powershell/profile.ps1"
    bashrc_file="$DOTFILES_DIR/.bashrc"
    # zsh: aliases wrapping noglob _qq_call
    grep -qE "^alias qq='noglob _qq_call" "$ALIASES_FILE"
    grep -qE "^alias qf='noglob _qq_call" "$ALIASES_FILE"
    # bash: functions delegating to _qq_call
    grep -qE '^qq\(\)' "$bashrc_file"
    grep -qE '^qf\(\)' "$bashrc_file"
    # PowerShell: standalone functions
    grep -qE 'function qq' "$ps_file"
    grep -qE 'function qf' "$ps_file"
    # All three name the same models so behavior matches across shells.
    grep -qF 'opencode-go/qwen3.6-plus' "$ALIASES_FILE"
    grep -qF 'opencode-go/qwen3.6-plus' "$bashrc_file"
    grep -qF 'opencode-go/qwen3.6-plus' "$ps_file"
    grep -qF 'opencode-go/deepseek-v4-flash' "$ALIASES_FILE"
    grep -qF 'opencode-go/deepseek-v4-flash' "$bashrc_file"
    grep -qF 'opencode-go/deepseek-v4-flash' "$ps_file"
}

@test "parity: ghcs/ghce removed from both aliases.zsh and profile.ps1" {
    ps_file="$DOTFILES_DIR/powershell/profile.ps1"
    ! grep -qE '^(alias ghcs|ghcs\(\)|function ghcs)' "$ALIASES_FILE"
    ! grep -qE '^(alias ghce|ghce\(\)|function ghce)' "$ALIASES_FILE"
    ! grep -qE '^\s*function ghcs' "$ps_file"
    ! grep -qE '^\s*function ghce' "$ps_file"
}

# --- tmux aliases ---

@test "aliases.zsh defines tx alias (attach-or-create with -A)" {
    grep -qE "^alias tx=.*new.*-A.*-s" "$ALIASES_FILE"
}

@test "aliases.zsh defines txl, txa, txk aliases" {
    grep -qE "^alias txl=" "$ALIASES_FILE"
    grep -qE "^alias txa=" "$ALIASES_FILE"
    grep -qE "^alias txk=" "$ALIASES_FILE"
}

# --- tmux functions (sshmux) ---

@test "functions.zsh defines sshmux function" {
    fn_file="$DOTFILES_DIR/.zsh/functions.zsh"
    grep -qE "^sshmux\(\)" "$fn_file"
}

@test "sshmux uses ssh -t with remote tmux new-session -A" {
    fn_file="$DOTFILES_DIR/.zsh/functions.zsh"
    grep -qE 'ssh -t.*tmux new-session -A' "$fn_file"
}

@test "sshmux prints usage when no host given" {
    fn_file="$DOTFILES_DIR/.zsh/functions.zsh"
    grep -qE 'Usage: sshmux' "$fn_file"
}
