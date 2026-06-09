#!/usr/bin/env bats
# Tests for versions.conf format and sourcing

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
    ! grep -q '^export ' "$VERSIONS_CONF"
}

@test "versions.conf contains no quoted values" {
    ! grep -qE '^[A-Z_]+=".+"' "$VERSIONS_CONF"
    ! grep -qE "^[A-Z_]+='.+'" "$VERSIONS_CONF"
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
