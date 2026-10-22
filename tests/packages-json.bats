#!/usr/bin/env bats
# packages.json is the pin SSOT for every catalog tool (ADR-036). A name listed
# twice installs twice and every by-name reader takes the first: a duplicated
# copilot entry shipped in a PR (AI-038, #1321) and read as "already installed;
# skipping" on its second line. The Go loader refuses it; this is the same
# invariant at the data layer, so a review sees it before a binary does.

setup() {
    export CATALOG="$BATS_TEST_DIRNAME/../packages.json"
}

mise_catalog_excludes_windows() {
    jq -e 'any(.tools[]; .name == "mise" and
        ((.source.platforms // ["windows"]) | index("windows") == null))' "$1" >/dev/null
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
         then (has("version") | not) and ([.source.apt, .source.brew, .source.cask, .source.winget] | any(type == "string"))
         else (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+")) and (.source.type | IN("npm", "github-release", "uv-tool"))
         end))' "$CATALOG" >/dev/null
}

# A CLI has one owner on each OS. One that versions.conf marks for mise
# (`# mise: cli`) and packages.json also lists for an OS where the catalog
# installs mise is installed twice on a fresh box: setup's first
# `dotf tools install` runs before `dotf tools sync`, so the catalog put an
# unpinned OS copy of zoxide on PATH before mise placed the pinned one (#2195).
# Where the catalog does not install mise (Windows), the catalog is the only
# owner, which is how uv reaches Windows (winget astral-sh.uv, #2754).
@test "no packages.json tool is also a CLI versions.conf marks for mise, on an OS where mise runs" {
    local marked both
    marked="$(awk '
        m && /^[A-Z0-9_]+_VERSION=/ {
            n = $0; sub(/_VERSION=.*/, "", n); n = tolower(n); gsub(/_/, "-", n); print n
        }

        { m = ($0 == "# mise: cli") }' "$BATS_TEST_DIRNAME/../versions.conf")"
    [ -n "$marked" ]
    # An entry's OSes: its platforms, else for a system entry the OSes whose
    # manager it names, else all three. A name owned twice is one whose OSes
    # meet the OSes the catalog installs mise on.
    both="$(jq -r --arg marked "$marked" '
        def oses: if .source.platforms then .source.platforms
            elif .source.type == "system" then
                [(if .source.apt then "linux" else empty end),
                 (if .source.brew or .source.cask then "darwin" else empty end),
                 (if .source.winget then "windows" else empty end)]
            else ["linux", "darwin", "windows"] end;
        ([.tools[] | select(.name == "mise") | oses] | add // []) as $mise
        | ($marked | split("\n")) as $pinned
        | .tools[] | select(.name as $n | $pinned | index($n))
        | select([oses[] | select(. as $o | $mise | index($o))] | length > 0)
        | .name' "$CATALOG")"
    [ -z "$both" ] || { echo "owned by both mise and packages.json: $both"; false; }
}

@test "mise stays disabled on Windows until the reader and full catalog are verified" {
    mise_catalog_excludes_windows "$CATALOG"
}

@test "a release bump alone cannot enable Windows mise without the reader" {
    local fixture="$BATS_TEST_TMPDIR/packages.json"
    printf '%s\n' '{"tools":[{"name":"mise","source":{"platforms":["windows"]}}]}' > "$fixture"
    DOTF_VERSION=0.69.0 run mise_catalog_excludes_windows "$fixture"
    [ "$status" -eq 1 ] || { echo "release pin bypassed the Windows catalog guard"; false; }
}

# A self_install without {version} installs whatever is newest: an unattended
# upgrade (hive ADR-020). cli/internal/tools/catalog.go (validateSelfInstall)
# is the authority; this is the shell-side mirror.
@test "every packages.json self_install passes the pin and belongs to a uv tool" {
    run jq -e '[.tools[] | select(.source.self_install)] as $s
        | ($s | length) > 0
        and all($s[]; .source.type == "uv-tool"
            and (.source.self_install.platforms | length) > 0
            and any(.source.self_install.args[]; contains("{version}")))' "$CATALOG"
    [ "$status" -eq 0 ]
}
