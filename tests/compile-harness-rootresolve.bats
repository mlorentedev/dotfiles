#!/usr/bin/env bats
# Regression guard: compile-harness.sh must resolve its repo/deploy root from the
# script's own location (not `git rev-parse`), so --check / --deploy work from the
# NON-GIT deploy copy (~/.dotfiles, ADR-012 copy-deploy) — that is where setup and
# healthcheck (section 12) invoke them. Requiring a git repo there false-failed.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="$(mktemp -d)"
}

teardown() { rm -rf "$TMP"; }

@test "compile-harness --check works from a non-git copy (root resolved from SCRIPT_DIR)" {
    # Assemble a complete, NON-git copy of exactly what --check reads. The
    # target list is DERIVED from the manifest, never hand-listed here: this
    # test used to name the targets literally, so #1176 added ai/orca/ORCA.md
    # to BOTH the manifest and this line while setup-linux.sh never learned to
    # mirror it. The test then passed by doing by hand what the deploy did not
    # do, and hid the drift FAIL it exists to catch (#1200).
    mkdir -p "$TMP/scripts"
    cp "$REPO/scripts/compile-harness.sh" "$TMP/scripts/"
    cp -r "$REPO/harness" "$TMP/"
    while IFS= read -r target; do
        [ -n "$target" ] || continue
        mkdir -p "$TMP/$(dirname "$target")"
        cp "$REPO/$target" "$TMP/$target"
    done < <(jq -r '.targets[].file' "$REPO/harness/manifest.json")
    [ ! -d "$TMP/.git" ]   # genuinely not a git repo

    # Run from an unrelated CWD to prove root resolution is CWD-independent.
    run bash -c "cd / && '$TMP/scripts/compile-harness.sh' --check"
    [ "$status" -eq 0 ]
    echo "$output" | grep -q 'no harness drift'
}

@test "every manifest target is a file the repo actually has" {
    # The manifest is the source the deploy copy list derives from, so an entry
    # naming a path that does not exist would silently mirror nothing and leave
    # --check to fail on the absence, one layer removed from the cause.
    while IFS= read -r target; do
        [ -n "$target" ] || continue
        [ -f "$REPO/$target" ] || {
            echo "manifest.json declares a target the repo does not have: $target"
            return 1
        }
    done < <(jq -r '.targets[].file' "$REPO/harness/manifest.json")
}

@test "compile-harness without its harness/ dir fails gracefully (no crash)" {
    mkdir -p "$TMP/scripts"
    cp "$REPO/scripts/compile-harness.sh" "$TMP/scripts/"
    run bash -c "cd / && '$TMP/scripts/compile-harness.sh' --check"
    [ "$status" -ne 0 ]   # missing manifest -> clean non-zero, not a stack trace
}
