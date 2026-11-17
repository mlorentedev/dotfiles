#!/usr/bin/env bats

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "zsh requires ./ when sourcing a config from the current directory" {
    # `. versions.conf` resolves to an empty version under zsh: a slashless
    # argument to `.` is searched on $PATH only, and unlike bash, zsh does not
    # fall back to the cwd (#916). Pin the working form, and pin that the broken
    # one really is broken. Kept from the retired check-doc-paths suite; the
    # instruction-file side of this rule is `dotf lint doc-paths` (bare-source).
    local good bad
    good="$(zsh -c "cd '$REPO' && . ./versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null)"
    [ -n "$good" ]

    bad="$(zsh -c "cd '$REPO' && . versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null || true)"
    [ -z "$bad" ]
}
