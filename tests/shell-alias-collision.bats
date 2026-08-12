#!/usr/bin/env bats
# GUARD: function names in the shell files we source must never collide with an
# alias that is live in the user's interactive shell.
#
# Incident (2026-08-04): .zsh/functions.sh defined `gpr()`, but oh-my-zsh's git
# plugin ships `alias gpr='git pull --rebase'`. zsh expands aliases at PARSE
# time, so the definition parsed as `git pull --rebase () {` -> a parse error
# that aborted the REST of the file, silently dropping the utils.sh load at its
# end in every zsh session. It was the second hit on the same rake: the helper
# had already been renamed `gp` -> `gpr` after colliding with `alias gp`.
#
# The pre-existing suite missed it twice over:
#   1. it sourced functions.sh in a bare `zsh -c`, where no oh-my-zsh alias is
#      live, so the collision could not occur during the test; and
#   2. zsh returns exit status 0 after this parse error, so a `[ $status -eq 0 ]`
#      assertion passes on a truncated file.
# These tests therefore assert REACH -- that a definition placed after the
# failure point still resolves -- rather than exit status.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    # Files sourced into an interactive shell; their function names are at risk.
    ZSH_SOURCED="$REPO/.zsh/functions.sh $REPO/.zsh/functions.zsh $REPO/.zsh/nvm.zsh"
}

# Function names defined in the given files, one per line.
function_names() {
    # shellcheck disable=SC2086
    grep -hoE '^[[:space:]]*(function[[:space:]]+)?[a-zA-Z_][a-zA-Z0-9_-]*[[:space:]]*\(\)' $1 2>/dev/null \
        | sed -E 's/^[[:space:]]*(function[[:space:]]+)?//; s/[[:space:]]*\(\)//' \
        | LC_ALL=C sort -u
}

# Alias names defined in the given file, one per line.
alias_names() {
    grep -hoE "^[[:space:]]*alias[[:space:]]+(-g[[:space:]]+)?[a-zA-Z_][a-zA-Z0-9_-]*=" "$1" 2>/dev/null \
        | sed -E 's/^[[:space:]]*alias[[:space:]]+(-g[[:space:]]+)?//; s/=$//' \
        | LC_ALL=C sort -u
}

# A HOME whose .dotfiles/scripts/utils.sh exists, so the load at the very end of
# functions.sh has something to source -- that load is our reach probe.
fake_home() {
    tmph="$(mktemp -d)"
    mkdir -p "$tmph/.dotfiles/scripts"
    cp "$REPO/scripts/utils.sh" "$tmph/.dotfiles/scripts/"
    printf '%s' "$tmph"
}

@test "functions.sh parses to completion in zsh with the historic git aliases live" {
    tmph="$(fake_home)"
    # Pre-define the two aliases this helper has collided with, then source.
    # version_gte comes from utils.sh, loaded by the LAST block of functions.sh,
    # so it resolves only if the whole file parsed.
    run env HOME="$tmph" zsh -c "
        alias gp='git push'
        alias gpr='git pull --rebase'
        . '$REPO/.zsh/functions.sh'
        type version_gte >/dev/null || exit 1
    "
    rm -rf "$tmph"
    [ "$status" -eq 0 ]
    # A parse error is reported on stderr while still exiting 0 -- catch it.
    [[ "$output" != *"parse error"* ]]
    [[ "$output" != *"defining function based on alias"* ]]
}

@test "functions.sh parses to completion in bash with the historic git aliases live" {
    tmph="$(fake_home)"
    run env HOME="$tmph" bash -c "
        shopt -s expand_aliases
        alias gp='git push'
        alias gpr='git pull --rebase'
        . '$REPO/.zsh/functions.sh'
        type version_gte >/dev/null || exit 1
    "
    rm -rf "$tmph"
    [ "$status" -eq 0 ]
}

