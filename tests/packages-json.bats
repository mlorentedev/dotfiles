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
