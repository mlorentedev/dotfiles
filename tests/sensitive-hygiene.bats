#!/usr/bin/env bats
# GUARD-002 (#669): sensitive/env-mapping.conf was retired twice (#587/#601,
# then silently resurrected by #659) with no guard against a third comeback.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "sensitive/env-mapping.conf does not exist (retired; registry.yaml is the mapping SSOT)" {
    [ ! -f "$DOTFILES_DIR/sensitive/env-mapping.conf" ]
}

@test "no tracked file references the nonexistent docs/SECRETS.md" {
    # Pathspec excludes this file itself -- it names the string as its own
    # search pattern, which would otherwise make the guard fail on itself.
    run git -C "$DOTFILES_DIR" grep -l "docs/SECRETS.md" -- . ':!tests/sensitive-hygiene.bats'
    [ "$status" -ne 0 ]
}
