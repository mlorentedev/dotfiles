#!/usr/bin/env bats
#
# REFACTOR-010: shell wrapper dedup. The opencode quick-question core
# (_qq_call) and the oc/ocfull wrappers live once in .zsh/functions.sh;
# each rc keeps only its thin shell-specific qq/qf/dbg wrapper.

setup() {
    REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
}

# --- AC1: _qq_call defined exactly once, in functions.sh ---

@test "AC1: _qq_call is defined only in .zsh/functions.sh" {
    run grep -lE '^_qq_call\(\)' "$REPO_ROOT/.zsh/functions.sh" "$REPO_ROOT/.zsh/aliases.zsh" "$REPO_ROOT/.bashrc"
    [ "$status" -eq 0 ]
    [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
    [ "$output" = "$REPO_ROOT/.zsh/functions.sh" ]
}

# --- AC2: oc/ocfull are portable functions in functions.sh, not aliases elsewhere ---

@test "AC2: oc and ocfull are functions in functions.sh" {
    grep -qE '^oc\(\)' "$REPO_ROOT/.zsh/functions.sh"
    grep -qE '^ocfull\(\)' "$REPO_ROOT/.zsh/functions.sh"
}

@test "AC2: no duplicate oc/ocfull aliases remain in aliases.zsh or .bashrc" {
    run grep -nE "alias (oc|ocfull)=" "$REPO_ROOT/.zsh/aliases.zsh" "$REPO_ROOT/.bashrc"
    [ "$status" -ne 0 ]
}

# --- AC3: shell-specific thin wrappers stay in their rc ---

@test "AC3: zsh keeps noglob qq/qf aliases in aliases.zsh" {
    grep -qE "alias qq='noglob _qq_call" "$REPO_ROOT/.zsh/aliases.zsh"
    grep -qE "alias qf='noglob _qq_call" "$REPO_ROOT/.zsh/aliases.zsh"
}

@test "AC3: bash keeps qq/qf functions in .bashrc calling the shared core" {
    grep -qE '^qq\(\) \{ _qq_call' "$REPO_ROOT/.bashrc"
    grep -qE '^qf\(\) \{ _qq_call' "$REPO_ROOT/.bashrc"
}

# --- AC4: dbg uses PATH-resolved nan-debug.sh, no hardcoded path ---

@test "AC4: dbg does not hardcode an absolute nan-debug.sh path" {
    run grep -n "/home/manu/Projects/dotfiles/scripts/nan-debug.sh" "$REPO_ROOT/.zsh/aliases.zsh" "$REPO_ROOT/.bashrc"
    [ "$status" -ne 0 ]
}

# --- AC5: the shared core resolves after sourcing functions.sh, in both shells ---

@test "AC5: _qq_call/oc/ocfull defined after sourcing functions.sh (bash)" {
    run bash -c ". '$REPO_ROOT/.zsh/functions.sh'; type _qq_call >/dev/null && type oc >/dev/null && type ocfull >/dev/null"
    [ "$status" -eq 0 ]
}

@test "AC5: _qq_call/oc/ocfull defined after sourcing functions.sh (zsh)" {
    if ! command -v zsh >/dev/null 2>&1; then skip "zsh not installed"; fi
    run zsh -c ". '$REPO_ROOT/.zsh/functions.sh'; type _qq_call >/dev/null && type oc >/dev/null && type ocfull >/dev/null"
    [ "$status" -eq 0 ]
}
