#!/usr/bin/env bats
# Guard (incident->guard, REFACTOR-011): versions.conf is the SOLE version source.
# Fails if an RC file re-introduces a hard-coded ${VAR_VERSION:-X.Y.Z} fallback,
# or references a *_VERSION key that versions.conf does not define (which would
# leave the corresponding *_HOME path empty and silently drop the tool off PATH).

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export VERSIONS_CONF="$DOTFILES_DIR/versions.conf"
}

@test ".zshrc has no hard-coded version fallback literal" {
    refute_grep '\$\{[A-Z_]+_VERSION:-[0-9]' "$DOTFILES_DIR/.zshrc"
}

@test ".bashrc has no hard-coded version fallback literal" {
    refute_grep '\$\{[A-Z_]+_VERSION:-[0-9]' "$DOTFILES_DIR/.bashrc"
}

@test "every *_VERSION referenced in .zshrc is defined in versions.conf" {
    . "$VERSIONS_CONF"
    while IFS= read -r var; do
        [[ -n "${!var}" ]] || { echo "undefined in versions.conf: $var"; return 1; }
    done < <(grep -oE '[A-Z_]+_VERSION' "$DOTFILES_DIR/.zshrc" | sort -u)
}

@test "every *_VERSION referenced in .bashrc is defined in versions.conf" {
    . "$VERSIONS_CONF"
    while IFS= read -r var; do
        [[ -n "${!var}" ]] || { echo "undefined in versions.conf: $var"; return 1; }
    done < <(grep -oE '[A-Z_]+_VERSION' "$DOTFILES_DIR/.bashrc" | sort -u)
}
