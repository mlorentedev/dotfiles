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
    # Pathspec exclusions: this file itself names the string as its own
    # search pattern (would otherwise fail on itself), and docs/lessons.md
    # discusses the incident in prose -- a historical mention is not a live
    # dead reference, and the guard's job is to catch the latter.
    run git -C "$DOTFILES_DIR" grep -l "docs/SECRETS.md" -- . \
        ':!tests/sensitive-hygiene.bats' ':!docs/lessons.md'
    [ "$status" -ne 0 ]
}
