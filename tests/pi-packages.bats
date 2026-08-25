#!/usr/bin/env bats
# AI-030 (#1224): the pi package manifest and the reconcile that consumes it.
#
# These assert the DECISIONS, following tests/pi-config.bats. Two of them exist
# because the obvious implementation is wrong in a way that looks right:
#
#   - the array must NOT live in ai/pi/settings.json, which is seed-if-missing,
#     so anything declared there reaches a fresh machine and never this one;
#   - the reconcile must call $PI_BIN, not `pi`, which on Linux is a shell
#     function wrapping `dotf secrets run` and fails on a locked vault.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    MANIFEST="$REPO/ai/pi/packages.json"
    PI_SETTINGS="$REPO/ai/pi/settings.json"
    SETUP_SH="$REPO/setup-linux.sh"
    SETUP_PS1="$REPO/setup-windows.ps1"
}

# --- the manifest ------------------------------------------------------------

@test "pi packages: the manifest exists and is valid JSON with at least one entry" {
    [ -f "$MANIFEST" ]
    run jq -e '.packages | length > 0' "$MANIFEST"
    [ "$status" -eq 0 ]
}

@test "pi packages: every declared source is pinned to a version" {
    # Upstream: "Pi packages run with full system access - extensions execute
    # arbitrary code and skills can instruct the model to run executables."
    # A floating `npm:pkg` hands whoever can publish that name a decision about
    # what runs inside an agent holding NAN_API_KEY. This repository pins its
    # GitHub Actions by commit SHA for the weaker version of the same reason.
    run jq -r '.packages[].source' "$MANIFEST"
    [ "$status" -eq 0 ]
    unpinned=""
    while IFS= read -r src; do
        [ -n "$src" ] || continue
        # npm:<name>@<version>, where <name> may itself be @scope/name.
        printf '%s' "$src" | grep -qE '^npm:@?[a-z0-9._/-]+@[0-9]+\.[0-9]+\.[0-9]+' \
            || unpinned="$unpinned $src"
    done <<< "$output"
    if [ -n "$unpinned" ]; then
        echo "these sources carry no pinned version:$unpinned"
        return 1
    fi
}

@test "pi packages: the pin guard rejects an unpinned source" {
    # The assertion above is worthless if its regex accepts everything. Proven
    # against the exact shape it exists to refuse, rather than trusted (#1203).
    run bash -c "printf '%s' 'npm:pi-effort' | grep -qE '^npm:@?[a-z0-9._/-]+@[0-9]+\.[0-9]+\.[0-9]+'"
    [ "$status" -ne 0 ]
    run bash -c "printf '%s' 'npm:@ayulab/pi-rewind@0.4.6' | grep -qE '^npm:@?[a-z0-9._/-]+@[0-9]+\.[0-9]+\.[0-9]+'"
    [ "$status" -eq 0 ]
}

@test "pi packages: no source is declared twice" {
    total=$(jq -r '.packages | length' "$MANIFEST")
    unique=$(jq -r '[.packages[].source] | unique | length' "$MANIFEST")
    [ "$total" -eq "$unique" ]
}

@test "pi packages: every entry says what it is for" {
    # An entry nobody can justify is one to delete. Nine third-party packages
    # running with full system access is a list that must stay reviewable.
    run jq -e '[.packages[] | select((.why // "") == "")] | length == 0' "$MANIFEST"
    [ "$status" -eq 0 ]
}

# --- the placement decision --------------------------------------------------

@test "pi packages: the array is NOT declared in the seed settings.json" {
    # setup deploys ai/pi/settings.json ONLY when the destination is absent,
    # because pi rewrites it at runtime. Declaring packages there would change
    # what a fresh machine receives and nothing else -- the opposite of the
    # requirement. `pi install` writes the live array; one mechanism, one owner.
    run jq -e 'has("packages")' "$PI_SETTINGS"
    [ "$status" -ne 0 ]
}

@test "pi packages: setup-linux never writes the packages array itself" {
    # If this ever appears, the seed-if-missing contract (#754) is being
    # reintroduced through a side door, and the array would name packages that
    # `pi install` never unpacked to disk.
    refute_grep 'packages.*jq.*>.*settings\.json' "$SETUP_SH"
    refute_grep 'jq .*\.packages.*settings\.json.*>' "$SETUP_SH"
}

# --- the reconcile: Linux ----------------------------------------------------

@test "pi packages: setup-linux reconciles the manifest" {
    grep -q 'ai/pi/packages.json' "$SETUP_SH"
    grep -q 'PI_PACKAGES_SRC' "$SETUP_SH"
}

@test "pi packages: setup-linux installs through \$PI_BIN, not the shell function" {
    # `pi` on this machine is a shell function wrapping `dotf secrets run`, so
    # it resolves a Bitwarden item before exec and FAILS on a locked vault.
    # Setup must not require an unlocked vault to install an extension. The
    # existing version check reaches for $PI_BIN for exactly this reason.
    grep -q '"\$PI_BIN" install' "$SETUP_SH"
}

@test "pi packages: setup-linux compares whole entries, not substrings" {
    # `grep -Fxq`: a fixed-string, full-line match. Without -x, a declared
    # `npm:pi-memory@0.4.2` would be considered present because the live array
    # holds `npm:pi-memory-extra@1.0.0`.
    grep -q 'grep -Fxq' "$SETUP_SH"
}

@test "pi packages: setup-linux degrades to a warning when pi is absent" {
    # A bootstrap that aborts because an optional extension host is missing is
    # worse than one that reports it. Same posture as the npm guard above it.
    grep -q 'skipping pi package reconcile' "$SETUP_SH"
}

@test "pi packages: both setups guard on npm, not only on pi" {
    # `pi install` shells out to npm. Guarding only on pi means a missing Node
    # toolchain is reported N times as N package failures instead of once as its
    # cause -- the "symptom three layers from the cause" shape this repository
    # keeps paying for. Raised by the PR reviewer on #1226 against the
    # acceptance criterion that asked for both guards.
    grep -q 'npm not found — skipping pi package reconcile' "$SETUP_SH"
    grep -q 'npm not available - skipping pi package reconcile' "$SETUP_PS1"
}

@test "pi packages: setup-linux refuses an unreadable manifest instead of reading it empty" {
    # An empty want-list installs nothing and logs exactly like "everything is
    # already present". `jq -e` makes a malformed manifest loud.
    grep -q "jq -er '.packages\[\].source'" "$SETUP_SH"
    grep -q 'declares no readable packages' "$SETUP_SH"
}

# --- the reconcile: Windows parity -------------------------------------------

@test "pi packages: setup-windows reconciles the same manifest" {
    grep -q "ai\\\\pi\\\\packages.json" "$SETUP_PS1"
    grep -q 'piPackagesSrc' "$SETUP_PS1"
}

@test "pi packages: setup-windows installs through pi install" {
    grep -q '& pi install' "$SETUP_PS1"
}

@test "pi packages: setup-windows handles both the string and object entry forms" {
    # Upstream allows `"npm:pkg"` and `{ "source": "npm:pkg", ... }`. A reader
    # that handled only strings would reinstall every filtered entry on every
    # single setup run, which is the opposite of idempotent.
    grep -q 'is \[string\]' "$SETUP_PS1"
    grep -q 'if type == "object" then (.source // empty) else . end' "$SETUP_SH"
}

@test "pi packages: setup-windows degrades to a warning when pi is absent" {
    grep -q 'skipping pi package reconcile' "$SETUP_PS1"
}