@test "no sourced function name lands in the g* namespace owned by the git plugin" {
    # oh-my-zsh's git plugin owns ~150 short g-prefixed aliases. Any function we
    # define in that shape is a collision waiting to happen -- both `gp` and
    # `gpr` were. Stay out of the namespace rather than play whack-a-mole.
    #
    # This rule is the fast, hermetic half of the guard: the authoritative
    # cross-check runs against a vendored snapshot of the plugin's alias
    # names below, so it executes everywhere including CI (BUG-045 AC2).
    #
    # `gz` and `getcertnames` predate the rule and are verified collision-free
    # against the plugin by that cross-check (run manually against the real
    # ~/.oh-my-zsh install, BUG-045), so both are grandfathered -- the rule's
    # job is to stop NEW names entering the namespace, not to churn a working
    # helper.
    #
    # BUG-045: the length cap (originally 1-4 chars total) missed the real
    # distribution -- the installed plugin has g-prefixed aliases up to 9
    # chars in the issue's examples and 20 in the wild (`git-svn-dcommit-push`,
    # hyphenated, outside this pattern's letters-only shape). A fixed cap is
    # arbitrary and was already proven too low once. `g|g[a-z]*` is a bash
    # GLOB, not a regex: `g` alone (the plugin ships `alias g='git'`) needs
    # its own alternative, since `g[a-z]*` requires at least one more
    # character; `g[a-z]*` after that is deliberately broader than
    # "all-lowercase" -- it also matches a hyphen, digit or uppercase letter
    # right after the first one (e.g. `git-foo`, `gX9`). Over-matching here is
    # fine because staying out of the namespace is the whole rule; a real
    # helper that needs a g-prefixed name just gets grandfathered like `gz`
    # and `getcertnames` were.
    grandfathered=" gz getcertnames "
    offenders=""
    for name in $(function_names "$ZSH_SOURCED"); do
        case "$grandfathered" in *" $name "*) continue ;; esac
        case "$name" in
            g|g[a-z]*) offenders="$offenders $name" ;;
        esac
    done
    [ -z "$offenders" ] || {
        echo "function names inside the git-plugin namespace:$offenders"
        return 1
    }
}

@test "no sourced function name collides with an alias in our own aliases.zsh" {
    collisions="$(LC_ALL=C comm -12 \
        <(function_names "$ZSH_SOURCED") \
        <(alias_names "$REPO/.zsh/aliases.zsh"))"
    [ -z "$collisions" ] || {
        echo "function/alias collisions with .zsh/aliases.zsh: $collisions"
        return 1
    }
}

@test "no sourced function name collides with the vendored oh-my-zsh git-plugin alias snapshot" {
    # The authoritative cross-check, made to actually run in CI (BUG-045 AC2):
    # oh-my-zsh isn't installed on CI runners, so a check against the live
    # plugin (below) always skipped there. Comparing against a vendored
    # snapshot of just the alias NAMES -- not the plugin file, which is ~400
    # lines of code we don't own -- runs everywhere, always.
    fixture="$REPO/tests/fixtures/omz-git-plugin-aliases.txt"
    collisions="$(LC_ALL=C comm -12 \
        <(function_names "$ZSH_SOURCED") \
        <(grep -v '^#' "$fixture" | LC_ALL=C sort -u))"
    [ -z "$collisions" ] || {
        echo "function/alias collisions with the vendored oh-my-zsh git-plugin snapshot: $collisions"
        return 1
    }
}

@test "the vendored oh-my-zsh git-plugin snapshot is still fresh against the real install" {
    # Staleness detector, not a collision check (the test above owns that,
    # hermetically, against the fixture). Skips where oh-my-zsh isn't
    # installed -- Manu's dev machine is the only place this can run, and
    # it's the signal to re-run tests/fixtures/capture-omz-git-aliases.sh.
    plugin="${ZSH:-$HOME/.oh-my-zsh}/plugins/git/git.plugin.zsh"
    [ -f "$plugin" ] || skip "oh-my-zsh git plugin not installed here"
    fixture="$REPO/tests/fixtures/omz-git-plugin-aliases.txt"
    diff_out="$(diff \
        <(grep -v '^#' "$fixture" | LC_ALL=C sort -u) \
        <(alias_names "$plugin") || true)"
    [ -z "$diff_out" ] || {
        echo "vendored snapshot is stale -- re-run tests/fixtures/capture-omz-git-aliases.sh:"
        echo "$diff_out"
        return 1
    }
}
