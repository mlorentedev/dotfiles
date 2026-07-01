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

# ---- Age key discovery: AGE_KEY_PATH + SOPS_AGE_KEY_FILE (CLI-024 #518) ----
# The age key is the ADR-028 DR root-of-trust. Declaring its path in the
# contract makes `dotf env generate` export it into paths.{sh,ps1} so every
# shell (and any ad-hoc `sops` op, the literal #518 bug) discovers the key with
# zero per-shell config. Unlike the structural path vars above, these carry NO
# path_exists validation: a freshly provisioned box legitimately has no key
# until it is restored offline, and the WARN-level doctor presence check — not a
# hard env-contract FAIL — owns "key missing".

@test "env-contract.json declares AGE_KEY_PATH" {
    run jq -e '.env_vars[] | select(.name == "AGE_KEY_PATH")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "env-contract.json declares SOPS_AGE_KEY_FILE" {
    run jq -e '.env_vars[] | select(.name == "SOPS_AGE_KEY_FILE")' "$CONTRACT"
    [ "$status" -eq 0 ]
}

@test "each age key var resolves to age/key.txt on both OSes" {
    for name in AGE_KEY_PATH SOPS_AGE_KEY_FILE; do
        linux=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.linux" "$CONTRACT")
        windows=$(jq -r ".env_vars[] | select(.name == \"$name\") | .default.windows" "$CONTRACT")
        [[ "$linux" == *"/.config/age/key.txt" ]]
        [[ "$windows" == *'\.config\age\key.txt' ]]
    done
}

@test "each age key var is optional (required:false)" {
    for name in AGE_KEY_PATH SOPS_AGE_KEY_FILE; do
        required=$(jq -r ".env_vars[] | select(.name == \"$name\") | .required" "$CONTRACT")
        [ "$required" = "false" ]
    done
}

@test "no age key var carries path_exists validation (fresh box must not FAIL)" {
    for name in AGE_KEY_PATH SOPS_AGE_KEY_FILE; do
        has_validation=$(jq -r ".env_vars[] | select(.name == \"$name\") | has(\"validation\")" "$CONTRACT")
        [ "$has_validation" = "false" ]
    done
}

# The generic `dotf init` starter template must stay repo-agnostic: the age key
# is a dotfiles machine fact, not a universal per-repo expectation, so it must
# NOT leak into every onboarded repo's seed contract.
@test "dotf init starter template does not declare age key vars" {
    local tmpl="$DOTFILES_DIR/cli/internal/initrepo/templates/env-contract.json"
    run jq -e '.env_vars | map(.name) | any(. == "AGE_KEY_PATH" or . == "SOPS_AGE_KEY_FILE")' "$tmpl"
    [ "$status" -ne 0 ]
}

# ---- RC parity: .zshrc + .bashrc + powershell/profile.ps1 -----------------

@test ".zshrc sources the generated path file and declares the 4 path vars" {
    # ADR-025: paths.sh (rendered by `dotf env generate`) is the primary source;
    # the 4 vars also appear in the bootstrap fallback (indented in the else
    # branch), so match the export anywhere on the line, not only at column 0.
    grep -qE 'paths\.sh' "$ZSHRC"
    grep -qE 'export SCRIPTS_DIR=' "$ZSHRC"
    grep -qE 'export AGY_HOME=' "$ZSHRC"
    grep -qE 'export COPILOT_HOME=' "$ZSHRC"
    grep -qE 'export OPENCODE_HOME=' "$ZSHRC"
}

@test ".bashrc sources the generated path file and declares the 4 path vars" {
    # ADR-025: see the .zshrc test above — primary source is paths.sh, the
    # inline exports are the indented bootstrap fallback.
    grep -qE 'paths\.sh' "$BASHRC"
    grep -qE 'export SCRIPTS_DIR=' "$BASHRC"
    grep -qE 'export AGY_HOME=' "$BASHRC"
    grep -qE 'export COPILOT_HOME=' "$BASHRC"
    grep -qE 'export OPENCODE_HOME=' "$BASHRC"
}

@test "powershell/profile.ps1 exports the 4 new path vars" {
    grep -qE 'paths\.ps1' "$PS_PROFILE"
    grep -qE '\$env:SCRIPTS_DIR\s*=' "$PS_PROFILE"
    grep -qE '\$env:AGY_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:COPILOT_HOME\s*=' "$PS_PROFILE"
    grep -qE '\$env:OPENCODE_HOME\s*=' "$PS_PROFILE"
}

@test ".zshrc and .bashrc agree on the 4 new exports (parity)" {
    for var in SCRIPTS_DIR AGY_HOME COPILOT_HOME OPENCODE_HOME; do
        # ADR-025: the exports live in the indented bootstrap fallback, so allow
        # leading whitespace when matching and stripping the `export VAR=` prefix.
        zsh_val=$(grep -E "export ${var}=" "$ZSHRC" | head -1 | sed -E "s|^[[:space:]]*export ${var}=||" | tr -d '"')
        bash_val=$(grep -E "export ${var}=" "$BASHRC" | head -1 | sed -E "s|^[[:space:]]*export ${var}=||" | tr -d '"')
        [ -n "$zsh_val" ]
        [ "$zsh_val" = "$bash_val" ]
    done
}

# Post-setup false-warning fix: setup-linux.sh must pre-export the 4 REFACTOR-002
# path vars before invoking `dotf doctor`, otherwise the running shell (which
# hasn't re-sourced the new RC yet) sees them as unset and the user gets cosmetic
# warnings on every fresh setup run.
@test "setup-linux.sh pre-exports the 4 path vars before post-setup dotf doctor" {
    local setup="$DOTFILES_DIR/setup-linux.sh"
    # The pre-export block lives immediately before the `dotf doctor` invocation.
    # Extract that window, then grep within it.
    local window
    window=$(awk '/Pre-export the REFACTOR-002 path vars/,/command -v dotf/' "$setup")
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
