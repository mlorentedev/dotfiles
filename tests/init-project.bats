#!/usr/bin/env bats
# Tests for scripts/init-project.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export TEST_DIR="/tmp/bats_init_$$"
    export MOCK_HOME="/tmp/bats_home_$$"
    mkdir -p "$TEST_DIR"
    mkdir -p "$MOCK_HOME"
}

teardown() {
    rm -rf "$TEST_DIR"
    rm -rf "$MOCK_HOME"
}

@test "init-project.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh creates project structure" {
    run bash -c 'export HOME="$3"; cd "$1" && bash "$2/init-project.sh" . none 2>&1' -- "$TEST_DIR" "$SCRIPTS_DIR" "$MOCK_HOME"
    [[ -d "$TEST_DIR/src" ]]
    [[ -d "$TEST_DIR/tests" ]]
    [[ ! -d "$TEST_DIR/tasks" ]]
    
    local kb_dir="$MOCK_HOME/Projects/knowledge/10_projects/$(basename "$TEST_DIR")"
    [[ -d "$kb_dir" ]]
    [[ -f "$kb_dir/11-tasks.md" ]]
    [[ -f "$kb_dir/90-lessons.md" ]]
}

@test "init-project.sh creates structure under zsh" {
    local zdir="/tmp/bats_init_zsh_$$"
    mkdir -p "$zdir"
    run zsh -c 'export HOME="$3"; cd "$1" && zsh "$2/init-project.sh" . none 2>&1' -- "$zdir" "$SCRIPTS_DIR" "$MOCK_HOME"
    [[ -d "$zdir/src" ]]
    [[ ! -d "$zdir/tasks" ]]

    local kb_dir="$MOCK_HOME/Projects/knowledge/10_projects/$(basename "$zdir")"
    [[ -d "$kb_dir" ]]
    rm -rf "$zdir"
}

# --- REFACTOR-004: init-repo-* wiring into init-project default flow ---
# These structural asserts lock the wiring contract: the 3 skip flags exist
# in the arg parser, the 3 helper invocations exist after the .gitignore
# block, the github-defaults invocation is guarded by an origin-remote check,
# and the PowerShell side has parity switches (-SkipAgents/-SkipStandards/
# -SkipGithub). Functional tests of the helper invocations themselves require
# a vault + gh auth state we don't reproduce in bats; relying on the helpers'
# own bats coverage for that.

@test "REFACTOR-004: init-project.sh arg parser recognizes --skip-agents" {
    grep -qF -- '--skip-agents) SKIP_AGENTS=1' "$SCRIPTS_DIR/init-project.sh"
}

@test "REFACTOR-004: init-project.sh arg parser recognizes --skip-standards" {
    grep -qF -- '--skip-standards) SKIP_STANDARDS=1' "$SCRIPTS_DIR/init-project.sh"
}

@test "REFACTOR-004: init-project.sh arg parser recognizes --skip-github" {
    grep -qF -- '--skip-github) SKIP_GITHUB=1' "$SCRIPTS_DIR/init-project.sh"
}

@test "REFACTOR-004: init-project.sh invokes all 3 init-repo-* helpers" {
    grep -qF 'init-repo-agents.sh' "$SCRIPTS_DIR/init-project.sh"
    grep -qF 'init-repo-standards.sh' "$SCRIPTS_DIR/init-project.sh"
    grep -qF 'init-repo-github-defaults.sh' "$SCRIPTS_DIR/init-project.sh"
}

@test "REFACTOR-004: github-defaults invocation guarded by origin remote check" {
    grep -qF 'git config --get remote.origin.url' "$SCRIPTS_DIR/init-project.sh"
}

@test "REFACTOR-004: helper failures are non-fatal (log_error ... continuing)" {
    # Each helper invocation must end with `|| log_error ... continuing` so a
    # single helper failure doesn't abort the whole init.
    [ "$(grep -c 'log_error "init-repo.* (non-fatal); continuing"' "$SCRIPTS_DIR/init-project.sh")" -eq 3 ]
}

@test "parity: init-project.ps1 declares -SkipAgents/-SkipStandards/-SkipGithub switches" {
    grep -qF '[switch]$SkipAgents' "$SCRIPTS_DIR/init-project.ps1"
    grep -qF '[switch]$SkipStandards' "$SCRIPTS_DIR/init-project.ps1"
    grep -qF '[switch]$SkipGithub' "$SCRIPTS_DIR/init-project.ps1"
}

@test "parity: init-project.ps1 invokes all 3 init-repo-* helpers" {
    grep -qF 'init-repo-agents.ps1' "$SCRIPTS_DIR/init-project.ps1"
    grep -qF 'init-repo-standards.ps1' "$SCRIPTS_DIR/init-project.ps1"
    grep -qF 'init-repo-github-defaults.ps1' "$SCRIPTS_DIR/init-project.ps1"
}

@test "parity: init-project.ps1 github invocation guarded by origin remote check" {
    grep -qF 'remote.origin.url' "$SCRIPTS_DIR/init-project.ps1"
}
