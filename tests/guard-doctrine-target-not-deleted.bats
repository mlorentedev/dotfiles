#!/usr/bin/env bats
# GUARD (OPS-040): no setup script may delete a file the harness manifest
# declares as a doctrine deploy target.
#
# WHAT THIS CAUGHT
# ----------------
# `harness/manifest.json` .doctrine.deploy declares `.gemini/GEMINI.md` as agy's
# rules file, and records why the injection is append-and-replace-in-place:
# Antigravity reads global rules from it, the file is shared with Gemini CLI, and
# a user's own rules around the generated region are meant to survive a deploy.
# `tests/compile-harness.bats` asserts exactly that survival.
#
# Both setup scripts nevertheless carried an SDD-007 block that `rm -f`'d that
# same path on every run, describing it as "a legacy orphan pointing to the
# retired binary". The description was wrong — the manifest says agy reads it —
# and the two behaviours were in direct conflict. Only ordering hid it: setup
# deleted the file early and `compile-harness.sh --deploy` rewrote it near the
# end, so the loss was invisible unless a run stopped in between, or the deploy
# was skipped, or anything moved. Measured 2026-09-02 on msi: 12029 bytes of
# generated doctrine present at the path setup deletes.
#
# The pairing is the bug. A file cannot be both a deploy target and a thing
# setup removes, and the append-in-place contract exists precisely so that user
# content in it is not destroyed.
#
# WHY IT READS THE MANIFEST
# -------------------------
# Hardcoding GEMINI.md would only re-catch this one instance. The manifest is
# the SSOT for which files carry doctrine, so a target added later is covered
# the day it is declared, with no edit here.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    load 'lib/refute'
}

# A removal naming the target's basename anywhere in a setup script. Deliberately
# broader than the exact path: the scripts reach these files through several
# spellings ($GEMINI_HOME, $HOME/.gemini, Join-Path $GeminiHome), and a guard
# that only knew one of them would be bypassed by the next one. Basenames here
# are distinctive enough that a match is a real hit, not a coincidence.
REMOVAL_ERE='(rm[[:space:]]+-[rf]+|Remove-Item)[^|;]*'

@test "guard: no setup script deletes a harness doctrine deploy target" {
    local manifest="$DOTFILES_DIR/harness/manifest.json"
    [ -f "$manifest" ] || skip "harness/manifest.json not found — cannot resolve doctrine targets"
    command -v jq >/dev/null 2>&1 || skip "jq not available — cannot read the manifest"

    local targets
    targets="$(jq -r '.doctrine.deploy[]?.file // empty' "$manifest")"

    # C15: a check that cannot answer must say so, not pass. An empty target list
    # would make every assertion below vacuous, and a vacuous pass here is
    # indistinguishable from "no setup script deletes a doctrine target".
    if [ -z "$targets" ]; then
        skip "manifest declares no .doctrine.deploy targets — nothing to assert"
    fi

    local file base script
    while IFS= read -r file; do
        [ -n "$file" ] || continue
        base="${file##*/}"
        for script in setup-linux.sh setup-windows.ps1; do
            [ -f "$DOTFILES_DIR/$script" ] || continue
            refute_grep "${REMOVAL_ERE}${base}" "$DOTFILES_DIR/$script"
        done
    done <<EOF
$targets
EOF
}

# The guard above is only worth its lines if the manifest reaches it. This pins
# that the query resolves to real targets on the shipped manifest, so a schema
# move (.doctrine.deploy renamed, say) surfaces as a red test rather than as the
# guard quietly skipping forever.
@test "guard: the shipped manifest resolves at least one doctrine deploy target" {
    local manifest="$DOTFILES_DIR/harness/manifest.json"
    [ -f "$manifest" ] || skip "harness/manifest.json not found"
    command -v jq >/dev/null 2>&1 || skip "jq not available"

    run jq -r '[.doctrine.deploy[]?.file] | length' "$manifest"
    [ "$status" -eq 0 ]
    [ "$output" -ge 1 ]
}
