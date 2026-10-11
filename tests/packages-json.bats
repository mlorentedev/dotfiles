#!/usr/bin/env bats
# packages.json is the pin SSOT for every catalog tool (ADR-036). A name listed
# twice installs twice and every by-name reader takes the first: a duplicated
# copilot entry shipped in a PR (AI-038, #1321) and read as "already installed;
# skipping" on its second line. The Go loader refuses it; this is the same
# invariant at the data layer, so a review sees it before a binary does.

setup() {
    export CATALOG="$BATS_TEST_DIRNAME/../packages.json"
}

@test "packages.json is valid JSON with a tools array" {
    jq -e '.tools | type == "array"' "$CATALOG" >/dev/null
}

@test "packages.json tool names are unique" {
    total=$(jq '[.tools[].name] | length' "$CATALOG")
    unique=$(jq '[.tools[].name] | unique | length' "$CATALOG")
    [ "$total" -eq "$unique" ]
}

# A "system" entry is the OS package manager's, so it carries no version and
# names at least one manager's package; cli/internal/tools/catalog.go
# (validateSystem) is the authority, this is the cheap shell-side mirror.
@test "every packages.json tool declares name, version, profile and a typed source" {
    jq -e 'all(.tools[]; (.name | type == "string") and (.profile | type == "string") and
        (if .source.type == "system"
         then (has("version") | not) and ([.source.apt, .source.brew, .source.winget] | any(type == "string"))
         else (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+")) and (.source.type | IN("npm", "github-release", "uv-tool"))
         end))' "$CATALOG" >/dev/null
}

# A CLI has one owner. One that versions.conf marks for mise (`# mise: cli`) and
# packages.json also lists is installed twice on a fresh box: setup's first
# `dotf tools install` runs before `dotf tools sync`, so the catalog put an
# unpinned OS copy of zoxide on PATH before mise placed the pinned one (#2195).
@test "no packages.json tool is also a CLI versions.conf marks for mise" {
    local marked both
    marked="$(awk '
        m && /^[A-Z0-9_]+_VERSION=/ {
            n = $0; sub(/_VERSION=.*/, "", n); n = tolower(n); gsub(/_/, "-", n); print n
        }
        { m = ($0 == "# mise: cli") }' "$BATS_TEST_DIRNAME/../versions.conf" | sort)"
    [ -n "$marked" ]
    both="$(jq -r '.tools[].name' "$CATALOG" | sort | comm -12 - <(printf '%s\n' "$marked"))"
    [ -z "$both" ] || { echo "owned by both mise and packages.json: $both"; false; }
}
