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

@test "env-contract.json declares AGY_HOME" {
    run jq -e '.env_vars[] | select(.name == "AGY_HOME")' "$CONTRACT"
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
    for name in SCRIPTS_DIR AGY_HOME COPILOT_HOME OPENCODE_HOME; do
        linux=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.linux" "$CONTRACT")
        windows=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.windows" "$CONTRACT")
        [ -n "$linux" ] && [ "$linux" != "null" ]
        [ -n "$windows" ] && [ "$windows" != "null" ]
    done
}

@test "each new var validation is path_exists" {
    for name in SCRIPTS_DIR AGY_HOME COPILOT_HOME OPENCODE_HOME; do
        validation=$(jq -r ".env_vars[] | select(.name == \"$name\") | .validation" "$CONTRACT")
        [ "$validation" = "path_exists" ]
    done
}

# ---- RC parity: .zshrc + .bashrc + powershell/profile.ps1 -----------------

@test ".zshrc exports the 4 new path vars" {
    grep -qE '^export SCRIPTS_DIR=' "$ZSHRC"
    grep -qE '^export AGY_HOME=' "$ZSHRC"
    grep -qE '^export COPILOT_HOME=' "$ZSHRC"
    grep -qE '^export OPENCODE_HOME=' "$ZSHRC"
}

@test ".bashrc exports the 4 new path vars" {
    grep -qE '^export SCRIPTS_DIR=' "$BASHRC"
    grep -qE '^export AGY_HOME=' "$BASHRC"
    grep -qE '^export COPILOT_HOME=' "$BASHRC"
    grep -qE '^export OPENCODE_HOME=' "$BASHRC"
}

@test "powershell/profile.ps1 exports the 4 new path vars" {
    grep -qE '\$env:SCRIPTS_DIR\s*=' "$PS_PROFILE"
    grep -qE '\$env:AGY_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:COPILOT_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:OPENCODE_HOME\s*=' "$PS_PROFILE"
}

@test ".zshrc and .bashrc agree on the 4 new exports (parity)" {
    for var in SCRIPTS_DIR AGY_HOME COPILOT_HOME OPENCODE_HOME; do
        zsh_val=$(grep -E "^export ${var}=" "$ZSHRC" | head -1 | sed -E "s|^export ${var}=||" | tr -d '"')
        bash_val=$(grep -E "^export ${var}=" "$BASHRC" | head -1 | sed -E "s|^export ${var}=||" | tr -d '"')
        [ -n "$zsh_val" ]
        [ "$zsh_val" = "$bash_val" ]
    done
}

# Post-setup doctor false-warning fix: setup-linux.sh must pre-export the 4
# REFACTOR-002 path vars before invoking doctor, otherwise the running shell
# (which hasn't re-sourced the new RC yet) sees them as unset and the user
# gets 4 cosmetic warnings on every fresh setup run.
@test "setup-linux.sh pre-exports the 4 path vars before post-setup doctor check" {
    local setup="$DOTFILES_DIR/setup-linux.sh"
    # The pre-export block lives immediately before the DOCTOR_SCRIPT line.
    # Use a single sed expression to extract that window, then grep within it.
    local window
    window=$(awk '/Pre-export the REFACTOR-002 path vars/,/^DOCTOR_SCRIPT=/' "$setup")
    [[ "$window" == *"export SCRIPTS_DIR="* ]]
    [[ "$window" == *"export AGY_HOME="* ]]
    [[ "$window" == *"export COPILOT_HOME="* ]]
    [[ "$window" == *"export OPENCODE_HOME="* ]]
}

@test "setup-windows.ps1 pre-exports the 4 path vars before post-setup doctor check" {
    local setup="$DOTFILES_DIR/setup-windows.ps1"
    # Same bug class as setup-linux.sh, but `& pwsh -NoProfile` skips the
    # deployed profile so the child process doesn't see the new vars.
    # Cross-OS parity: same fix shape, PowerShell syntax.
    local window
    window=$(awk '/Pre-export the REFACTOR-002 path vars/,/\$doctorScript = /' "$setup")
    [[ "$window" == *"\$env:SCRIPTS_DIR"* ]]
    [[ "$window" == *"\$env:AGY_HOME"* ]]
    [[ "$window" == *"\$env:COPILOT_HOME"* ]]
    [[ "$window" == *"\$env:OPENCODE_HOME"* ]]
}
