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
         then (has("version") | not) and ([.source.apt, .source.brew, .source.cask, .source.winget] | any(type == "string"))
         else (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+")) and (.source.type | IN("npm", "github-release", "github-archive", "uv-tool"))
         end))' "$CATALOG" >/dev/null
}

# A CLI has one owner on each OS. One that versions.conf marks for mise
# (`# mise: cli`) and packages.json also lists for an OS where the catalog
# installs mise is installed twice on a fresh box: setup's first
# `dotf tools install` runs before `dotf tools sync`, so the catalog put an
# unpinned OS copy of zoxide on PATH before mise placed the pinned one (#2195).
# The catalog installs mise on every OS (PLAT-001c), so a marked CLI has no
# catalog entry anywhere: uv reaches Windows through mise too.
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

# Windows mise renders bats as npm:bats and direnv from its GitHub release;
# a dotf older than that reader renders the aqua channels, which install
# nothing usable there, so the catalog may not reach Windows behind it
# (007c92fc, first released in 0.68.0).
@test "mise is in the catalog on every OS, and the dotf pin carries the Windows reader" {
    jq -e 'any(.tools[]; .name == "mise" and .source.platforms == null
        and .source.asset["windows/amd64"] and .source.asset["windows/arm64"])' "$CATALOG" >/dev/null
    local pin
    pin="$(sed -n 's/^DOTF_VERSION=//p' "$BATS_TEST_DIRNAME/../versions.conf")"
    [ -n "$pin" ]
    awk -v v="$pin" 'BEGIN { split(v, p, "."); exit !(p[1] > 0 || p[2] >= 68) }'
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

# uv resolves hive's dependencies before hive runs, so hive's own guard
# against a cryptography source build (hive#523) cannot reach the bootstrap.
# Without no_build, a fresh Windows Arm64 machine fails at link.exe and ends
# with no hive (AI-028 V6).
@test "hive's Windows self install never builds cryptography from source" {
    run jq -e '.tools[] | select(.name == "hive") | .source.self_install
        | (.platforms | index("windows")) != null
        and (.no_build | index("cryptography")) != null' "$CATALOG"
    [ "$status" -eq 0 ] || { echo "hive self_install lost no_build cryptography"; false; }
}
