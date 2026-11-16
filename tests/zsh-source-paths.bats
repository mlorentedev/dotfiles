#!/usr/bin/env bats

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "zsh requires ./ when sourcing a config from the current directory" {
    local good bad
    good="$(zsh -c "cd '$REPO' && . ./versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null)"
    [ -n "$good" ]

    bad="$(zsh -c "cd '$REPO' && . versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null || true)"
    [ -z "$bad" ]
}
