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

# --- the reconcile: one command for both twins (HARNESS-139, #1628) ---------
#
# The reconcile used to be a loop in each setup script that only installed.
# It is now `dotf pi packages apply`, which converges both ways and is tested
# in Go (cli/internal/pi, cli/internal/cmd/pi_test.go). What is left to pin
# here is that both twins call it, and that neither grew the loop back.

@test "pi packages: both twins reconcile through dotf pi packages apply, naming their own checkout" {
    # --repo, never the cwd: from inside another checkout (a worktree on
    # another branch), the command would reconcile pi against THAT manifest and
    # remove whatever it does not declare. WIN-014 (#1751) is the same defect
    # in `dotf harness mirror`.
    grep -q 'pi packages apply --repo "\$CURRENT_DIR"' "$SETUP_SH"
    grep -q 'pi packages apply --repo \$DotfilesDir' "$SETUP_PS1"
}

@test "pi packages: setup-linux hands the command \$PI_BIN, not the shell function" {
    # `pi` on this machine is a shell function wrapping `dotf secrets run`, so it
    # FAILS on a locked vault. The command resolves a binary, and setup passes
    # the one it installed.
    grep -q 'pi packages apply --repo "\$CURRENT_DIR" --pi "\$PI_BIN"' "$SETUP_SH"
}

@test "pi packages: neither twin carries the reconcile loop any more" {
    # ADR-020 section 5: the port replaces the twins' logic, it does not sit
    # beside it. An install-only loop coming back would reintroduce the defect
    # that a package dropped from the manifest is never removed.
    refute_grep 'PI_PACKAGES_SRC' "$SETUP_SH"
    refute_grep '"\$PI_BIN" install' "$SETUP_SH"
    refute_grep 'piPackagesSrc' "$SETUP_PS1"
    refute_grep '& pi install' "$SETUP_PS1"
}

# --- CI-002 (#1478): DOTFILES_SKIP_PI_PACKAGES -------------------------------
#
# The reconcile is 883-2200s of the Windows CI job, measured across four runs of
# the same nine pinned packages, and a PR that cannot change what it does was
# paying all of it. The guard takes it off the PR path. These tests exist
# because every way this can go wrong is SILENT: a guard that skips when it
# should not, a filter that under-matches, or a skip that logs like a success.

@test "pi packages: the command honours DOTFILES_SKIP_PI_PACKAGES, and says what was NOT verified" {
    # The skip moved into the Go command with the loop. The order (skip before
    # any probe) is asserted there, in TestPiPackagesApplySkipIsFirstAndLoud; a
    # line reading "skipped" is indistinguishable from "fine", so the wording is
    # pinned here too.
    CMD="$BATS_TEST_DIRNAME/../cli/internal/cmd/pi.go"
    grep -q 'DOTFILES_SKIP_PI_PACKAGES' "$CMD"
    grep -q 'nothing installed, nothing verified' "$CMD"
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
    # The reconcile is Go now: a change to it alone must still run the job.
    echo "$filter" | grep -qF "cli/internal/pi/**"
    echo "$filter" | grep -qF "cli/internal/cmd/pi.go"
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
