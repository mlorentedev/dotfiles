#!/usr/bin/env bats
# Tests for versions.conf format and sourcing

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export VERSIONS_CONF="$DOTFILES_DIR/versions.conf"
}

@test "versions.conf exists" {
    [[ -f "$VERSIONS_CONF" ]]
}

@test "versions.conf valid bash syntax" {
    bash -n "$VERSIONS_CONF"
}

@test "versions.conf valid zsh syntax" {
    zsh -n "$VERSIONS_CONF"
}

@test "versions.conf contains no export statements" {
    refute_grep '^export ' "$VERSIONS_CONF"
}

@test "versions.conf contains no quoted values" {
    refute_grep '^[A-Z_]+=".+"' "$VERSIONS_CONF"
    refute_grep "^[A-Z_]+='.+'" "$VERSIONS_CONF"
}

@test "versions.conf sets JAVA_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$JAVA_VERSION" ]]
}

@test "versions.conf sets MAVEN_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$MAVEN_VERSION" ]]
}

@test "versions.conf sets PYTHON_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$PYTHON_VERSION" ]]
}

@test "versions.conf sets GO_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$GO_VERSION" ]]
}

@test "versions.conf sets BATS_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$BATS_VERSION" ]]
}

@test "versions.conf sets OPENCODE_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$OPENCODE_VERSION" ]]
}

@test "versions.conf sets PI_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$PI_VERSION" ]]
}

@test "versions.conf all values match semver pattern" {
    while IFS= read -r line; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" ]] && continue
        value="${line#*=}"
        [[ "$value" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    done < "$VERSIONS_CONF"
}

@test "versions.conf sourceable under zsh" {
    run zsh -c ". '$VERSIONS_CONF'; echo \$JAVA_VERSION"
    [[ $status -eq 0 ]]
    [[ -n "$output" ]]
}

@test "versions.conf sets GOLANGCI_LINT_VERSION [#919]" {
    . "$VERSIONS_CONF"
    [[ -n "$GOLANGCI_LINT_VERSION" ]]
}

@test "cli.yml pins golangci-lint from versions.conf, never the action default [#919]" {
    local wf="$DOTFILES_DIR/.github/workflows/cli.yml"
    # The action resolves 'latest' when no version is given, which is what made
    # a local run unable to reproduce CI (BUG-071). Assert the pin is both
    # sourced from versions.conf and handed to the action.
    grep -q 'GOLANGCI_LINT_VERSION' "$wf"
    grep -q 'version: \${{ steps.golangci.outputs.version }}' "$wf"
}

@test "cli.yml fails loudly when the golangci-lint pin is missing [#919]" {
    local wf="$DOTFILES_DIR/.github/workflows/cli.yml"
    # A silently-empty version input falls back to 'latest', reintroducing the
    # bug while looking green. The resolve step must exit non-zero instead.
    grep -q 'GOLANGCI_LINT_VERSION missing from versions.conf' "$wf"
}
