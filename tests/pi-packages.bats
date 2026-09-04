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

# --- CI-002 (#1478): DOTFILES_SKIP_PI_PACKAGES -------------------------------
#
# The reconcile is 883-2200s of the Windows CI job, measured across four runs of
# the same nine pinned packages, and a PR that cannot change what it does was
# paying all of it. The guard takes it off the PR path. These tests exist
# because every way this can go wrong is SILENT: a guard that skips when it
# should not, a filter that under-matches, or a skip that logs like a success.

@test "pi packages: both twins honour DOTFILES_SKIP_PI_PACKAGES" {
    grep -q 'DOTFILES_SKIP_PI_PACKAGES' "$SETUP_SH"
    grep -q 'DOTFILES_SKIP_PI_PACKAGES' "$SETUP_PS1"
}

@test "pi packages: the skip is the FIRST branch, before the pi/npm probes" {
    # Order is the contract, not style. The runner sets the variable precisely
    # so the expensive block never starts; if the guard sat after the `pi` and
    # `npm` probes, a runner that has both would run the probes' side effects
    # and only then decide to skip.
    # Anchor on the BRANCH, never on a mention. `grep -n | grep -v '^ *#'` does
    # not filter comments here — grep -n prefixes a line number, so the line
    # starts with a digit and the comment pattern never matches. That version
    # of this test matched the explanatory comment above the block, which sits
    # before everything, so it passed no matter where the guard actually was.
    # Caught by mutation: moving the guard below the $PI_BIN probe did not fail
    # it. Same genus as every other green-for-the-wrong-reason in this repo.
    # Sliced from the RECONCILE BLOCK, not from the file. `[ ! -x "$PI_BIN" ]`
    # also appears in pi's own install block 150 lines earlier, so a file-wide
    # search finds that one and compares against the wrong branch.
    sh_block=$(awk '/^PI_PACKAGES_SRC=/{f=1} f' "$SETUP_SH")
    sh_skip=$(printf '%s\n' "$sh_block" | grep -n 'DOTFILES_SKIP_PI_PACKAGES:-' | head -1 | cut -d: -f1)
    sh_pi=$(printf '%s\n' "$sh_block" | grep -n '\[ ! -x "\$PI_BIN" \]' | head -1 | cut -d: -f1)
    [ -n "$sh_skip" ]
    [ -n "$sh_pi" ]
    [ "$sh_skip" -lt "$sh_pi" ]

    ps_block=$(awk '/^\$piPackagesSrc = Join-Path/{f=1} f' "$SETUP_PS1")
    ps_skip=$(printf '%s\n' "$ps_block" | grep -n 'env:DOTFILES_SKIP_PI_PACKAGES' | head -1 | cut -d: -f1)
    ps_pi=$(printf '%s\n' "$ps_block" | grep -n 'Get-Command pi -ErrorAction SilentlyContinue' | head -1 | cut -d: -f1)
    [ -n "$ps_skip" ]
    [ -n "$ps_pi" ]
    [ "$ps_skip" -lt "$ps_pi" ]
}

@test "pi packages: the skip says what was NOT verified, not merely that it skipped" {
    # A line reading "skipped" is indistinguishable from "fine" when someone
    # scans a green log. Both twins must name the consequence.
    grep -q 'nothing installed, nothing verified' "$SETUP_SH"
    grep -q 'nothing installed, nothing verified' "$SETUP_PS1"
}

@test "pi packages: CI never sets the skip on a push to the default branch" {
    # The guard's entire safety property. If this expression ever evaluates
    # truthy on `push`, the reconcile runs NOWHERE and nothing says so.
    CI="$BATS_TEST_DIRNAME/../.github/workflows/ci.yml"
    run grep -n "DOTFILES_SKIP_PI_PACKAGES:" "$CI"
    [ "$status" -eq 0 ]
    echo "$output" | grep -q "github.event_name == 'pull_request'"
}

@test "pi packages: the CI pi filter covers the manifest and BOTH twins" {
    # A filter that misses one of these skips the reconcile on the exact PR
    # that needed it, and the PR goes green.
    CI="$BATS_TEST_DIRNAME/../.github/workflows/ci.yml"
    # The slice STOPS AT THE FIRST NON-ENTRY, and that is load-bearing. The
    # obvious terminator -- "stop at the next 12-space key" -- does not fire
    # here, so the slice ran past the block and swept up unrelated lines that
    # mention these very paths; deleting an entry then still matched. Two
    # earlier versions of this test were green for that reason and for the
    # comment-prose variant of it. Anchor on the LIST, never on what follows it.
    filter=$(awk '/^            pi:$/{f=1;next} f && !/^ *- /{exit} f' "$CI")
    [ -n "$filter" ]
    echo "$filter" | grep -q "ai/pi/\*\*"
    echo "$filter" | grep -q "setup-linux.sh"
    echo "$filter" | grep -q "setup-windows.ps1"
}

@test "pi packages: the CI pi filter covers wherever PI_VERSION is DECLARED" {
    # DERIVED, not enumerated, and that distinction is the point. The test above
    # greps for the three entries someone already wrote, so it is green by
    # construction against any omission — an assertion that cannot fail for the
    # reason it exists. It was, and `versions.conf` was missing: PI_VERSION
    # selects the pi binary that performs the reconcile
    # (setup-linux.sh:722, setup-windows.ps1:1148), so a version bump ran
    # test-windows with the reconcile skipped. Caught in review on #1482, not by
    # this suite.
    #
    # This one asks the REPOSITORY where PI_VERSION lives and requires the
    # answer to be in the filter, so it survives the file being renamed or the
    # pin moving, and it fails if the filter forgets it again.
    CI="$BATS_TEST_DIRNAME/../.github/workflows/ci.yml"
    REPO="$BATS_TEST_DIRNAME/.."
    decl=$(cd "$REPO" && grep -rl '^PI_VERSION=' --include='*.conf' . | head -1)
    [ -n "$decl" ]
    decl=$(basename "$decl")
    # The slice STOPS AT THE FIRST NON-ENTRY, and that is load-bearing. The
    # obvious terminator -- "stop at the next 12-space key" -- does not fire
    # here, so the slice ran past the block and swept up unrelated lines that
    # mention these very paths; deleting an entry then still matched. Two
    # earlier versions of this test were green for that reason and for the
    # comment-prose variant of it. Anchor on the LIST, never on what follows it.
    filter=$(awk '/^            pi:$/{f=1;next} f && !/^ *- /{exit} f' "$CI")
    echo "$filter" | grep -q "$decl"
}
