#!/usr/bin/env bats
# Tests for env-contract.json + RC-file exports parity (REFACTOR-002).

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    CONTRACT="$DOTFILES_DIR/env-contract.json"
    ZSHRC="$DOTFILES_DIR/.zshrc"
    BASHRC="$DOTFILES_DIR/.bashrc"
    PS_PROFILE="$DOTFILES_DIR/powershell/profile.ps1"
}

@test "env-contract.json is valid JSON" {
    run jq -e . "$CONTRACT"
    [ "$status" -eq 0 ]
}

# ---- Contract: 4 new entries (REFACTOR-002) -------------------------------

@test "env-contract.json declares SCRIPTS_DIR" {
    run jq -e '.env_vars[] | select(.name == "SCRIPTS_DIR")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "env-contract.json declares GEMINI_HOME" {
    run jq -e '.env_vars[] | select(.name == "GEMINI_HOME")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "env-contract.json declares COPILOT_HOME" {
    run jq -e '.env_vars[] | select(.name == "COPILOT_HOME")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "env-contract.json declares OPENCODE_HOME" {
    run jq -e '.env_vars[] | select(.name == "OPENCODE_HOME")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "each new var has both linux and windows defaults" {
    for name in SCRIPTS_DIR GEMINI_HOME COPILOT_HOME OPENCODE_HOME; do
        linux=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.linux" "$CONTRACT")
        windows=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.windows" "$CONTRACT")
        [ -n "$linux" ] && [ "$linux" != "null" ]
        [ -n "$windows" ] && [ "$windows" != "null" ]
    done
}

@test "each new var validation is path_exists" {
    for name in SCRIPTS_DIR GEMINI_HOME COPILOT_HOME OPENCODE_HOME; do
        validation=$(jq -r ".env_vars[] | select(.name == \"$name\") | .validation" "$CONTRACT")
        [ "$validation" = "path_exists" ]
    done
}

# ---- RC parity: .zshrc + .bashrc + powershell/profile.ps1 -----------------

@test ".zshrc exports the 4 new path vars" {
    grep -qE '^export SCRIPTS_DIR=' "$ZSHRC"
    grep -qE '^export GEMINI_HOME=' "$ZSHRC"
    grep -qE '^export COPILOT_HOME=' "$ZSHRC"
    grep -qE '^export OPENCODE_HOME=' "$ZSHRC"
}

@test ".bashrc exports the 4 new path vars" {
    grep -qE '^export SCRIPTS_DIR=' "$BASHRC"
    grep -qE '^export GEMINI_HOME=' "$BASHRC"
    grep -qE '^export COPILOT_HOME=' "$BASHRC"
    grep -qE '^export OPENCODE_HOME=' "$BASHRC"
}

@test "powershell/profile.ps1 exports the 4 new path vars" {
    grep -qE '\$env:SCRIPTS_DIR\s*=' "$PS_PROFILE"
    grep -qE '\$env:GEMINI_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:COPILOT_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:OPENCODE_HOME\s*=' "$PS_PROFILE"
}

@test ".zshrc and .bashrc agree on the 4 new exports (parity)" {
    for var in SCRIPTS_DIR GEMINI_HOME COPILOT_HOME OPENCODE_HOME; do
        zsh_val=$(grep -E "^export ${var}=" "$ZSHRC" | head -1 | sed -E "s|^export ${var}=||" | tr -d '"')
        bash_val=$(grep -E "^export ${var}=" "$BASHRC" | head -1 | sed -E "s|^export ${var}=||" | tr -d '"')
        [ -n "$zsh_val" ]
        [ "$zsh_val" = "$bash_val" ]
    done
}
