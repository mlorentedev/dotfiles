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
# comment except "# mise: cli", so "# mise: python-package" turns that
# machine's `dotf tools sync` and doctor into a parse error. Marking
# PYTHON_VERSION under such a binary is no safer: it installs mise's python
# without its packages, and that shim shadows the system python the suite
# imports yaml from (#2062, lesson 373). v0.66.0 is the first release that
# reads both, so DOTF_VERSION must not drop below it while either is marked.
@test "versions.conf marks python for mise only while DOTF_VERSION parses the python-package marker" {
    # The released parser trims each line, so an indented marker counts the
    # same; the version compare is numeric per field (0.100.0 > 0.66.0).
    run awk -v floor="0.66.0" '
        function ge(a, b,   x, y, i) {
            split(a, x, "."); split(b, y, ".")
            for (i = 1; i <= 3; i++) if (x[i] + 0 != y[i] + 0) return x[i] + 0 > y[i] + 0
            return 1
        }
        marked { marked = 0; if ($0 ~ /^[[:space:]]*PYTHON_VERSION[[:space:]]*=/) uses = "PYTHON_VERSION is marked" }
        {
            t = $0; sub(/^[[:space:]]+/, "", t); sub(/[[:space:]]+$/, "", t)
            if (t == "# mise: cli") marked = 1
            if (t == "# mise: python-package") uses = "a python-package marker"
        }
        /^DOTF_VERSION=/ { dotf = $0; sub(/^DOTF_VERSION=/, "", dotf) }
        END {
            if (dotf == "") { print "no DOTF_VERSION"; exit 1 }
            if (uses != "" && !ge(dotf, floor)) { print uses " but DOTF_VERSION=" dotf " is below " floor; exit 1 }
        }' "$VERSIONS_CONF"
    [[ "$status" -eq 0 ]] || { printf '%s\n' "$output"; false; }
}

# "# mise: latest" (#2013 D9) is a third marker, and every release up to
# v0.67.0 rejects it as a near miss: the whole tools step of converge, and
# doctor, fail on that machine. The first release carrying the parser is the
# floor. If 0.68.0 ships without it, raise the floor to the release that does.
@test "versions.conf uses the latest marker only while DOTF_VERSION parses it" {
    run awk -v floor="0.68.0" '
        function ge(a, b,   x, y, i) {
            split(a, x, "."); split(b, y, ".")
            for (i = 1; i <= 3; i++) if (x[i] + 0 != y[i] + 0) return x[i] + 0 > y[i] + 0
            return 1
        }
        {
            t = $0; sub(/^[[:space:]]+/, "", t); sub(/[[:space:]]+$/, "", t)
            if (t == "# mise: latest") uses = 1
        }
        /^DOTF_VERSION=/ { dotf = $0; sub(/^DOTF_VERSION=/, "", dotf) }
        END {
            if (dotf == "") { print "no DOTF_VERSION"; exit 1 }
            if (uses && !ge(dotf, floor)) { print "a \"# mise: latest\" marker but DOTF_VERSION=" dotf " is below " floor; exit 1 }
        }' "$VERSIONS_CONF"
    [[ "$status" -eq 0 ]] || { printf '%s\n' "$output"; false; }
}

# Python is a hard dependency of the toolchain (#2062): the suite imports
# tomllib and yaml, and doctor fails without them. Unmarking either line would
# leave a machine's python to whatever the OS ships (3.9 on macOS).
@test "versions.conf declares python and pyyaml for mise (#2062)" {
    run awk '
        m == "cli" && /^PYTHON_VERSION=/ { py = 1 }
        m == "pkg" && /^PYYAML_VERSION=/ { yaml = 1 }
        { m = ($0 == "# mise: cli") ? "cli" : ($0 == "# mise: python-package") ? "pkg" : "" }
        END {
            if (!py) print "PYTHON_VERSION is not under \"# mise: cli\""
            if (!yaml) print "PYYAML_VERSION is not under \"# mise: python-package\""
            exit !(py && yaml)
        }' "$VERSIONS_CONF"
    [[ "$status" -eq 0 ]] || { printf '%s\n' "$output"; false; }
}

# The released dotf probes a marked CLI with `<tool> --version`, and these
# seven reject that flag. Up to v0.66.0 no release knows another way to ask
# them, so marking one turns every machine's `dotf tools sync` into "not
# running at their pin" and doctor red, while CI, which builds dotf from the
# tree, stays green (lesson 373). The tree's dotf asks them the right way
# (versionArgs in cli/internal/tools/mise.go); each may be marked once
# DOTF_VERSION is a release carrying it (#2013). `last` is the newest release
# without the table: if a release ships before versionArgs merges, raise it to
# that release, and check a release's notes name versionArgs before marking.
@test "versions.conf marks no CLI without --version while DOTF_VERSION probes only --version" {
    run awk -v last="0.66.0" '
        function gt(a, b,   x, y, i) {
            split(a, x, "."); split(b, y, ".")
            for (i = 1; i <= 3; i++) if (x[i] + 0 != y[i] + 0) return x[i] + 0 > y[i] + 0
            return 0
        }
        marked && /^[[:space:]]*(ARGOCD|HELM|HCLOUD|K9S|KUBECONFORM|KUBECTL|KUSTOMIZE)_VERSION[[:space:]]*=/ {
            n = $0; sub(/^[[:space:]]*/, "", n); sub(/_VERSION.*/, "", n); uses = uses " " n
        }
        { t = $0; sub(/^[[:space:]]+/, "", t); sub(/[[:space:]]+$/, "", t); marked = (t == "# mise: cli") }
        /^DOTF_VERSION=/ { dotf = $0; sub(/^DOTF_VERSION=/, "", dotf) }
        END {
            if (dotf == "") { print "no DOTF_VERSION"; exit 1 }
            if (uses != "" && !gt(dotf, last)) { print "marked for mise:" uses ", but DOTF_VERSION=" dotf " probes them with --version"; exit 1 }
        }' "$VERSIONS_CONF"
    [[ "$status" -eq 0 ]] || { printf '%s\n' "$output"; false; }
}
