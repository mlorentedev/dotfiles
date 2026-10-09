#!/usr/bin/env bats
# Tests for versions.conf format and sourcing

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export VERSIONS_CONF="$DOTFILES_DIR/versions.conf"
}

@test "versions.conf exists" {
    [[ -f "$VERSIONS_CONF" ]] || false
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
    [[ -n "$JAVA_VERSION" ]] || false
}

@test "versions.conf sets MAVEN_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$MAVEN_VERSION" ]] || false
}

@test "versions.conf sets PYTHON_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$PYTHON_VERSION" ]] || false
}

@test "versions.conf sets GO_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$GO_VERSION" ]] || false
}

@test "versions.conf sets BATS_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$BATS_VERSION" ]] || false
}

@test "versions.conf does not pin opencode (packages.json is the catalog pin SSOT, ADR-036)" {
    . "$VERSIONS_CONF"
    [[ -z "${OPENCODE_VERSION:-}" ]] || false
}

@test "versions.conf sets PI_VERSION" {
    . "$VERSIONS_CONF"
    [[ -n "$PI_VERSION" ]] || false
}

@test "versions.conf sets GIT_VERSION (git-for-windows floor read by dotf doctor, #912)" {
    # The floor, not merely "set": a manifest carrying 2.54.0 would still pass a
    # non-empty check while doctor's WARN names 2.55.0 (CodeRabbit on #1350).
    unset GIT_VERSION
    . "$VERSIONS_CONF"
    [[ "$GIT_VERSION" == "2.55.0" ]] || false
}

@test "versions.conf all values match semver pattern" {
    while IFS= read -r line; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" ]] && continue
        value="${line#*=}"
        [[ "$value" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || false
    done < "$VERSIONS_CONF"
}

@test "versions.conf sourceable under zsh" {
    run zsh -c ". '$VERSIONS_CONF'; echo \$JAVA_VERSION"
    [[ $status -eq 0 ]] || false
    [[ -n "$output" ]] || false
}

@test "versions.conf sets GOLANGCI_LINT_VERSION [#919]" {
    . "$VERSIONS_CONF"
    [[ -n "$GOLANGCI_LINT_VERSION" ]] || false
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

# The released dotf reads this file too: setup and doctor run DOTF_VERSION's
# binary, not this tree's. Up to v0.65.0 its parser rejects every "# mise:"
# comment except "# mise: cli", so "# mise: python-package" would turn every
# machine's `dotf tools sync` and doctor into a parse error until a release
# carrying ParseMisePins is the pinned one (#2062). Marking PYTHON_VERSION
# early is no safer: that binary would install mise's python without its
# packages, and its shim would shadow the system python the suite imports yaml
# from. Relax this in the PR that marks both, once DOTF_VERSION carries the
# parser.
@test "versions.conf marks python for mise only once DOTF_VERSION parses the python-package marker" {
    # The released parser trims each line and matches markers case-blind, so
    # an indented or differently cased marker counts the same.
    run awk '
        marked { marked = 0; if ($0 ~ /^[[:space:]]*PYTHON_VERSION[[:space:]]*=/) { print NR ": PYTHON_VERSION is marked"; bad = 1 } }
        tolower($0) ~ /^[[:space:]]*#[[:space:]]*mise[[:space:]]*:/ {
            t = $0; sub(/^[[:space:]]+/, "", t); sub(/[[:space:]]+$/, "", t)
            if (t != "# mise: cli") { print NR ": " t; bad = 1 } else { marked = 1 }
        }
        END { exit bad }' "$VERSIONS_CONF"
    [[ "$status" -eq 0 ]] || { printf '%s\n' "$output"; false; }
}
